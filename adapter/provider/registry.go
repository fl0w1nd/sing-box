package provider

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
)

type ConstructorFunc[T any] func(ctx context.Context, logger log.ContextLogger, tag string, options T) (adapter.OutboundProvider, error)

func Register[T any](registry *Registry, providerType string, constructor ConstructorFunc[T]) {
	registry.register(providerType, func() any { return new(T) }, func(ctx context.Context, logger log.ContextLogger, tag string, rawOptions any) (adapter.OutboundProvider, error) {
		var options *T
		if rawOptions != nil {
			options = rawOptions.(*T)
		}
		return constructor(ctx, logger, tag, common.PtrValueOrDefault(options))
	})
}

type (
	optionsConstructorFunc func() any
	constructorFunc        func(context.Context, log.ContextLogger, string, any) (adapter.OutboundProvider, error)
)

type Registry struct {
	access      sync.Mutex
	optionsType map[string]optionsConstructorFunc
	constructor map[string]constructorFunc
}

func NewRegistry() *Registry {
	return &Registry{optionsType: make(map[string]optionsConstructorFunc), constructor: make(map[string]constructorFunc)}
}

func (r *Registry) OptionTypes() []string {
	r.access.Lock()
	defer r.access.Unlock()
	return slices.Sorted(maps.Keys(r.optionsType))
}

func (r *Registry) CreateOptions(providerType string) (any, bool) {
	r.access.Lock()
	defer r.access.Unlock()
	constructor, loaded := r.optionsType[providerType]
	if !loaded {
		return nil, false
	}
	return constructor(), true
}

func (r *Registry) Create(ctx context.Context, logger log.ContextLogger, tag string, providerType string, options any) (adapter.OutboundProvider, error) {
	r.access.Lock()
	constructor, loaded := r.constructor[providerType]
	r.access.Unlock()
	if !loaded {
		return nil, E.New("provider type not found: ", providerType)
	}
	return constructor(ctx, logger, tag, options)
}

func (r *Registry) register(providerType string, optionsConstructor optionsConstructorFunc, constructor constructorFunc) {
	r.access.Lock()
	defer r.access.Unlock()
	r.optionsType[providerType] = optionsConstructor
	r.constructor[providerType] = constructor
}

var _ adapter.ProviderRegistry = (*Registry)(nil)
