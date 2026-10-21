//go:build with_tailscale

package daemon

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/protocol/tailscale"
)

const withTailscale = true

func setTailscaleForceRelay(cacheFile adapter.CacheFile, enabled bool) error {
	return tailscale.SetForceRelay(cacheFile, enabled)
}

func subscribeTailscaleForceRelay(ctx context.Context, listener func(bool)) error {
	return tailscale.SubscribeForceRelay(ctx, listener)
}
