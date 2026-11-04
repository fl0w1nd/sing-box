package dnsquerylog

import (
	"context"
	"net/netip"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"

	"github.com/miekg/dns"
)

var _ adapter.DNSRouter = (*Router)(nil)

// Router records the queries of the DNS router it wraps.
type Router struct {
	adapter.DNSRouter
	store *Store
}

func (r *Router) Exchange(ctx context.Context, message *dns.Msg, options adapter.DNSQueryOptions) (*dns.Msg, error) {
	if len(message.Question) == 0 {
		return r.DNSRouter.Exchange(ctx, message, options)
	}
	question := message.Question[0]
	ctx, rec := r.store.beginRecord(ctx, MethodExchange, question.Name, dns.Type(question.Qtype).String())
	response, err := r.DNSRouter.Exchange(ctx, message, options)
	rec.finish(response, nil, err)
	return response, err
}

func (r *Router) ExchangeAsync(ctx context.Context, message *dns.Msg, options adapter.DNSQueryOptions, callback func(response *dns.Msg, err error)) {
	if len(message.Question) == 0 {
		r.DNSRouter.ExchangeAsync(ctx, message, options, callback)
		return
	}
	question := message.Question[0]
	ctx, rec := r.store.beginRecord(ctx, MethodExchange, question.Name, dns.Type(question.Qtype).String())
	r.DNSRouter.ExchangeAsync(ctx, message, options, func(response *dns.Msg, err error) {
		rec.finish(response, nil, err)
		callback(response, err)
	})
}

func (r *Router) Lookup(ctx context.Context, domain string, options adapter.DNSQueryOptions) ([]netip.Addr, error) {
	ctx, rec := r.store.beginRecord(ctx, MethodLookup, domain, lookupQueryType(options))
	addresses, err := r.DNSRouter.Lookup(ctx, domain, options)
	recorded := addresses
	if recorded == nil && err == nil {
		recorded = []netip.Addr{}
	}
	rec.finish(nil, recorded, err)
	return addresses, err
}

// lookupQueryType is the query type of a lookup when the strategy fixes it.
// Otherwise it is left to the exchanges made.
func lookupQueryType(options adapter.DNSQueryOptions) string {
	strategy := options.LookupStrategy
	if strategy == C.DomainStrategyAsIS {
		strategy = options.Strategy
	}
	switch strategy {
	case C.DomainStrategyIPv4Only:
		return "A"
	case C.DomainStrategyIPv6Only:
		return "AAAA"
	default:
		return ""
	}
}
