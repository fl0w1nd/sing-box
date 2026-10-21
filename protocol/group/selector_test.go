package group

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	boxOutbound "github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/interrupt"
	"github.com/sagernet/sing-box/log"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/x/list"
	"github.com/sagernet/sing/service"
)

func TestSelectorEmptyDialers(t *testing.T) {
	selector := &Selector{}
	stream, streamErr := selector.DialContext(context.Background(), N.NetworkTCP, M.Socksaddr{})
	if stream != nil || streamErr == nil || !strings.Contains(streamErr.Error(), "selector contains no outbounds") {
		t.Fatalf("unexpected stream failure: conn=%v err=%v", stream, streamErr)
	}
	packet, packetErr := selector.ListenPacket(context.Background(), M.Socksaddr{})
	if packet != nil || packetErr == nil || !strings.Contains(packetErr.Error(), "selector contains no outbounds") {
		t.Fatalf("unexpected packet failure: conn=%v err=%v", packet, packetErr)
	}
}

type selectorTestOutbound struct {
	boxOutbound.Adapter
}

func (o *selectorTestOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, nil
}

func (o *selectorTestOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, nil
}

type selectorTestOutboundManager struct {
	outbounds map[string]adapter.Outbound
}

func (m *selectorTestOutboundManager) Start(adapter.StartStage, *adapter.Scope) error { return nil }
func (m *selectorTestOutboundManager) Outbounds() []adapter.Outbound {
	result := make([]adapter.Outbound, 0, len(m.outbounds))
	for _, item := range m.outbounds {
		result = append(result, item)
	}
	return result
}

func (m *selectorTestOutboundManager) Outbound(tag string) (adapter.Outbound, bool) {
	item, loaded := m.outbounds[tag]
	return item, loaded
}
func (m *selectorTestOutboundManager) Default() adapter.Outbound { return nil }

func (m *selectorTestOutboundManager) Create(context.Context, adapter.Router, log.ContextLogger, string, string, any) error {
	return nil
}

type selectorTestProvider struct {
	outbounds []adapter.Outbound
}

func (p *selectorTestProvider) Start(adapter.StartStage, *adapter.Scope) error { return nil }
func (p *selectorTestProvider) Type() string                                   { return "selector-test" }
func (p *selectorTestProvider) Tag() string                                    { return "provider" }
func (p *selectorTestProvider) Outbounds() []adapter.Outbound {
	return append([]adapter.Outbound(nil), p.outbounds...)
}
func (p *selectorTestProvider) UpdatedAt() time.Time { return time.Time{} }
func (p *selectorTestProvider) Updating() bool       { return false }
func (p *selectorTestProvider) LastError() error     { return nil }
func (p *selectorTestProvider) SubscriptionInfo() adapter.OutboundProviderSubscriptionInfo {
	return adapter.OutboundProviderSubscriptionInfo{}
}
func (p *selectorTestProvider) Update(context.Context) error { return nil }
func (p *selectorTestProvider) RegisterCallback(adapter.OutboundProviderCallback) *list.Element[adapter.OutboundProviderCallback] {
	return nil
}

func (p *selectorTestProvider) UnregisterCallback(*list.Element[adapter.OutboundProviderCallback]) {}

func newSelectorTestOutbound(tag string) adapter.Outbound {
	return &selectorTestOutbound{
		Adapter: boxOutbound.NewAdapter("selector-test", tag, []string{N.NetworkTCP, N.NetworkUDP}, nil),
	}
}

func TestSelectorRebuildPrefersCachedSelectionAfterProviderLoad(t *testing.T) {
	manual := newSelectorTestOutbound("手动切换")
	providerNode := newSelectorTestOutbound("provider/node")
	selector := &Selector{
		members: &members{
			outbound:   &selectorTestOutboundManager{outbounds: map[string]adapter.Outbound{"手动切换": manual}},
			staticTags: []string{"手动切换"},
			providers:  []adapter.OutboundProvider{&selectorTestProvider{outbounds: []adapter.Outbound{providerNode}}},
		},
		interruptGroup: interrupt.NewGroup(),
	}
	selector.preferredTag.Store(providerNode.Tag())
	selector.selected.Store(manual)

	selector.rebuild()

	if selected := selector.Selected(N.NetworkTCP); selected != providerNode {
		t.Fatalf("cached provider selection was not restored: %v", selected)
	}
}

func TestSelectorRebuildPrefersDefaultOverFallback(t *testing.T) {
	direct := newSelectorTestOutbound("direct")
	providerNode := newSelectorTestOutbound("provider/node")
	selector := &Selector{
		members: &members{
			outbound:   &selectorTestOutboundManager{outbounds: map[string]adapter.Outbound{"direct": direct}},
			staticTags: []string{"direct"},
			providers:  []adapter.OutboundProvider{&selectorTestProvider{outbounds: []adapter.Outbound{providerNode}}},
		},
		defaultTag:     providerNode.Tag(),
		interruptGroup: interrupt.NewGroup(),
	}
	// The provider was not loaded at start, so the first member was picked.
	selector.selected.Store(direct)

	selector.rebuild()

	if selected := selector.Selected(N.NetworkTCP); selected != providerNode {
		t.Fatalf("default provider node was not selected: %v", selected)
	}
}

type selectorTestCacheFile struct {
	adapter.CacheFile
	selected map[string]string
}

func (c *selectorTestCacheFile) LoadSelected(group string) string {
	return c.selected[group]
}

func (c *selectorTestCacheFile) StoreSelected(group, selected string) error {
	c.selected[group] = selected
	return nil
}

func TestSelectorSelectOutboundStoresSameSelection(t *testing.T) {
	manual := newSelectorTestOutbound("手动切换")
	cacheFile := &selectorTestCacheFile{selected: map[string]string{"selector": "provider/node"}}
	ctx := service.ContextWith[adapter.CacheFile](context.Background(), cacheFile)
	selector := &Selector{
		Adapter:        boxOutbound.NewAdapter("selector", "selector", nil, nil),
		ctx:            ctx,
		logger:         log.NewNOPFactory().Logger(),
		interruptGroup: interrupt.NewGroup(),
	}
	selector.state.Store(&selectorState{
		tags:      []string{manual.Tag()},
		outbounds: map[string]adapter.Outbound{manual.Tag(): manual},
	})
	selector.selected.Store(manual)
	selector.preferredTag.Store("provider/node")

	if !selector.SelectOutbound(manual.Tag()) {
		t.Fatal("selecting the current outbound failed")
	}
	if selected := cacheFile.selected[selector.Tag()]; selected != manual.Tag() {
		t.Fatalf("same selection was not stored: %q", selected)
	}
	if preferred := selector.preferredTag.Load(); preferred != manual.Tag() {
		t.Fatalf("same selection did not update preference: %q", preferred)
	}
}
