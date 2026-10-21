package adapter

// DynamicOutboundManager publishes outbounds created at runtime, such as provider
// nodes. The caller owns their lifecycle: an outbound is started before it is
// published and closed after it is replaced or removed.
type DynamicOutboundManager interface {
	OutboundManager
	// SwapDynamic replaces previous with next under tag. previous must be the
	// dynamic outbound currently registered under tag, or nil if tag is unused;
	// a nil next removes the tag.
	SwapDynamic(tag string, previous Outbound, next Outbound) error
}
