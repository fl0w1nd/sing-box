package dnsquerylog

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func remoteServer(serverType string, tag string, server string, port uint16, detour string) option.DNSServerOptions {
	remote := option.RemoteDNSServerOptions{
		RawLocalDNSServerOptions: option.RawLocalDNSServerOptions{DialerOptions: option.DialerOptions{Detour: detour}},
		DNSServerAddressOptions:  option.DNSServerAddressOptions{Server: server, ServerPort: port},
	}
	options := option.DNSServerOptions{Type: serverType, Tag: tag}
	switch serverType {
	case C.DNSTypeUDP, C.DNSTypeTCP:
		options.Options = &remote
	case C.DNSTypeTLS, C.DNSTypeQUIC:
		options.Options = &option.RemoteTLSDNSServerOptions{RemoteDNSServerOptions: remote}
	}
	return options
}

func httpsServer(serverType string, tag string, server string, port uint16, path string, detour string) option.DNSServerOptions {
	options := remoteServer(C.DNSTypeUDP, tag, server, port, detour)
	return option.DNSServerOptions{Type: serverType, Tag: tag, Options: &option.RemoteHTTPSDNSServerOptions{
		RemoteTLSDNSServerOptions: option.RemoteTLSDNSServerOptions{RemoteDNSServerOptions: *options.Options.(*option.RemoteDNSServerOptions)},
		Path:                      path,
	}}
}

func TestDescribeServer(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		server option.DNSServerOptions
		info   serverInfo
	}{
		{"udp", remoteServer(C.DNSTypeUDP, "a", "223.5.5.5", 53, ""), serverInfo{address: "udp://223.5.5.5"}},
		{"udp without port", remoteServer(C.DNSTypeUDP, "a", "223.5.5.5", 0, ""), serverInfo{address: "udp://223.5.5.5"}},
		{"udp with port", remoteServer(C.DNSTypeUDP, "a", "223.5.5.5", 5353, ""), serverInfo{address: "udp://223.5.5.5:5353"}},
		{"tcp IPv6", remoteServer(C.DNSTypeTCP, "a", "2400:3200::1", 0, ""), serverInfo{address: "tcp://[2400:3200::1]"}},
		{"tcp IPv6 with port", remoteServer(C.DNSTypeTCP, "a", "2400:3200::1", 5353, ""), serverInfo{address: "tcp://[2400:3200::1]:5353"}},
		{"tls", remoteServer(C.DNSTypeTLS, "a", "dns.google", 853, "proxy"), serverInfo{address: "tls://dns.google", detour: "proxy"}},
		{"quic with port", remoteServer(C.DNSTypeQUIC, "a", "dns.adguard.com", 784, ""), serverInfo{address: "quic://dns.adguard.com:784"}},
		{"https", httpsServer(C.DNSTypeHTTPS, "a", "dns.google", 443, "", "dns-out"), serverInfo{address: "https://dns.google/dns-query", detour: "dns-out"}},
		{"https with port and path", httpsServer(C.DNSTypeHTTPS, "a", "1.1.1.1", 8443, "resolve", ""), serverInfo{address: "https://1.1.1.1:8443/resolve"}},
		{"h3", httpsServer(C.DNSTypeHTTP3, "a", "dns.google", 0, "/dns-query", ""), serverInfo{address: "h3://dns.google/dns-query"}},
		{"local", option.DNSServerOptions{Type: C.DNSTypeLocal, Tag: "a", Options: &option.LocalDNSServerOptions{
			RawLocalDNSServerOptions: option.RawLocalDNSServerOptions{DialerOptions: option.DialerOptions{Detour: "direct"}},
		}}, serverInfo{detour: "direct"}},
		{"tailscale", option.DNSServerOptions{Type: C.DNSTypeTailscale, Tag: "a", Options: &option.TailscaleDNSServerOptions{Endpoint: "ts-ep"}}, serverInfo{detour: "ts-ep"}},
		{"fakeip", option.DNSServerOptions{Type: C.DNSTypeFakeIP, Tag: "a", Options: &option.FakeIPDNSServerOptions{}}, serverInfo{}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.info, describeServer(testCase.server))
		})
	}
}

type testConfiguredTransport struct {
	testTransport
	addresses []netip.Addr
}

func (t *testConfiguredTransport) ServerAddresses() []netip.Addr { return t.addresses }
func (t *testConfiguredTransport) SearchDomains() []string       { return nil }

// Exchanges are logged with the configured address and detour, or with the
// addresses a local server uses at the time.
func TestUpstreamRecordsServer(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	store.SetServers([]option.DNSServerOptions{
		httpsServer(C.DNSTypeHTTPS, "remote", "dns.google", 443, "", "dns-out"),
		{Type: C.DNSTypeLocal, Tag: "local", Options: &option.LocalDNSServerOptions{}},
		{Type: C.DNSTypeTailscale, Tag: "tailscale", Options: &option.TailscaleDNSServerOptions{Endpoint: "ts-ep"}},
	})
	ctx := service.ContextWithPtr(context.Background(), store)
	exchange := func(transport adapter.DNSTransport, name string) {
		message := new(dns.Msg)
		message.SetQuestion(name, dns.TypeA)
		finish := BeginUpstream(ctx, transport, message)
		finish(answer(message, "1.2.3.4"), nil)
	}
	exchange(&testTransport{tag: "remote", transportType: C.DNSTypeHTTPS}, "remote.test.")
	exchange(&testConfiguredTransport{
		testTransport: testTransport{tag: "local", transportType: C.DNSTypeLocal},
		addresses:     []netip.Addr{netip.MustParseAddr("192.168.1.1"), netip.MustParseAddr("fe80::1")},
	}, "local.test.")
	exchange(&testTransport{tag: "unknown", transportType: C.DNSTypeUDP}, "unknown.test.")
	// The resolvers of a tailnet are not what a MagicDNS name is answered by.
	exchange(&testConfiguredTransport{
		testTransport: testTransport{tag: "tailscale", transportType: C.DNSTypeTailscale},
		addresses:     []netip.Addr{netip.MustParseAddr("199.247.155.53")},
	}, "host.tailnet.ts.net.")

	entries := queryAll(t, store, Filter{})
	require.Len(t, entries, 4)
	byDomain := make(map[string]Upstream)
	for _, entry := range entries {
		require.Len(t, entry.Upstreams, 1)
		byDomain[entry.Domain] = entry.Upstreams[0]
	}
	require.Equal(t, "https://dns.google/dns-query", byDomain["remote.test"].Address)
	require.Equal(t, "dns-out", byDomain["remote.test"].Detour)
	require.Equal(t, "192.168.1.1,fe80::1", byDomain["local.test"].Address)
	require.Empty(t, byDomain["local.test"].Detour)
	require.Empty(t, byDomain["unknown.test"].Address)
	require.Empty(t, byDomain["host.tailnet.ts.net"].Address)
	require.Equal(t, "ts-ep", byDomain["host.tailnet.ts.net"].Detour)
}

// The stats of a server show the address of the newest exchange.
func TestStatsUpstreamAddressIsNewest(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	for _, address := range []string{"udp://223.5.5.5", "udp://223.6.6.6"} {
		store.add(Entry{Time: time.Now().UnixMilli(), Domain: "a.test", QueryType: "A", Status: StatusUpstream, Upstreams: []Upstream{
			{Server: "alidns", Type: C.DNSTypeUDP, Address: address, QueryType: "A"},
		}})
	}
	stats, err := store.Stats(context.Background(), StatsOptions{})
	require.NoError(t, err)
	require.Len(t, stats.Upstreams, 1)
	require.Equal(t, "udp://223.6.6.6", stats.Upstreams[0].Address)
	require.Equal(t, int64(2), stats.Upstreams[0].Exchanges)
}
