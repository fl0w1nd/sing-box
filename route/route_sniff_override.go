package route

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	R "github.com/sagernet/sing-box/route/rule"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"
)

// markSniffOverride records that a sniff action asked to hand the sniffed
// domain to the outbound. The destination is kept for the following rules.
func markSniffOverride(metadata *adapter.InboundContext, action *R.RuleActionSniff) {
	if !action.OverrideOutbound {
		return
	}
	metadata.SniffOverrideDestination = true
	metadata.SniffOverrideDirect = metadata.SniffOverrideDirect || action.OverrideDirect
}

// overrideOutboundDestination hands the sniffed domain instead of the IP to
// the outbound when a sniff action asked for it, so a proxy resolves the
// domain on the remote side. Rules have already matched the original IP, and
// outbounds that would resolve the domain locally again (direct and
// endpoints) keep the IP the client got. With override_direct direct
// resolves it again too, for a server whose clients resolved the IP on their
// side. For UDP the connection manager maps the replies back to the original
// destination.
//
// Only a domain read from the connection itself is used. A domain from the
// DNS reverse mapping is a guess when several domains share the IP, so it is
// left to rule matching, as Mihomo does with its DNS mapping.
func (r *Router) overrideOutboundDestination(ctx context.Context, metadata *adapter.InboundContext, outbound adapter.Outbound) {
	if !metadata.SniffOverrideDestination {
		return
	}
	if !shouldOverrideOutboundDestination(metadata, outbound, service.FromContext[adapter.EndpointManager](r.ctx)) {
		return
	}
	metadata.RouteOriginalDestination = metadata.Destination
	metadata.Destination = M.Socksaddr{
		Fqdn: metadata.Domain,
		Port: metadata.Destination.Port,
	}
	r.logger.DebugContext(ctx, "override destination from ", metadata.RouteOriginalDestination, " to ", metadata.Destination)
}

func shouldOverrideOutboundDestination(metadata *adapter.InboundContext, outbound adapter.Outbound, endpointManager adapter.EndpointManager) bool {
	if !metadata.SniffOverrideDestination || !metadata.Destination.IsIP() || !sniffedDomain(metadata) {
		return false
	}
	// The destination was rewritten by route options.
	if metadata.RouteOriginalDestination.IsValid() {
		return false
	}
	if outbound.Type() == C.TypeDirect {
		return metadata.SniffOverrideDirect
	}
	// Outbounds implement adapter.Endpoint too, so look the tag up instead.
	if endpointManager != nil {
		if _, isEndpoint := endpointManager.Get(outbound.Tag()); isEndpoint {
			return false
		}
	}
	return true
}

// sniffedDomain reports whether metadata.Domain was read from the connection:
// the TLS and QUIC sniffers set it from the server name and the HTTP sniffer
// from the host, while other protocols leave the domain of the DNS reverse
// mapping in place.
func sniffedDomain(metadata *adapter.InboundContext) bool {
	if metadata.SniffError != nil || !M.IsDomainName(metadata.Domain) {
		return false
	}
	switch metadata.Protocol {
	case C.ProtocolTLS, C.ProtocolHTTP, C.ProtocolQUIC:
		return true
	default:
		return false
	}
}
