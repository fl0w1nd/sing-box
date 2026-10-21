package urltest

import (
	"context"

	"github.com/sagernet/sing-box/option"
)

type contextKeyIsUnifiedDelay struct{}

func ContextWithIsUnifiedDelay(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyIsUnifiedDelay{}, true)
}

// ContextWithExperimentalOptions enables unified delay for URL tests run under
// ctx when the configuration asks for it.
func ContextWithExperimentalOptions(ctx context.Context, options *option.ExperimentalOptions) context.Context {
	if options != nil && options.UnifiedDelay != nil && options.UnifiedDelay.Enabled {
		return ContextWithIsUnifiedDelay(ctx)
	}
	return ctx
}

func IsUnifiedDelayFromContext(ctx context.Context) bool {
	return ctx.Value(contextKeyIsUnifiedDelay{}) != nil
}
