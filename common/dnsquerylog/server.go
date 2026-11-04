package dnsquerylog

import (
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// serverInfo is what the configuration says about a DNS server.
type serverInfo struct {
	address string
	detour  string
}

// SetServers records the address and detour of each configured DNS server,
// so that exchanges are logged with them and not only with the tag. It must
// be called before the store starts.
func (s *Store) SetServers(servers []option.DNSServerOptions) {
	if s == nil {
		return
	}
	s.servers = make(map[string]serverInfo, len(servers))
	for _, server := range servers {
		s.servers[server.Tag] = describeServer(server)
	}
}

// serverOf returns the address and detour of transport. The local and dhcp
// servers, which find their upstream at runtime, report the addresses they use
// now. Others with runtime addresses, such as tailscale, route domains to
// different resolvers or answer some themselves, so their address list would
// not tell which one a query used.
func (s *Store) serverOf(transport adapter.DNSTransport) serverInfo {
	info := s.servers[transport.Tag()]
	switch transport.Type() {
	case C.DNSTypeLocal, C.DNSTypeDHCP:
		if configured, isConfigured := transport.(adapter.DNSTransportWithConfiguration); isConfigured {
			addresses := configured.ServerAddresses()
			if len(addresses) > 0 {
				names := make([]string, 0, len(addresses))
				for _, address := range addresses {
					names = append(names, address.String())
				}
				info.address = strings.Join(names, ",")
			}
		}
	}
	return info
}

func describeServer(server option.DNSServerOptions) serverInfo {
	switch options := server.Options.(type) {
	case *option.RemoteDNSServerOptions:
		return serverInfo{
			address: server.Type + "://" + formatServerHost(options.Server, options.ServerPort, 53),
			detour:  options.Detour,
		}
	case *option.RemoteTLSDNSServerOptions:
		return serverInfo{
			address: server.Type + "://" + formatServerHost(options.Server, options.ServerPort, 853),
			detour:  options.Detour,
		}
	case *option.RemoteHTTPSDNSServerOptions:
		path := options.Path
		if path == "" {
			path = "/dns-query"
		} else if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return serverInfo{
			address: server.Type + "://" + formatServerHost(options.Server, options.ServerPort, 443) + path,
			detour:  options.Detour,
		}
	case *option.LocalDNSServerOptions:
		return serverInfo{detour: options.Detour}
	case *option.DHCPDNSServerOptions:
		return serverInfo{detour: options.Detour}
	case *option.MDNSDNSServerOptions:
		return serverInfo{detour: options.Detour}
	case *option.TailscaleDNSServerOptions:
		// Queries go through the endpoint, which takes the place of a detour.
		return serverInfo{detour: options.Endpoint}
	}
	return serverInfo{}
}

// formatServerHost formats host and port, leaving out the default port of the
// protocol.
func formatServerHost(host string, port uint16, defaultPort uint16) string {
	if port == 0 || port == defaultPort {
		if address, err := netip.ParseAddr(host); err == nil && address.Is6() {
			return "[" + address.String() + "]"
		}
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}
