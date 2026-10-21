package group

import (
	"context"
	"regexp"

	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/sing/service"
)

// members are the static outbounds of a group followed by the outbounds of its
// providers, which change at runtime.
type members struct {
	outbound        adapter.OutboundManager
	providerManager adapter.ProviderManager
	staticTags      []string
	providerTags    []string
	useAllProviders bool
	include         *regexp.Regexp
	exclude         *regexp.Regexp
	providers       []adapter.OutboundProvider
}

func newMembers(ctx context.Context, staticTags []string, providerTags []string, useAllProviders bool, include *badoption.Regexp, exclude *badoption.Regexp) (*members, error) {
	if len(staticTags) == 0 && len(providerTags) == 0 && !useAllProviders {
		return nil, E.New("missing tags")
	}
	return &members{
		outbound:        service.FromContext[adapter.OutboundManager](ctx),
		providerManager: service.FromContext[adapter.ProviderManager](ctx),
		staticTags:      staticTags,
		providerTags:    providerTags,
		useAllProviders: useAllProviders,
		include:         buildRegexp(include),
		exclude:         buildRegexp(exclude),
	}, nil
}

func (m *members) dynamic() bool {
	return len(m.providerTags) > 0 || m.useAllProviders
}

// start resolves the members and calls update after every provider change
// until scope closes.
func (m *members) start(scope *adapter.Scope, update func()) error {
	for i, tag := range m.staticTags {
		_, loaded := m.outbound.Outbound(tag)
		if !loaded {
			return E.New("outbound ", i, " not found: ", tag)
		}
	}
	if !m.dynamic() {
		return nil
	}
	if m.providerManager == nil {
		return E.New("missing provider manager")
	}
	if m.useAllProviders {
		m.providers = m.providerManager.Providers()
	} else {
		for _, tag := range m.providerTags {
			provider, loaded := m.providerManager.Get(tag)
			if !loaded {
				return E.New("provider not found: ", tag)
			}
			m.providers = append(m.providers, provider)
		}
	}
	for _, provider := range m.providers {
		element := provider.RegisterCallback(update)
		scope.Add(func() error {
			provider.UnregisterCallback(element)
			return nil
		})
	}
	return nil
}

func (m *members) outbounds() []adapter.Outbound {
	outbounds := make([]adapter.Outbound, 0, len(m.staticTags))
	seen := make(map[string]bool, len(m.staticTags))
	for _, tag := range m.staticTags {
		if outbound, loaded := m.outbound.Outbound(tag); loaded {
			outbounds = append(outbounds, outbound)
			seen[tag] = true
		}
	}
	for _, provider := range m.providers {
		for _, outbound := range provider.Outbounds() {
			tag := outbound.Tag()
			if seen[tag] || m.exclude != nil && m.exclude.MatchString(tag) || m.include != nil && !m.include.MatchString(tag) {
				continue
			}
			seen[tag] = true
			outbounds = append(outbounds, outbound)
		}
	}
	return outbounds
}

func buildRegexp(value *badoption.Regexp) *regexp.Regexp {
	if value == nil {
		return nil
	}
	return value.Build()
}

func outboundTags(outbounds []adapter.Outbound) []string {
	tags := make([]string, 0, len(outbounds))
	for _, outbound := range outbounds {
		tags = append(tags, outbound.Tag())
	}
	return tags
}
