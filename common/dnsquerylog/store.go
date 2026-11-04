package dnsquerylog

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service/filemanager"
)

const (
	defaultPath            = "dns_query_log"
	defaultRetention       = 24 * time.Hour
	defaultMaxSize         = 64 * 1024 * 1024
	defaultMemorySize      = 1000
	defaultFlushInterval   = 30 * time.Second
	segmentDuration        = time.Hour
	minSegmentSize         = 256 * 1024
	subscriptionBufferSize = 1024
	maxQueryLimit          = 1000
	defaultQueryLimit      = 100
)

var _ adapter.LifecycleService = (*Store)(nil)

// Store keeps DNS query entries in memory and, unless memory only, in
// append-only segment files under path. Old segments are removed once they
// are older than the retention or the total size exceeds the limit.
type Store struct {
	ctx           context.Context
	logger        log.ContextLogger
	path          string
	memoryOnly    bool
	retention     time.Duration
	maxSize       int64
	memorySize    int
	flushInterval time.Duration
	segmentSize   int64
	// servers is set by SetServers before start and read only after.
	servers map[string]serverInfo

	idCounter atomic.Uint64
	started   atomic.Bool

	access        sync.Mutex
	ring          []Entry
	ringHead      int
	ringCount     int
	pending       []Entry
	subscriptions map[*Subscription]struct{}

	fileAccess sync.Mutex
	segments   []*segment
	current    *os.File

	flushSignal chan struct{}
	closing     chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

// New creates a store, or returns nil when the query log is disabled.
func New(ctx context.Context, logger log.ContextLogger, options *option.DNSQueryLogOptions) *Store {
	if options == nil || !options.Enabled {
		return nil
	}
	path := options.Path
	if path == "" {
		path = defaultPath
	}
	retention := time.Duration(options.Retention)
	if retention <= 0 {
		retention = defaultRetention
	}
	maxSize := int64(options.MaxSize.Value())
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	memorySize := options.MemorySize
	if memorySize <= 0 {
		memorySize = defaultMemorySize
	}
	flushInterval := time.Duration(options.FlushInterval)
	if flushInterval <= 0 {
		flushInterval = defaultFlushInterval
	}
	s := &Store{
		ctx:           ctx,
		logger:        logger,
		path:          filemanager.BasePath(ctx, path),
		memoryOnly:    options.MemoryOnly,
		retention:     retention,
		maxSize:       maxSize,
		memorySize:    memorySize,
		flushInterval: flushInterval,
		segmentSize:   max(maxSize/8, minSegmentSize),
		ring:          make([]Entry, memorySize),
		subscriptions: make(map[*Subscription]struct{}),
		flushSignal:   make(chan struct{}, 1),
		closing:       make(chan struct{}),
		closed:        make(chan struct{}),
	}
	return s
}

func (s *Store) Name() string {
	return "dns-query-log"
}

func (s *Store) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	if stage != adapter.StartStateInitialize {
		return nil
	}
	if !s.memoryOnly {
		err := s.openSegments()
		if err != nil {
			return err
		}
		go s.loopFlush()
	} else {
		close(s.closed)
	}
	s.started.Store(true)
	activeStores.Add(1)
	return nil
}

func (s *Store) Close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.started.Swap(false) {
			activeStores.Add(-1)
			close(s.closing)
			<-s.closed
		}
		s.access.Lock()
		for subscription := range s.subscriptions {
			subscription.closeLocked()
		}
		s.subscriptions = nil
		s.access.Unlock()
		if !s.memoryOnly {
			err = s.flush()
			s.fileAccess.Lock()
			if s.current != nil {
				closeErr := s.current.Close()
				if err == nil {
					err = closeErr
				}
				s.current = nil
			}
			s.fileAccess.Unlock()
		}
	})
	return err
}

func (s *Store) nextID() uint64 {
	return s.idCounter.Add(1)
}

// WrapRouter returns router with its queries recorded, or router itself
// when the store is nil.
func (s *Store) WrapRouter(router adapter.DNSRouter) adapter.DNSRouter {
	if s == nil {
		return router
	}
	return &Router{DNSRouter: router, store: s}
}

func (s *Store) add(entry Entry) {
	s.access.Lock()
	entry.ID = s.nextID()
	s.ring[s.ringHead] = entry
	s.ringHead = (s.ringHead + 1) % len(s.ring)
	if s.ringCount < len(s.ring) {
		s.ringCount++
	}
	var signal bool
	if !s.memoryOnly {
		s.pending = append(s.pending, entry)
		signal = len(s.pending) >= s.memorySize
	}
	for subscription := range s.subscriptions {
		select {
		case subscription.entries <- entry:
		default:
			subscription.dropped.Add(1)
		}
	}
	s.access.Unlock()
	if signal {
		select {
		case s.flushSignal <- struct{}{}:
		default:
		}
	}
}

// recentLocked returns the entries in memory, newest first.
func (s *Store) recentLocked() []Entry {
	entries := make([]Entry, 0, s.ringCount)
	for i := 1; i <= s.ringCount; i++ {
		entries = append(entries, s.ring[(s.ringHead-i+len(s.ring))%len(s.ring)])
	}
	return entries
}

// Query returns entries matching filter, newest first. Entries with an ID at
// or after before are skipped when before is not zero. next is the cursor for
// the following page, or zero when there are no more entries.
func (s *Store) Query(filter Filter, before uint64, limit int) (entries []Entry, next uint64, err error) {
	if limit <= 0 {
		limit = defaultQueryLimit
	} else if limit > maxQueryLimit {
		limit = maxQueryLimit
	}
	err = s.walk(filter, before, func(entry Entry) bool {
		entries = append(entries, entry)
		return len(entries) <= limit
	})
	if len(entries) > limit {
		entries = entries[:limit]
		next = entries[limit-1].ID
	}
	return entries, next, err
}

// walk visits the entries matching filter, newest first, until visit returns
// false. Entries with an ID at or after before are skipped when before is not
// zero.
func (s *Store) walk(filter Filter, before uint64, visit func(entry Entry) bool) error {
	filter.Prepare()
	match := func(entry Entry) bool {
		if before != 0 && entry.ID >= before {
			return true
		}
		if !filter.Match(&entry) {
			return true
		}
		return visit(entry)
	}
	if !s.memoryOnly {
		return s.queryFiles(filter, before, match)
	}
	s.access.Lock()
	recent := s.recentLocked()
	s.access.Unlock()
	for _, entry := range recent {
		if !match(entry) {
			break
		}
	}
	return nil
}

// Clear removes all entries.
func (s *Store) Clear() error {
	s.fileAccess.Lock()
	defer s.fileAccess.Unlock()
	s.access.Lock()
	s.ring = make([]Entry, len(s.ring))
	s.ringHead = 0
	s.ringCount = 0
	s.pending = nil
	s.access.Unlock()
	if s.memoryOnly {
		return nil
	}
	return s.removeAllSegmentsLocked()
}

type Status struct {
	Persistent    bool
	Path          string
	Retention     time.Duration
	MaxSize       int64
	DiskUsage     int64
	Segments      int
	MemoryEntries int
	Pending       int
}

func (s *Store) Status() Status {
	status := Status{
		Persistent: !s.memoryOnly,
		Retention:  s.retention,
		MaxSize:    s.maxSize,
	}
	if !s.memoryOnly {
		status.Path = s.path
		s.fileAccess.Lock()
		status.Segments = len(s.segments)
		for _, item := range s.segments {
			status.DiskUsage += item.size
		}
		s.fileAccess.Unlock()
	}
	s.access.Lock()
	status.MemoryEntries = s.ringCount
	status.Pending = len(s.pending)
	s.access.Unlock()
	return status
}

// Subscription receives new entries until closed. Entries are dropped when
// the subscriber falls behind.
type Subscription struct {
	store   *Store
	entries chan Entry
	dropped atomic.Uint64
	closed  bool
}

func (s *Store) Subscribe() *Subscription {
	subscription := &Subscription{
		store:   s,
		entries: make(chan Entry, subscriptionBufferSize),
	}
	s.access.Lock()
	if s.subscriptions == nil {
		subscription.closed = true
		close(subscription.entries)
	} else {
		s.subscriptions[subscription] = struct{}{}
	}
	s.access.Unlock()
	return subscription
}

// Entries is closed when the subscription or the store is closed.
func (s *Subscription) Entries() <-chan Entry {
	return s.entries
}

// Dropped returns the number of entries dropped so far.
func (s *Subscription) Dropped() uint64 {
	return s.dropped.Load()
}

func (s *Subscription) Close() {
	s.store.access.Lock()
	defer s.store.access.Unlock()
	if s.store.subscriptions != nil {
		delete(s.store.subscriptions, s)
	}
	s.closeLocked()
}

func (s *Subscription) closeLocked() {
	if s.closed {
		return
	}
	s.closed = true
	close(s.entries)
}
