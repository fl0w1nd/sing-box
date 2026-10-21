package clashapi

import (
	"context"
	"time"

	"github.com/sagernet/sing-box/common/urltest"
)

// createURLTestContext carries the unified delay setting of the server into a
// URL test started by an API request.
func createURLTestContext(server *Server, parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	if urltest.IsUnifiedDelayFromContext(server.ctx) {
		ctx = urltest.ContextWithIsUnifiedDelay(ctx)
	}
	return ctx, cancel
}
