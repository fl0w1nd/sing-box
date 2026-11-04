package dnsquerylog

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/byteformats"
	"github.com/sagernet/sing/common/json/badoption"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T, options option.DNSQueryLogOptions) *Store {
	t.Helper()
	options.Enabled = true
	if !options.MemoryOnly && options.Path == "" {
		options.Path = t.TempDir()
	}
	store := New(context.Background(), log.NewNOPFactory().NewLogger("test"), &options)
	require.NoError(t, store.Start(adapter.StartStateInitialize, nil))
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func addEntries(store *Store, count int, prefix string) {
	for i := range count {
		store.add(Entry{
			Time:      time.Now().UnixMilli(),
			Method:    MethodExchange,
			Domain:    fmt.Sprintf("%s-%d.test", prefix, i),
			QueryType: "A",
			Status:    StatusUpstream,
			Upstreams: []Upstream{{Server: "remote", QueryType: "A"}},
			Answers:   []string{"A 1.2.3.4"},
		})
	}
}

func queryAll(t *testing.T, store *Store, filter Filter) []Entry {
	t.Helper()
	var all []Entry
	var before uint64
	for {
		entries, next, err := store.Query(context.Background(), filter, before, 7)
		require.NoError(t, err)
		all = append(all, entries...)
		if next == 0 {
			return all
		}
		before = next
	}
}

func TestMemoryStoreKeepsNewestEntries(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true, MemorySize: 10})
	addEntries(store, 25, "memory")
	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 10)
	require.Equal(t, "memory-24.test", entries[0].Domain)
	require.Equal(t, "memory-15.test", entries[9].Domain)
	for i := 1; i < len(entries); i++ {
		require.Less(t, entries[i].ID, entries[i-1].ID)
	}
}

func TestPersistentStorePagesAcrossPendingAndSegments(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemorySize: 1000})
	addEntries(store, 30, "flushed")
	require.NoError(t, store.flush())
	addEntries(store, 5, "pending")
	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 35)
	require.Equal(t, "pending-4.test", entries[0].Domain)
	require.Equal(t, "flushed-0.test", entries[34].Domain)
	seen := make(map[uint64]bool)
	for _, entry := range entries {
		require.False(t, seen[entry.ID], "duplicate entry %d", entry.ID)
		seen[entry.ID] = true
	}
	matched := queryAll(t, store, Filter{Search: "PENDING-"})
	require.Len(t, matched, 5)
}

// A slow query is stored after a later fast one, since entries are stored as
// they finish while Time is when they started.
func TestQuerySinceWithSlowQuery(t *testing.T) {
	base := time.Now().Add(-time.Minute).UnixMilli()
	addOutOfOrder := func(store *Store) {
		store.add(Entry{Time: base + 5000, Domain: "fast.test"})
		store.add(Entry{Time: base, Domain: "slow.test"})
	}
	filter := Filter{Since: base + 4000}
	for _, testCase := range []struct {
		name    string
		options option.DNSQueryLogOptions
		flush   bool
	}{
		{name: "memory", options: option.DNSQueryLogOptions{MemoryOnly: true}},
		{name: "pending", options: option.DNSQueryLogOptions{}},
		{name: "segment", options: option.DNSQueryLogOptions{}, flush: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := newTestStore(t, testCase.options)
			addOutOfOrder(store)
			if testCase.flush {
				require.NoError(t, store.flush())
			}
			entries := queryAll(t, store, filter)
			require.Len(t, entries, 1)
			require.Equal(t, "fast.test", entries[0].Domain)
		})
	}
}

func TestPersistentStoreContinuesAfterRestart(t *testing.T) {
	path := t.TempDir()
	first := newTestStore(t, option.DNSQueryLogOptions{Path: path})
	addEntries(first, 3, "first")
	require.NoError(t, first.Close())

	second := newTestStore(t, option.DNSQueryLogOptions{Path: path})
	addEntries(second, 2, "second")
	entries := queryAll(t, second, Filter{})
	require.Len(t, entries, 5)
	require.Equal(t, uint64(5), entries[0].ID)
	require.Equal(t, uint64(1), entries[4].ID)
}

func TestPersistentStoreSkipsIncompleteLines(t *testing.T) {
	path := t.TempDir()
	first := newTestStore(t, option.DNSQueryLogOptions{Path: path})
	addEntries(first, 2, "complete")
	require.NoError(t, first.Close())
	segmentPath := filepath.Join(path, segmentName(1))
	file, err := os.OpenFile(segmentPath, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = file.WriteString(`{"id":3,"domain":"trunc`)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	second := newTestStore(t, option.DNSQueryLogOptions{Path: path})
	entries := queryAll(t, second, Filter{})
	require.Len(t, entries, 2)
}

func TestPersistentStoreRemovesBySize(t *testing.T) {
	maxSize := byteformats.MemoryBytes{}
	require.NoError(t, maxSize.UnmarshalJSON([]byte(`"1MB"`)))
	store := newTestStore(t, option.DNSQueryLogOptions{MaxSize: &maxSize, MemorySize: 100000})
	for range 20 {
		addEntries(store, 1000, "size")
		require.NoError(t, store.flush())
	}
	status := store.Status()
	require.Greater(t, status.Segments, 1)
	require.LessOrEqual(t, status.DiskUsage, int64(1000*1000))
	var diskUsage int64
	dirEntries, err := os.ReadDir(store.path)
	require.NoError(t, err)
	for _, dirEntry := range dirEntries {
		info, infoErr := dirEntry.Info()
		require.NoError(t, infoErr)
		diskUsage += info.Size()
	}
	require.Equal(t, status.DiskUsage, diskUsage)
	entries, _, err := store.Query(context.Background(), Filter{}, 0, 1)
	require.NoError(t, err)
	require.Equal(t, "size-999.test", entries[0].Domain)
}

func TestPersistentStoreRemovesByRetention(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{Retention: badoption.Duration(time.Hour)})
	addEntries(store, 3, "old")
	require.NoError(t, store.flush())
	// Start a new segment, as one does every hour.
	store.fileAccess.Lock()
	require.NoError(t, store.current.Close())
	store.current = nil
	store.fileAccess.Unlock()
	addEntries(store, 2, "new")
	require.NoError(t, store.flush())

	store.fileAccess.Lock()
	require.Len(t, store.segments, 2)
	store.segments[0].modTime = time.Now().Add(-2 * time.Hour)
	store.cleanupLocked(time.Now())
	require.Len(t, store.segments, 1)
	store.fileAccess.Unlock()

	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 2)
	require.Equal(t, "new-1.test", entries[0].Domain)
}

// Entries expire while no query comes in, the current segment too.
func TestPersistentStoreRemovesByRetentionWhenIdle(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{Retention: badoption.Duration(time.Hour)})
	addEntries(store, 3, "old")
	require.NoError(t, store.flush())
	store.fileAccess.Lock()
	require.Len(t, store.segments, 1)
	require.NotNil(t, store.current)
	store.segments[0].modTime = time.Now().Add(-2 * time.Hour)
	store.fileAccess.Unlock()

	require.NoError(t, store.flush())
	require.Empty(t, queryAll(t, store, Filter{}))
	dirEntries, err := os.ReadDir(store.path)
	require.NoError(t, err)
	require.Empty(t, dirEntries)

	addEntries(store, 2, "new")
	require.NoError(t, store.flush())
	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 2)
	require.Equal(t, "new-1.test", entries[0].Domain)
}

// A flush that fails after writing a segment retries only the rest.
func TestPersistentStoreRetriesFailedWrite(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemorySize: 100000})
	addEntries(store, 1, "first")
	require.NoError(t, store.flush())
	store.segmentSize = 4 * 1024
	// Writing to the open segment works, creating the next one fails.
	require.NoError(t, os.Chmod(store.path, 0o500))
	addEntries(store, 100, "batch")
	require.Error(t, store.flush())
	require.NoError(t, os.Chmod(store.path, 0o755))
	require.NoError(t, store.flush())

	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 101)
	for i, entry := range entries {
		require.Equal(t, uint64(101-i), entry.ID)
	}
}

// A write that fails halfway leaves no broken line for the retry to follow.
func TestPersistentStoreDiscardsPartialWrite(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{})
	addEntries(store, 1, "first")
	require.NoError(t, store.flush())
	store.fileAccess.Lock()
	current := store.segments[len(store.segments)-1]
	_, err := store.current.WriteString(`{"id":2,"domain":"brok`)
	require.NoError(t, err)
	store.discardPartialWriteLocked(current, len(`{"id":2,"domain":"brok`))
	store.fileAccess.Unlock()
	addEntries(store, 1, "second")
	require.NoError(t, store.flush())

	second := newTestStore(t, option.DNSQueryLogOptions{Path: store.path})
	entries := queryAll(t, second, Filter{})
	require.Len(t, entries, 2)
	require.Equal(t, "second-0.test", entries[0].Domain)
}

func TestStoreClear(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{})
	addEntries(store, 3, "clear")
	require.NoError(t, store.flush())
	addEntries(store, 2, "pending")
	require.NoError(t, store.Clear())
	require.Empty(t, queryAll(t, store, Filter{}))
	dirEntries, err := os.ReadDir(store.path)
	require.NoError(t, err)
	require.Empty(t, dirEntries)
	addEntries(store, 1, "after")
	require.Len(t, queryAll(t, store, Filter{}), 1)
}

func TestSubscriptionReceivesEntries(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	subscription := store.Subscribe()
	addEntries(store, 2, "live")
	require.Equal(t, "live-0.test", (<-subscription.Entries()).Domain)
	require.Equal(t, "live-1.test", (<-subscription.Entries()).Domain)
	subscription.Close()
	_, loaded := <-subscription.Entries()
	require.False(t, loaded)
}

func TestFilter(t *testing.T) {
	entry := &Entry{
		Time:      1000,
		Domain:    "www.Example.com",
		QueryType: "A,AAAA",
		Status:    StatusCached,
		Inbound:   "dns-in",
		Source:    "192.168.1.20:5353",
		Upstreams: []Upstream{{Server: "remote"}},
		Answers:   []string{"A 93.184.216.34"},
	}
	match := func(filter Filter) bool {
		filter.Prepare()
		return filter.Match(entry)
	}
	require.True(t, match(Filter{}))
	require.True(t, match(Filter{Search: "example.COM"}))
	require.True(t, match(Filter{Search: "192.168.1.20"}))
	require.True(t, match(Filter{Search: "93.184"}))
	require.True(t, match(Filter{QueryType: "aaaa"}))
	require.False(t, match(Filter{QueryType: "HTTPS"}))
	require.True(t, match(Filter{Status: "Cached"}))
	require.False(t, match(Filter{Status: StatusUpstream}))
	require.True(t, match(Filter{Server: "remote"}))
	require.False(t, match(Filter{Server: "local"}))
	require.False(t, match(Filter{Inbound: "tun-in"}))
	require.False(t, match(Filter{Since: 1001}))
	require.False(t, match(Filter{Until: 999}))
}

type testTransport struct {
	adapter.DNSTransport
	tag           string
	transportType string
}

func (t *testTransport) Tag() string  { return t.tag }
func (t *testTransport) Type() string { return t.transportType }

type testRule struct {
	adapter.DNSRule
	action adapter.RuleAction
}

func (r *testRule) String() string             { return "domain_suffix=test" }
func (r *testRule) Action() adapter.RuleAction { return r.action }

type testAction struct {
	actionType string
}

func (a *testAction) Type() string   { return a.actionType }
func (a *testAction) String() string { return a.actionType + "(test)" }

// testRouter answers like the DNS router, calling the hooks the real router
// and client call.
type testRouter struct {
	adapter.DNSRouter
	exchange func(ctx context.Context, message *dns.Msg) (*dns.Msg, error)
}

func (r *testRouter) Exchange(ctx context.Context, message *dns.Msg, options adapter.DNSQueryOptions) (*dns.Msg, error) {
	return r.exchange(ctx, message)
}

func (r *testRouter) Lookup(ctx context.Context, domain string, options adapter.DNSQueryOptions) ([]netip.Addr, error) {
	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	response, err := r.exchange(ctx, message)
	if err != nil {
		return nil, err
	}
	if response.Rcode != dns.RcodeSuccess {
		return nil, rcodeError(response.Rcode)
	}
	return adapter.DNSResponseAddresses(response), nil
}

type rcodeError int

func (e rcodeError) Error() string {
	return dns.RcodeToString[int(e)]
}

func answer(message *dns.Msg, address string) *dns.Msg {
	response := new(dns.Msg)
	response.SetReply(message)
	response.Answer = append(response.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: message.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30},
		A:   netip.MustParseAddr(address).AsSlice(),
	})
	return response
}

func TestRouterRecordsQueries(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	remote := &testTransport{tag: "remote", transportType: C.DNSTypeHTTPS}
	polluted := &testTransport{tag: "local", transportType: C.DNSTypeUDP}
	fakeIP := &testTransport{tag: "fakeip", transportType: C.DNSTypeFakeIP}
	router := store.WrapRouter(&testRouter{exchange: func(ctx context.Context, message *dns.Msg) (*dns.Msg, error) {
		switch message.Question[0].Name {
		case "evaluated.test.":
			Rule(ctx, 0, &testRule{action: &testAction{actionType: C.RuleActionTypeEvaluate}})
			finish := BeginUpstream(adapter.ContextWithDNSTransportTag(ctx, polluted.tag), polluted, message)
			pollutedResponse := answer(message, "0.0.0.1")
			finish(pollutedResponse, nil)
			MarkRejected(adapter.ContextWithDNSTransportTag(ctx, polluted.tag), pollutedResponse)
			finish = BeginUpstream(ctx, remote, message)
			response := answer(message, "1.2.3.4")
			finish(response, nil)
			return response, nil
		case "cached.test.":
			MarkCached(ctx)
			return answer(message, "1.2.3.4"), nil
		case "fake.test.":
			finish := BeginUpstream(ctx, fakeIP, message)
			response := answer(message, "198.18.0.1")
			finish(response, nil)
			return response, nil
		case "local.test.":
			response := new(dns.Msg)
			response.SetReply(message)
			return response, nil
		case "blocked.test.":
			Rule(ctx, 1, &testRule{action: &testAction{actionType: C.RuleActionTypeReject}})
			response := new(dns.Msg)
			response.SetRcode(message, dns.RcodeRefused)
			return response, nil
		case "missing.test.":
			finish := BeginUpstream(ctx, remote, message)
			response := new(dns.Msg)
			response.SetRcode(message, dns.RcodeNameError)
			finish(response, nil)
			return response, nil
		default:
			finish := BeginUpstream(ctx, remote, message)
			err := errors.New("timeout")
			finish(nil, err)
			return nil, err
		}
	}})
	ctx := adapter.WithContext(context.Background(), &adapter.InboundContext{
		Inbound:     "dns-in",
		InboundType: C.TypeDirect,
		Source:      M.ParseSocksaddr("192.168.1.20:53000"),
		ProcessInfo: &adapter.ConnectionOwner{ProcessPaths: []string{"/usr/bin/curl"}},
	})
	exchange := func(domain string) {
		message := new(dns.Msg)
		message.SetQuestion(dns.Fqdn(domain), dns.TypeA)
		_, _ = router.Exchange(ctx, message, adapter.DNSQueryOptions{})
	}
	exchange("evaluated.test")
	exchange("cached.test")
	exchange("fake.test")
	exchange("blocked.test")
	exchange("local.test")
	exchange("missing.test")
	exchange("failed.test")
	_, err := router.Lookup(ctx, "missing.test", adapter.DNSQueryOptions{Strategy: C.DomainStrategyIPv4Only})
	require.Error(t, err)

	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 8)
	byDomain := make(map[string][]Entry)
	for _, entry := range entries {
		byDomain[entry.Domain] = append(byDomain[entry.Domain], entry)
	}

	evaluated := byDomain["evaluated.test"][0]
	require.Equal(t, StatusUpstream, evaluated.Status)
	require.Equal(t, "dns-in", evaluated.Inbound)
	require.Equal(t, "192.168.1.20:53000", evaluated.Source)
	require.Equal(t, "/usr/bin/curl", evaluated.Process)
	require.True(t, evaluated.Rejected)
	require.Len(t, evaluated.Upstreams, 2)
	require.True(t, evaluated.Upstreams[0].Rejected)
	require.Equal(t, []string{"A 0.0.0.1"}, evaluated.Upstreams[0].Answers)
	require.False(t, evaluated.Upstreams[1].Rejected)
	require.Equal(t, []string{"A 1.2.3.4"}, evaluated.Answers)
	require.Equal(t, uint32(30), evaluated.TTL)
	require.Equal(t, C.RuleActionTypeEvaluate, evaluated.Rules[0].ActionType)

	require.Equal(t, StatusCached, byDomain["cached.test"][0].Status)
	require.Equal(t, StatusFakeIP, byDomain["fake.test"][0].Status)
	require.Equal(t, StatusRejected, byDomain["blocked.test"][0].Status)
	require.Equal(t, StatusLocal, byDomain["local.test"][0].Status)
	require.Equal(t, "REFUSED", byDomain["blocked.test"][0].Rcode)
	require.Equal(t, StatusError, byDomain["failed.test"][0].Status)
	require.Equal(t, "timeout", byDomain["failed.test"][0].Error)
	require.Equal(t, "timeout", byDomain["failed.test"][0].Upstreams[0].Error)

	var missingExchange, missingLookup Entry
	for _, entry := range byDomain["missing.test"] {
		if entry.Method == MethodLookup {
			missingLookup = entry
		} else {
			missingExchange = entry
		}
	}
	require.Equal(t, StatusUpstream, missingExchange.Status)
	require.Equal(t, "NXDOMAIN", missingExchange.Rcode)
	require.Equal(t, StatusError, missingLookup.Status)
	require.Equal(t, "NXDOMAIN", missingLookup.Rcode)
	require.Equal(t, "A", missingLookup.QueryType)
}

func TestBackgroundExchangeIsRecorded(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	ctx := service.ContextWithPtr(context.Background(), store)
	remote := &testTransport{tag: "remote", transportType: C.DNSTypeUDP}
	message := new(dns.Msg)
	message.SetQuestion("refresh.test.", dns.TypeAAAA)
	finish := BeginUpstream(ctx, remote, message)
	finish(answer(message, "1.2.3.4"), nil)
	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 1)
	require.Equal(t, MethodBackground, entries[0].Method)
	require.Equal(t, "AAAA", entries[0].QueryType)
	require.Equal(t, StatusUpstream, entries[0].Status)
}

// The A and AAAA exchanges of a lookup run at the same time, so a rejection
// marks the exchange of its own type.
func TestMarkRejectedMatchesQueryType(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	remote := &testTransport{tag: "remote", transportType: C.DNSTypeUDP}
	ctx, rec := store.beginRecord(context.Background(), MethodLookup, "both.test.", "")
	ctx = adapter.ContextWithDNSTransportTag(ctx, remote.tag)
	messageA := new(dns.Msg)
	messageA.SetQuestion("both.test.", dns.TypeA)
	messageAAAA := new(dns.Msg)
	messageAAAA.SetQuestion("both.test.", dns.TypeAAAA)
	finishA := BeginUpstream(ctx, remote, messageA)
	finishAAAA := BeginUpstream(ctx, remote, messageAAAA)
	responseA := answer(messageA, "1.2.3.4")
	finishA(responseA, nil)
	finishAAAA(answer(messageAAAA, "1.2.3.4"), nil)
	MarkRejected(ctx, responseA)
	rec.finish(nil, nil, nil)
	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 1)
	require.Equal(t, "A", entries[0].Upstreams[0].QueryType)
	require.True(t, entries[0].Upstreams[0].Rejected)
	require.False(t, entries[0].Upstreams[1].Rejected)
}

// A background query keeps its entry open until its response is checked.
func TestBackgroundQueryRecordsRejection(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	remote := &testTransport{tag: "remote", transportType: C.DNSTypeUDP}
	ctx := adapter.ContextWithDNSTransportTag(service.ContextWithPtr(context.Background(), store), remote.tag)
	message := new(dns.Msg)
	message.SetQuestion("refresh.test.", dns.TypeA)
	ctx, finishBackground := BeginBackground(ctx, message)
	response := answer(message, "0.0.0.1")
	BeginUpstream(ctx, remote, message)(response, nil)
	require.Empty(t, queryAll(t, store, Filter{}))
	MarkRejected(ctx, response)
	finishBackground(response, nil)
	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 1)
	require.Equal(t, MethodBackground, entries[0].Method)
	require.Equal(t, StatusError, entries[0].Status)
	require.Equal(t, "response rejected", entries[0].Error)
	require.Len(t, entries[0].Upstreams, 1)
	require.True(t, entries[0].Upstreams[0].Rejected)
}

func TestHooksWithoutStore(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("idle.test.", dns.TypeA)
	ctx := context.Background()
	BeginUpstream(ctx, &testTransport{tag: "remote"}, message)(nil, nil)
	MarkCached(ctx)
	MarkOptimistic(ctx)
	MarkRejected(ctx, nil)
	MarkStrategyRejected(ctx)
	_, finishBackground := BeginBackground(ctx, message)
	finishBackground(nil, nil)
	Rule(ctx, 0, &testRule{action: &testAction{actionType: C.RuleActionTypeRoute}})
	var store *Store
	router := &testRouter{}
	require.Same(t, adapter.DNSRouter(router), store.WrapRouter(router))
}
