package provider

import (
	"context"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
)

var _ adapter.ProviderManager = (*Manager)(nil)

type Manager struct {
	registry      adapter.ProviderRegistry
	access        sync.Mutex
	providers     []adapter.OutboundProvider
	providerByTag map[string]adapter.OutboundProvider
}

func NewManager(registry adapter.ProviderRegistry) *Manager {
	return &Manager{
		registry:      registry,
		providerByTag: make(map[string]adapter.OutboundProvider),
	}
}

func (m *Manager) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	m.access.Lock()
	providers := m.providers
	m.access.Unlock()
	for _, provider := range providers {
		name := "provider/" + provider.Type() + "[" + provider.Tag() + "]"
		err := scope.Start(name, provider, stage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Providers() []adapter.OutboundProvider {
	m.access.Lock()
	defer m.access.Unlock()
	return m.providers
}

func (m *Manager) Get(tag string) (adapter.OutboundProvider, bool) {
	m.access.Lock()
	provider, found := m.providerByTag[tag]
	m.access.Unlock()
	return provider, found
}

func (m *Manager) Create(ctx context.Context, logger log.ContextLogger, tag string, providerType string, options any) error {
	provider, err := m.registry.Create(ctx, logger, tag, providerType, options)
	if err != nil {
		return err
	}
	m.access.Lock()
	defer m.access.Unlock()
	_, loaded := m.providerByTag[tag]
	if loaded {
		return E.New("duplicate provider tag: ", tag)
	}
	m.providers = append(m.providers, provider)
	m.providerByTag[tag] = provider
	return nil
}
