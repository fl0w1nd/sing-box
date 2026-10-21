package route

import (
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/sniff"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	R "github.com/sagernet/sing-box/route/rule"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"
)

type sniffTestRule struct {
	adapter.Rule
	match  func(metadata *adapter.InboundContext) bool
	action adapter.RuleAction
}

func (r *sniffTestRule) Match(metadata *adapter.InboundContext) bool { return r.match(metadata) }
func (r *sniffTestRule) Action() adapter.RuleAction                  { return r.action }
func (r *sniffTestRule) String() string                              { return "" }

type sniffTestOutbound struct {
	adapter.Outbound
	outboundType string
	tag          string
}

func (o *sniffTestOutbound) Type() string { return o.outboundType }
func (o *sniffTestOutbound) Tag() string  { return o.tag }

type sniffTestEndpointManager struct {
	adapter.EndpointManager
	tags []string
}

func (m *sniffTestEndpointManager) Get(tag string) (adapter.Endpoint, bool) {
	return nil, slices.Contains(m.tags, tag)
}

type sniffTestDNSTransportManager struct {
	adapter.DNSTransportManager
}

func (m *sniffTestDNSTransportManager) FakeIP() adapter.FakeIPTransport { return nil }

type sniffTestDNSRouter struct {
	adapter.DNSRouter
	reverseMapping map[netip.Addr]string
}

func (r *sniffTestDNSRouter) LookupReverseMapping(address netip.Addr) (string, bool) {
	domain, loaded := r.reverseMapping[address]
	return domain, loaded
}

func newSniffTestRouter(rules ...adapter.Rule) *Router {
	return &Router{
		logger:       log.NewNOPFactory().NewLogger("test"),
		dns:          &sniffTestDNSRouter{},
		dnsTransport: &sniffTestDNSTransportManager{},
		rules:        rules,
	}
}

func newSniffAction(t *testing.T, override bool) adapter.RuleAction {
	t.Helper()
	return newSniffActionWithOptions(t, option.RouteActionSniff{OverrideDestination: override})
}

func newSniffActionWithOptions(t *testing.T, options option.RouteActionSniff) adapter.RuleAction {
	t.Helper()
	action, err := R.NewRuleAction(log.NewNOPFactory().NewLogger("test"), option.RuleAction{
		Action:       C.RuleActionTypeSniff,
		SniffOptions: options,
	})
	if err != nil {
		t.Fatal(err)
	}
	return action
}

// Rules after a sniff with override_destination still see the original IP.
func TestSniffOverrideKeepsIPForRules(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	go clientConn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))

	router := newSniffTestRouter(
		&sniffTestRule{
			match:  func(*adapter.InboundContext) bool { return true },
			action: newSniffAction(t, true),
		},
		&sniffTestRule{
			match: func(metadata *adapter.InboundContext) bool {
				return metadata.Destination == M.ParseSocksaddr("203.0.113.1:80")
			},
			action: &R.RuleActionRoute{Outbound: "proxy"},
		},
	)
	metadata := adapter.InboundContext{
		Network:     "tcp",
		Destination: M.ParseSocksaddr("203.0.113.1:80"),
	}
	_, ruleIndex, _, _, err := router.matchRule(t.Context(), &metadata, serverConn, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ruleIndex != 1 {
		t.Fatalf("IP rule after sniff did not match: index=%d destination=%s", ruleIndex, metadata.Destination)
	}
	if metadata.Domain != "example.com" || !metadata.SniffOverrideDestination {
		t.Fatalf("sniff result not kept: domain=%q override=%v", metadata.Domain, metadata.SniffOverrideDestination)
	}
}

func TestSniffWithoutOverrideDoesNotMark(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	go clientConn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))

	router := newSniffTestRouter(&sniffTestRule{
		match:  func(*adapter.InboundContext) bool { return true },
		action: newSniffAction(t, false),
	})
	metadata := adapter.InboundContext{
		Network:     "tcp",
		Destination: M.ParseSocksaddr("203.0.113.1:80"),
	}
	if _, _, _, _, err := router.matchRule(t.Context(), &metadata, serverConn, nil); err != nil {
		t.Fatal(err)
	}
	if metadata.Domain != "example.com" || metadata.SniffOverrideDestination {
		t.Fatalf("unexpected sniff state: domain=%q override=%v", metadata.Domain, metadata.SniffOverrideDestination)
	}
}

// override_direct only takes effect with override_destination.
func TestSniffOverrideDirectMark(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		options  option.RouteActionSniff
		override bool
		direct   bool
	}{
		{name: "both", options: option.RouteActionSniff{OverrideDestination: true, OverrideDirect: true}, override: true, direct: true},
		{name: "destination only", options: option.RouteActionSniff{OverrideDestination: true}, override: true},
		{name: "direct only", options: option.RouteActionSniff{OverrideDirect: true}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			go clientConn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
			router := newSniffTestRouter(&sniffTestRule{
				match:  func(*adapter.InboundContext) bool { return true },
				action: newSniffActionWithOptions(t, testCase.options),
			})
			metadata := adapter.InboundContext{
				Network:     "tcp",
				Destination: M.ParseSocksaddr("203.0.113.1:80"),
			}
			if _, _, _, _, err := router.matchRule(t.Context(), &metadata, serverConn, nil); err != nil {
				t.Fatal(err)
			}
			if metadata.SniffOverrideDestination != testCase.override || metadata.SniffOverrideDirect != testCase.direct {
				t.Fatalf("override=%v direct=%v, want %v %v", metadata.SniffOverrideDestination, metadata.SniffOverrideDirect, testCase.override, testCase.direct)
			}
		})
	}
}

// A domain from the DNS reverse mapping matches rules but is not handed to
// the outbound, here with sniffing skipped for the SMTP port.
func TestSniffOverrideIgnoresReverseMapping(t *testing.T) {
	router := newSniffTestRouter(
		&sniffTestRule{
			match:  func(*adapter.InboundContext) bool { return true },
			action: newSniffAction(t, true),
		},
		&sniffTestRule{
			match: func(metadata *adapter.InboundContext) bool {
				return metadata.Domain == "mapped.example"
			},
			action: &R.RuleActionRoute{Outbound: "proxy"},
		},
	)
	router.dns = &sniffTestDNSRouter{reverseMapping: map[netip.Addr]string{
		netip.MustParseAddr("203.0.113.1"): "mapped.example",
	}}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	metadata := adapter.InboundContext{
		Network:     "tcp",
		Destination: M.ParseSocksaddr("203.0.113.1:25"),
	}
	_, ruleIndex, _, _, err := router.matchRule(t.Context(), &metadata, serverConn, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ruleIndex != 1 {
		t.Fatalf("domain rule did not match the reverse mapped domain: index=%d", ruleIndex)
	}
	proxy := &sniffTestOutbound{outboundType: C.TypeVMess, tag: "proxy"}
	if shouldOverrideOutboundDestination(&metadata, proxy, nil) {
		t.Fatal("reverse mapped domain handed to the outbound")
	}
}

func TestOverrideOutboundDestination(t *testing.T) {
	ipDestination := M.ParseSocksaddr("203.0.113.1:443")
	sniffed := func() adapter.InboundContext {
		return adapter.InboundContext{
			Destination:              ipDestination,
			Domain:                   "example.com",
			Protocol:                 C.ProtocolTLS,
			SniffOverrideDestination: true,
		}
	}
	proxy := &sniffTestOutbound{outboundType: C.TypeVMess, tag: "proxy"}
	endpoints := &sniffTestEndpointManager{tags: []string{"wg"}}
	for _, testCase := range []struct {
		name     string
		metadata func() adapter.InboundContext
		outbound adapter.Outbound
		override bool
	}{
		{name: "proxy", metadata: sniffed, outbound: proxy, override: true},
		{name: "direct", metadata: sniffed, outbound: &sniffTestOutbound{outboundType: C.TypeDirect, tag: "direct"}},
		{
			name: "direct with override_direct",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.SniffOverrideDirect = true
				return metadata
			},
			outbound: &sniffTestOutbound{outboundType: C.TypeDirect, tag: "direct"},
			override: true,
		},
		{
			name: "endpoint with override_direct",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.SniffOverrideDirect = true
				return metadata
			},
			outbound: &sniffTestOutbound{outboundType: C.TypeWireGuard, tag: "wg"},
		},
		{name: "endpoint", metadata: sniffed, outbound: &sniffTestOutbound{outboundType: C.TypeWireGuard, tag: "wg"}},
		{
			name: "not requested",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.SniffOverrideDestination = false
				return metadata
			},
			outbound: proxy,
		},
		{
			name: "no domain",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.Domain = ""
				return metadata
			},
			outbound: proxy,
		},
		{
			name: "http",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.Protocol = C.ProtocolHTTP
				return metadata
			},
			outbound: proxy,
			override: true,
		},
		{
			name: "quic",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.Protocol = C.ProtocolQUIC
				return metadata
			},
			outbound: proxy,
			override: true,
		},
		{
			// A domain from the DNS reverse mapping, with sniffing skipped.
			name: "reverse mapping only",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.Protocol = ""
				return metadata
			},
			outbound: proxy,
		},
		{
			name: "reverse mapping with ssh",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.Protocol = C.ProtocolSSH
				return metadata
			},
			outbound: proxy,
		},
		{
			name: "sniff failed",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.SniffError = sniff.ErrNeedMoreData
				return metadata
			},
			outbound: proxy,
		},
		{
			name: "domain destination",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.Destination = M.ParseSocksaddr("other.example:443")
				return metadata
			},
			outbound: proxy,
		},
		{
			name: "route options override address",
			metadata: func() adapter.InboundContext {
				metadata := sniffed()
				metadata.RouteOriginalDestination = M.ParseSocksaddr("198.51.100.1:443")
				return metadata
			},
			outbound: proxy,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := testCase.metadata()
			original := metadata
			if shouldOverrideOutboundDestination(&metadata, testCase.outbound, endpoints) != testCase.override {
				t.Fatalf("override = %v, want %v", !testCase.override, testCase.override)
			}
			router := &Router{logger: log.NewNOPFactory().NewLogger("test")}
			router.ctx = service.ContextWith[adapter.EndpointManager](t.Context(), endpoints)
			if !testCase.override {
				router.overrideOutboundDestination(t.Context(), &metadata, testCase.outbound)
				if metadata.Destination != original.Destination || metadata.RouteOriginalDestination != original.RouteOriginalDestination {
					t.Fatalf("destination changed: %s (original %s)", metadata.Destination, metadata.RouteOriginalDestination)
				}
				return
			}
			router.overrideOutboundDestination(t.Context(), &metadata, testCase.outbound)
			if metadata.Destination != M.ParseSocksaddr("example.com:443") {
				t.Fatalf("destination = %s, want example.com:443", metadata.Destination)
			}
			// The connection manager maps UDP replies back to this address.
			if metadata.RouteOriginalDestination != ipDestination {
				t.Fatalf("route original destination = %s, want %s", metadata.RouteOriginalDestination, ipDestination)
			}
		})
	}
}
