package dns

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dnsquerylog"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

type queryLogTestTransport struct {
	adapter.DNSTransport
}

func (t *queryLogTestTransport) Tag() string  { return "remote" }
func (t *queryLogTestTransport) Type() string { return C.DNSTypeUDP }

func (t *queryLogTestTransport) Exchange(_ context.Context, message *dns.Msg) (*dns.Msg, error) {
	response := new(dns.Msg)
	response.SetReply(message)
	response.Answer = append(response.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: message.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30},
		A:   netip.MustParseAddr("1.2.3.4").AsSlice(),
	})
	return response, nil
}

// queryLogTestRouter sends every query to the real DNS client.
type queryLogTestRouter struct {
	adapter.DNSRouter
	client *Client
}

func (r *queryLogTestRouter) Exchange(ctx context.Context, message *dns.Msg, options adapter.DNSQueryOptions) (*dns.Msg, error) {
	return r.client.Exchange(ctx, &queryLogTestTransport{}, message, options, nil)
}

func (r *queryLogTestRouter) Lookup(ctx context.Context, domain string, options adapter.DNSQueryOptions) ([]netip.Addr, error) {
	return r.client.Lookup(ctx, &queryLogTestTransport{}, domain, options, nil)
}

// A lookup answered from the cache, such as for the server address of an
// outbound, is recorded as cached.
func TestQueryLogLookupFromCache(t *testing.T) {
	store := dnsquerylog.New(context.Background(), log.NewNOPFactory().NewLogger("test"), &option.DNSQueryLogOptions{Enabled: true, MemoryOnly: true})
	require.NoError(t, store.Start(adapter.StartStateInitialize, nil))
	defer store.Close()
	router := store.WrapRouter(&queryLogTestRouter{client: NewClient(ClientOptions{
		Context: context.Background(),
		Logger:  log.NewNOPFactory().NewLogger("test"),
	})})
	options := adapter.DNSQueryOptions{Strategy: C.DomainStrategyIPv4Only}
	for range 2 {
		addresses, err := router.Lookup(context.Background(), "example.com", options)
		require.NoError(t, err)
		require.Equal(t, []netip.Addr{netip.MustParseAddr("1.2.3.4")}, addresses)
	}
	entries, _, err := store.Query(dnsquerylog.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	// Newest first.
	require.Equal(t, []string{dnsquerylog.StatusCached, dnsquerylog.StatusUpstream}, []string{entries[0].Status, entries[1].Status})
}

// The query log tells an answer excluded by the domain strategy from a cache
// hit, with the cache enabled or disabled.
func TestQueryLogStatusFromClient(t *testing.T) {
	for _, disableCache := range []bool{false, true} {
		store := dnsquerylog.New(context.Background(), log.NewNOPFactory().NewLogger("test"), &option.DNSQueryLogOptions{Enabled: true, MemoryOnly: true})
		require.NoError(t, store.Start(adapter.StartStateInitialize, nil))
		router := store.WrapRouter(&queryLogTestRouter{client: NewClient(ClientOptions{
			Context:      context.Background(),
			DisableCache: disableCache,
			Logger:       log.NewNOPFactory().NewLogger("test"),
		})})
		options := adapter.DNSQueryOptions{Strategy: C.DomainStrategyIPv4Only}
		for _, queryType := range []uint16{dns.TypeA, dns.TypeA, dns.TypeAAAA} {
			message := new(dns.Msg)
			message.SetQuestion("example.com.", queryType)
			_, err := router.Exchange(context.Background(), message, options)
			require.NoError(t, err)
		}
		entries, _, err := store.Query(dnsquerylog.Filter{}, 0, 10)
		require.NoError(t, err)
		require.Len(t, entries, 3)
		// Newest first.
		secondA := dnsquerylog.StatusCached
		if disableCache {
			secondA = dnsquerylog.StatusUpstream
		}
		require.Equal(t, []string{dnsquerylog.StatusStrategy, secondA, dnsquerylog.StatusUpstream},
			[]string{entries[0].Status, entries[1].Status, entries[2].Status}, "disable_cache=%v", disableCache)
		require.Equal(t, "NOERROR", entries[0].Rcode)
		require.Empty(t, entries[0].Upstreams)
		store.Close()
	}
}

// A background refresh whose response is rejected is recorded as such, not
// as an upstream answer.
func TestQueryLogBackgroundRefreshRejected(t *testing.T) {
	store := dnsquerylog.New(context.Background(), log.NewNOPFactory().NewLogger("test"), &option.DNSQueryLogOptions{Enabled: true, MemoryOnly: true})
	require.NoError(t, store.Start(adapter.StartStateInitialize, nil))
	defer store.Close()
	client := NewClient(ClientOptions{
		Context: service.ContextWithPtr(context.Background(), store),
		Logger:  log.NewNOPFactory().NewLogger("test"),
	})
	transport := &queryLogTestTransport{}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	options := adapter.DNSQueryOptions{}
	key := client.newCacheKey(transport, message.Question[0], message, options)
	client.backgroundRefreshDNS(transport, key, message, options, func(*dns.Msg) bool { return false })
	var entries []dnsquerylog.Entry
	require.Eventually(t, func() bool {
		var err error
		entries, _, err = store.Query(dnsquerylog.Filter{}, 0, 10)
		return err == nil && len(entries) > 0
	}, 5*time.Second, 10*time.Millisecond)
	require.Len(t, entries, 1)
	require.Equal(t, dnsquerylog.MethodBackground, entries[0].Method)
	require.Equal(t, dnsquerylog.StatusError, entries[0].Status)
	require.True(t, entries[0].Rejected)
	require.Len(t, entries[0].Upstreams, 1)
	require.True(t, entries[0].Upstreams[0].Rejected)
}
