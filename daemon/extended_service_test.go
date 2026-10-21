package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing/common/x/list"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestExtensionCapabilitiesKeepStartedAPIVersion(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := NewStartedService(ServiceOptions{Context: ctx})
	version, err := started.GetVersion(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if version.ApiVersion != APIVersion {
		t.Fatalf("unexpected StartedService API version: %d", version.ApiVersion)
	}
	capabilities, err := NewExtendedService(started).GetCapabilities(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	versions := make(map[string]int32)
	for _, capability := range capabilities.Capabilities {
		versions[capability.Name] = capability.Version
	}
	if versions["outbound_provider"] != 1 {
		t.Fatalf("unexpected extension capabilities: %+v", versions)
	}
}

type statusTestOutbound struct{ adapter.Outbound }

func (statusTestOutbound) Tag() string  { return "provider/node" }
func (statusTestOutbound) Type() string { return "test" }

type statusTestProvider struct{ adapter.OutboundProvider }

func (statusTestProvider) Tag() string          { return "provider" }
func (statusTestProvider) Type() string         { return "remote" }
func (statusTestProvider) Updating() bool       { return false }
func (statusTestProvider) UpdatedAt() time.Time { return time.Time{} }
func (statusTestProvider) LastError() error     { return nil }
func (statusTestProvider) Outbounds() []adapter.Outbound {
	return []adapter.Outbound{statusTestOutbound{}}
}
func (statusTestProvider) UnregisterCallback(*list.Element[adapter.OutboundProviderCallback]) {}
func (statusTestProvider) RegisterCallback(adapter.OutboundProviderCallback) *list.Element[adapter.OutboundProviderCallback] {
	return nil
}

func (statusTestProvider) SubscriptionInfo() adapter.OutboundProviderSubscriptionInfo {
	return adapter.OutboundProviderSubscriptionInfo{}
}

type statusTestManager struct{ adapter.ProviderManager }

func (statusTestManager) Providers() []adapter.OutboundProvider {
	return []adapter.OutboundProvider{statusTestProvider{}}
}

type statusTestServer struct {
	grpc.ServerStream
	ctx     context.Context
	updates chan *OutboundProviderStatusUpdate
}

func (s statusTestServer) Context() context.Context { return s.ctx }
func (s statusTestServer) Send(update *OutboundProviderStatusUpdate) error {
	s.updates <- update
	return nil
}

func TestSubscribeOutboundProvidersPushesURLTestResults(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := NewStartedService(ServiceOptions{Context: ctx})
	history := urltest.NewHistoryStorage()
	history.AddUpdateHook(started.urlTestSubscriber)
	started.instance = &Instance{providerManager: statusTestManager{}, urlTestHistoryStorage: history}
	started.serviceStatus = &ServiceStatus{Status: ServiceStatus_STARTED}
	updates := make(chan *OutboundProviderStatusUpdate, 10)
	done := make(chan error, 1)
	go func() {
		done <- NewExtendedService(started).SubscribeOutboundProviders(&emptypb.Empty{}, statusTestServer{ctx: ctx, updates: updates})
	}()
	<-updates
	deadline := time.After(5 * time.Second)
	for {
		// Keep storing until the stream is subscribed to history updates.
		history.StoreURLTestHistory("provider/node", &adapter.URLTestHistory{Time: time.Now(), Delay: 99})
		select {
		case update := <-updates:
			if update.Providers[0].Items[0].UrlTestDelay != 99 {
				t.Fatalf("unexpected delay: %d", update.Providers[0].Items[0].UrlTestDelay)
			}
			cancel()
			<-done
			return
		case <-time.After(100 * time.Millisecond):
		case <-deadline:
			t.Fatal("URL test result was not pushed")
		}
	}
}
