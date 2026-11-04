package dnsquerylog

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/service"

	"github.com/miekg/dns"
)

// activeStores lets the hooks return at once when no store is running.
var activeStores atomic.Int32

type recordContextKey struct{}

type record struct {
	store  *Store
	access sync.Mutex
	start  time.Time
	entry  Entry
	done   bool
	// strategyRejected is kept out of Entry: it only decides the status.
	strategyRejected bool
}

func recordFromContext(ctx context.Context) *record {
	if activeStores.Load() == 0 {
		return nil
	}
	rec, _ := ctx.Value(recordContextKey{}).(*record)
	return rec
}

func (s *Store) beginRecord(ctx context.Context, method string, domain string, queryType string) (context.Context, *record) {
	rec := &record{
		store: s,
		start: time.Now(),
		entry: Entry{
			Method:    method,
			Domain:    strings.TrimSuffix(domain, "."),
			QueryType: queryType,
		},
	}
	rec.entry.Time = rec.start.UnixMilli()
	if metadata := adapter.ContextFrom(ctx); metadata != nil {
		rec.entry.Inbound = metadata.Inbound
		rec.entry.InboundType = metadata.InboundType
		if metadata.Source.IsValid() {
			rec.entry.Source = metadata.Source.String()
		}
		rec.entry.User = metadata.User
		rec.entry.Outbound = metadata.Outbound
		if method == MethodLookup && metadata.Destination.IsValid() {
			rec.entry.Destination = metadata.Destination.String()
		}
		if processInfo := metadata.ProcessInfo; processInfo != nil {
			if len(processInfo.PackageNames) > 0 {
				rec.entry.Process = processInfo.PackageNames[0]
			} else if len(processInfo.ProcessPaths) > 0 {
				rec.entry.Process = processInfo.ProcessPaths[0]
			}
		}
	}
	return context.WithValue(ctx, recordContextKey{}, rec), rec
}

func (r *record) finish(response *dns.Msg, addresses []netip.Addr, err error) {
	r.access.Lock()
	if r.done {
		r.access.Unlock()
		return
	}
	r.done = true
	entry := r.entry
	entry.Rules = append([]RuleMatch(nil), r.entry.Rules...)
	entry.Upstreams = append([]Upstream(nil), r.entry.Upstreams...)
	strategyRejected := r.strategyRejected
	r.access.Unlock()
	entry.Duration = time.Since(r.start).Microseconds()
	if response != nil {
		entry.Rcode = dns.RcodeToString[response.Rcode]
		entry.Answers, entry.TTL = formatAnswers(response.Answer)
	}
	if addresses != nil {
		entry.Answers = make([]string, 0, len(addresses))
		for _, address := range addresses {
			entry.Answers = append(entry.Answers, address.String())
		}
		entry.Rcode = dns.RcodeToString[dns.RcodeSuccess]
	}
	if entry.QueryType == "" {
		entry.QueryType = upstreamQueryTypes(entry.Upstreams, addresses)
	}
	if err != nil {
		entry.Error = err.Error()
		if entry.Rcode == "" {
			entry.Rcode = rcodeFromError(err)
		}
	}
	entry.Status = computeStatus(&entry, strategyRejected, err)
	r.store.add(entry)
}

func computeStatus(entry *Entry, strategyRejected bool, err error) string {
	// A lookup queries A and AAAA apart, so a rule that answered one of them,
	// such as one answering every AAAA query, does not decide the status of a
	// lookup that got addresses from the other.
	if last := len(entry.Rules) - 1; last >= 0 && !(entry.Method == MethodLookup && len(entry.Answers) > 0) {
		switch entry.Rules[last].ActionType {
		case C.RuleActionTypeReject:
			return StatusRejected
		case C.RuleActionTypePredefined:
			return StatusPredefined
		}
	}
	if err != nil {
		return StatusError
	}
	var exchanged bool
	for _, upstream := range entry.Upstreams {
		if upstream.Error != "" || upstream.Rejected {
			continue
		}
		if upstream.Type == C.DNSTypeFakeIP {
			return StatusFakeIP
		}
		exchanged = true
	}
	switch {
	case exchanged:
		return StatusUpstream
	case entry.Optimistic:
		return StatusOptimistic
	case entry.Cached:
		return StatusCached
	case strategyRejected:
		return StatusStrategy
	default:
		return StatusLocal
	}
}

// rcodeFromError finds the rcode of a failed lookup, whose error text is the
// rcode name.
func rcodeFromError(err error) string {
	if _, isRcode := dns.StringToRcode[err.Error()]; isRcode {
		return err.Error()
	}
	switch unwrapErr := err.(type) {
	case interface{ Unwrap() error }:
		if inner := unwrapErr.Unwrap(); inner != nil {
			return rcodeFromError(inner)
		}
	case interface{ Unwrap() []error }:
		for _, inner := range unwrapErr.Unwrap() {
			if rcode := rcodeFromError(inner); rcode != "" {
				return rcode
			}
		}
	}
	return ""
}

// upstreamQueryTypes finds the query types of a lookup from its exchanges,
// or from the addresses it got when all were answered from the cache.
func upstreamQueryTypes(upstreams []Upstream, addresses []netip.Addr) string {
	var types []string
	addType := func(queryType string) {
		if queryType != "" && !slices.Contains(types, queryType) {
			types = append(types, queryType)
		}
	}
	for _, upstream := range upstreams {
		addType(upstream.QueryType)
	}
	if len(types) == 0 {
		for _, address := range addresses {
			if address.Is4() {
				addType("A")
			} else {
				addType("AAAA")
			}
		}
	}
	slices.Sort(types)
	return strings.Join(types, ",")
}

func formatAnswers(records []dns.RR) ([]string, uint32) {
	if len(records) == 0 {
		return nil, 0
	}
	answers := make([]string, 0, len(records))
	var ttl uint32
	for i, rr := range records {
		header := rr.Header()
		if i == 0 || header.Ttl < ttl {
			ttl = header.Ttl
		}
		data := strings.TrimSpace(strings.TrimPrefix(rr.String(), header.String()))
		answers = append(answers, dns.Type(header.Rrtype).String()+" "+data)
	}
	return answers, ttl
}

// Rule records a matched DNS rule of the query in ctx.
func Rule(ctx context.Context, index int, rule adapter.DNSRule) {
	rec := recordFromContext(ctx)
	if rec == nil {
		return
	}
	action := rule.Action()
	match := RuleMatch{
		Index:      index,
		Rule:       rule.String(),
		Action:     action.String(),
		ActionType: action.Type(),
	}
	rec.access.Lock()
	rec.entry.Rules = append(rec.entry.Rules, match)
	rec.access.Unlock()
}

// MarkCached records that the query in ctx was answered from the cache.
func MarkCached(ctx context.Context) {
	rec := recordFromContext(ctx)
	if rec == nil {
		return
	}
	rec.access.Lock()
	rec.entry.Cached = true
	rec.access.Unlock()
}

// MarkOptimistic records that the query in ctx was answered from an expired
// cache entry while it is refreshed in the background.
func MarkOptimistic(ctx context.Context) {
	rec := recordFromContext(ctx)
	if rec == nil {
		return
	}
	rec.access.Lock()
	rec.entry.Cached = true
	rec.entry.Optimistic = true
	rec.access.Unlock()
}

// MarkStrategyRejected records that the DNS client answered the query in ctx
// with an empty response because the domain strategy excludes its type.
func MarkStrategyRejected(ctx context.Context) {
	rec := recordFromContext(ctx)
	if rec == nil {
		return
	}
	rec.access.Lock()
	rec.strategyRejected = true
	rec.access.Unlock()
}

// MarkRejected records that response of the query in ctx was rejected. The
// last exchange of its type with the server in ctx, if any, is marked too, as
// the A and AAAA exchanges of a lookup run at the same time.
func MarkRejected(ctx context.Context, response *dns.Msg) {
	rec := recordFromContext(ctx)
	if rec == nil {
		return
	}
	transportTag, _ := adapter.DNSTransportTagFromContext(ctx)
	var queryType string
	if response != nil && len(response.Question) > 0 {
		queryType = dns.Type(response.Question[0].Qtype).String()
	}
	rec.access.Lock()
	rec.entry.Rejected = true
	if transportTag != "" {
		for i, upstream := range slices.Backward(rec.entry.Upstreams) {
			if upstream.Server == transportTag && (queryType == "" || upstream.QueryType == queryType) {
				rec.entry.Upstreams[i].Rejected = true
				break
			}
		}
	}
	rec.access.Unlock()
}

var errResponseRejected = errors.New("response rejected")

// BeginBackground records a query the DNS client makes on its own, such as a
// background cache refresh, as one entry with its exchange. The returned
// function must be called after the response is checked, so that a rejected
// response is recorded as such.
func BeginBackground(ctx context.Context, message *dns.Msg) (context.Context, func(response *dns.Msg, err error)) {
	store := backgroundStore(ctx, message)
	if store == nil {
		return ctx, noopFinish
	}
	question := message.Question[0]
	ctx, rec := store.beginRecord(ctx, MethodBackground, question.Name, dns.Type(question.Qtype).String())
	return ctx, func(response *dns.Msg, err error) {
		rec.access.Lock()
		rejected := rec.entry.Rejected
		rec.access.Unlock()
		if err == nil && rejected {
			err = errResponseRejected
		}
		rec.finish(response, nil, err)
	}
}

func backgroundStore(ctx context.Context, message *dns.Msg) *Store {
	if activeStores.Load() == 0 || len(message.Question) == 0 {
		return nil
	}
	store := service.PtrFromContext[Store](ctx)
	if store == nil || !store.started.Load() {
		return nil
	}
	return store
}

// BeginUpstream records an exchange with transport. The returned function
// must be called with the result. Exchanges outside of a recorded query are
// recorded as their own entries.
func BeginUpstream(ctx context.Context, transport adapter.DNSTransport, message *dns.Msg) func(response *dns.Msg, err error) {
	if activeStores.Load() == 0 || len(message.Question) == 0 {
		return noopFinish
	}
	question := message.Question[0]
	rec := recordFromContext(ctx)
	var background *record
	if rec == nil {
		store := backgroundStore(ctx, message)
		if store == nil {
			return noopFinish
		}
		_, background = store.beginRecord(ctx, MethodBackground, question.Name, dns.Type(question.Qtype).String())
		rec = background
	}
	start := time.Now()
	server := rec.store.serverOf(transport)
	return func(response *dns.Msg, err error) {
		upstream := Upstream{
			Server:    transport.Tag(),
			Type:      transport.Type(),
			Address:   server.address,
			Detour:    server.detour,
			QueryType: dns.Type(question.Qtype).String(),
			Duration:  time.Since(start).Microseconds(),
		}
		if response != nil {
			upstream.Rcode = dns.RcodeToString[response.Rcode]
			upstream.Answers, _ = formatAnswers(response.Answer)
		}
		if err != nil {
			upstream.Error = err.Error()
		}
		rec.access.Lock()
		if !rec.done {
			rec.entry.Upstreams = append(rec.entry.Upstreams, upstream)
		}
		rec.access.Unlock()
		if background != nil {
			background.finish(response, nil, err)
		}
	}
}

func noopFinish(*dns.Msg, error) {}
