package outbound

import (
	"context"
	"net"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/endpoint"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type dynamicTestOutbound struct {
	Adapter
	starts int
}

func (o *dynamicTestOutbound) Start(adapter.StartStage, *adapter.Scope) error {
	o.starts++
	return nil
}

func (o *dynamicTestOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, nil
}

func (o *dynamicTestOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, nil
}

func newDynamicTestOutbound(tag string) *dynamicTestOutbound {
	return &dynamicTestOutbound{Adapter: NewAdapter("test", tag, []string{N.NetworkTCP}, nil)}
}

func TestSwapDynamic(t *testing.T) {
	manager := NewManager(NewRegistry(), endpoint.NewManager(endpoint.NewRegistry()), "")
	static := newDynamicTestOutbound("static")
	manager.outbounds = append(manager.outbounds, static)
	manager.outboundByTag["static"] = static

	if err := manager.SwapDynamic("static", nil, newDynamicTestOutbound("static")); err == nil {
		t.Fatal("dynamic outbound replaced a static tag")
	}
	if err := manager.SwapDynamic("static", static, newDynamicTestOutbound("static")); err == nil {
		t.Fatal("static outbound was swapped as dynamic")
	}
	first := newDynamicTestOutbound("node")
	if err := manager.SwapDynamic("node", nil, first); err != nil {
		t.Fatal(err)
	}
	if err := manager.SwapDynamic("node", nil, newDynamicTestOutbound("node")); err == nil {
		t.Fatal("registered dynamic tag was taken over")
	}
	published := manager.Outbounds()
	second := newDynamicTestOutbound("node")
	if err := manager.SwapDynamic("node", first, second); err != nil {
		t.Fatal(err)
	}
	if published[1] != first {
		t.Fatal("published outbound list was modified in place")
	}
	if current, _ := manager.Outbound("node"); current != second {
		t.Fatal("replacement is not registered")
	}
	if err := manager.SwapDynamic("node", first, nil); err == nil {
		t.Fatal("stale outbound removed its replacement")
	}

	scope := adapter.NewScope(context.Background(), nil)
	for _, stage := range []adapter.StartStage{adapter.StartStatePostStart, adapter.StartStateStarted} {
		if err := manager.Start(stage, scope); err != nil {
			t.Fatal(err)
		}
	}
	if static.starts != 2 || second.starts != 0 {
		t.Fatalf("manager started dynamic outbound: static=%d dynamic=%d", static.starts, second.starts)
	}

	if err := manager.SwapDynamic("node", second, nil); err != nil {
		t.Fatal(err)
	}
	if _, loaded := manager.Outbound("node"); loaded || len(manager.Outbounds()) != 1 {
		t.Fatal("dynamic outbound was not removed")
	}
}
