//go:build !with_tailscale

package daemon

import (
	"context"

	"github.com/sagernet/sing-box/adapter"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const withTailscale = false

func setTailscaleForceRelay(adapter.CacheFile, bool) error {
	return status.Error(codes.Unimplemented, "Tailscale is not included in this build")
}

func subscribeTailscaleForceRelay(context.Context, func(bool)) error {
	return status.Error(codes.Unimplemented, "Tailscale is not included in this build")
}
