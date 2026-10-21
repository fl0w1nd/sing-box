package group

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	boxOutbound "github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/interrupt"
	"github.com/sagernet/sing-box/common/urltest"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type providerURLTestOutbound struct {
	boxOutbound.Adapter
}

type trackingCloser struct {
	closed bool
}

func (c *trackingCloser) Close() error {
	c.closed = true
	return nil
}

func (o *providerURLTestOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	panic("unexpected URL test")
}

func (o *providerURLTestOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	panic("unexpected packet listen")
}

func TestReplaceProviderOutboundsUsesExistingHealthResults(t *testing.T) {
	first := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/first", []string{N.NetworkTCP, N.NetworkUDP}, nil)}
	second := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/second", []string{N.NetworkTCP, N.NetworkUDP}, nil)}
	history := urltest.NewHistoryStorage()
	now := time.Now()
	history.StoreURLTestHistory(first.Tag(), &adapter.URLTestHistory{Time: now, Delay: 100})
	history.StoreURLTestHistory(second.Tag(), &adapter.URLTestHistory{Time: now, Delay: 10})
	group := &URLTestGroup{
		history:        history,
		interruptGroup: interrupt.NewGroup(),
	}
	outbounds := []adapter.Outbound{first, second}
	group.outbounds.Store(outbounds)
	group.selectedOutboundTCP.Store(first)
	group.selectedOutboundUDP.Store(first)

	group.ReplaceOutbounds(outbounds)

	if selected := group.selectedOutboundTCP.Load(); selected != second {
		t.Fatalf("unexpected TCP selection: %v", selected)
	}
	if selected := group.selectedOutboundUDP.Load(); selected != second {
		t.Fatalf("unexpected UDP selection: %v", selected)
	}
}

func TestReplaceProviderOutboundsInterruptsRemovedSelection(t *testing.T) {
	first := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/first", []string{N.NetworkTCP, N.NetworkUDP}, nil)}
	second := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/second", []string{N.NetworkTCP, N.NetworkUDP}, nil)}
	group := &URLTestGroup{
		history:                      urltest.NewHistoryStorage(),
		interruptGroup:               interrupt.NewGroup(),
		interruptExternalConnections: true,
	}
	group.outbounds.Store([]adapter.Outbound{first, second})
	group.selectedOutboundTCP.Store(first)
	group.selectedOutboundUDP.Store(first)
	connection := &trackingCloser{}
	group.interruptGroup.Add(connection, true)

	group.ReplaceOutbounds([]adapter.Outbound{second})

	if !connection.closed {
		t.Fatal("removed selected outbound left an existing connection open")
	}
	if group.selectedOutboundTCP.Load() != second || group.selectedOutboundUDP.Load() != second {
		t.Fatalf("removed outbound remains selected: tcp=%v udp=%v", group.selectedOutboundTCP.Load(), group.selectedOutboundUDP.Load())
	}
}

func TestReplaceProviderOutboundsDropsSelectionWithoutNetwork(t *testing.T) {
	first := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/first", []string{N.NetworkTCP, N.NetworkUDP}, nil)}
	replacement := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/first", []string{N.NetworkTCP}, nil)}
	second := &providerURLTestOutbound{Adapter: boxOutbound.NewAdapter("provider-test", "provider/second", []string{N.NetworkTCP, N.NetworkUDP}, nil)}
	history := urltest.NewHistoryStorage()
	now := time.Now()
	history.StoreURLTestHistory(first.Tag(), &adapter.URLTestHistory{Time: now, Delay: 10})
	history.StoreURLTestHistory(second.Tag(), &adapter.URLTestHistory{Time: now, Delay: 100})
	group := &URLTestGroup{
		history:        history,
		interruptGroup: interrupt.NewGroup(),
	}
	group.outbounds.Store([]adapter.Outbound{first, second})
	group.selectedOutboundTCP.Store(first)
	group.selectedOutboundUDP.Store(first)

	group.ReplaceOutbounds([]adapter.Outbound{replacement, second})

	if selected := group.selectedOutboundTCP.Load(); selected != replacement {
		t.Fatalf("unexpected TCP selection: %v", selected)
	}
	if selected := group.selectedOutboundUDP.Load(); selected != second {
		t.Fatalf("UDP selection kept a TCP-only replacement: %v", selected)
	}
}
