package daemon

import (
	"context"
	"errors"
	"time"

	"github.com/sagernet/sing-box/adapter"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	OutboundProviderExtensionVersion    = 1
	TailscaleForceRelayExtensionVersion = 1
)

type ExtendedService struct {
	UnimplementedExtendedServiceServer
	started *StartedService
}

func NewExtendedService(started *StartedService) *ExtendedService {
	return &ExtendedService{started: started}
}

func (s *ExtendedService) GetCapabilities(context.Context, *emptypb.Empty) (*ExtensionCapabilities, error) {
	capabilities := []*ExtensionCapability{
		{Name: "outbound_provider", Version: OutboundProviderExtensionVersion},
	}
	if withTailscale {
		capabilities = append(capabilities, &ExtensionCapability{Name: "tailscale_force_relay", Version: TailscaleForceRelayExtensionVersion})
	}
	return &ExtensionCapabilities{Capabilities: capabilities}, nil
}

func (s *ExtendedService) SetTailscaleForceRelay(ctx context.Context, request *SetTailscaleForceRelayRequest) (*emptypb.Empty, error) {
	err := s.started.waitForStarted(ctx)
	if err != nil {
		return nil, err
	}
	s.started.serviceAccess.RLock()
	instance := s.started.instance
	s.started.serviceAccess.RUnlock()
	if instance == nil {
		return nil, status.Error(codes.Unavailable, "service is unavailable")
	}
	err = setTailscaleForceRelay(instance.cacheFile, request.Enabled)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *ExtendedService) SubscribeTailscaleForceRelayStatus(
	_ *emptypb.Empty,
	server ExtendedService_SubscribeTailscaleForceRelayStatusServer,
) error {
	var sendErr error
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()
	err := subscribeTailscaleForceRelay(ctx, func(enabled bool) {
		sendErr = server.Send(&TailscaleForceRelayStatus{Enabled: enabled})
		if sendErr != nil {
			cancel()
		}
	})
	if sendErr != nil {
		return sendErr
	}
	return err
}

func (s *ExtendedService) SubscribeOutboundProviders(_ *emptypb.Empty, server ExtendedService_SubscribeOutboundProvidersServer) error {
	err := s.started.waitForStarted(server.Context())
	if err != nil {
		return err
	}
	return s.started.followInstance(server.Context(), func(ctx context.Context, instance *Instance) error {
		var providers []adapter.OutboundProvider
		if instance != nil {
			providers = instance.providerManager.Providers()
		}
		updated := make(chan struct{}, 1)
		updated <- struct{}{}
		for _, provider := range providers {
			element := provider.RegisterCallback(func() {
				select {
				case updated <- struct{}{}:
				default:
				}
			})
			defer provider.UnregisterCallback(element)
		}
		// Items carry URL test delays, so history updates are pushed too.
		urlTestUpdates, urlTestDone, err := s.started.urlTestObserver.Subscribe()
		if err != nil {
			return err
		}
		defer s.started.urlTestObserver.UnSubscribe(urlTestUpdates)
		for {
			select {
			case <-updated:
			case <-urlTestUpdates:
			case <-urlTestDone:
				return nil
			case <-ctx.Done():
				return nil
			}
			err = server.Send(buildProviderStatusUpdate(providers, instance))
			if err != nil {
				return err
			}
			// A URL test round updates history once per node.
			throttle := time.NewTimer(urlTestPushMinInterval)
			select {
			case <-throttle.C:
			case <-ctx.Done():
				throttle.Stop()
				return nil
			}
		}
	})
}

func (s *ExtendedService) UpdateOutboundProvider(ctx context.Context, request *OutboundProviderRequest) (*emptypb.Empty, error) {
	// Providers have no HTTP transport until they start.
	err := s.started.waitForStarted(ctx)
	if err != nil {
		return nil, err
	}
	s.started.serviceAccess.RLock()
	instance := s.started.instance
	s.started.serviceAccess.RUnlock()
	if instance == nil {
		return nil, status.Error(codes.Unavailable, "service is unavailable")
	}
	provider, loaded := instance.providerManager.Get(request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "outbound provider not found: "+request.Tag)
	}
	err = provider.Update(ctx)
	if errors.Is(err, adapter.ErrProviderUpdating) {
		return nil, status.Error(codes.Aborted, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func buildProviderStatusUpdate(providers []adapter.OutboundProvider, instance *Instance) *OutboundProviderStatusUpdate {
	update := &OutboundProviderStatusUpdate{Providers: make([]*OutboundProviderStatus, 0, len(providers))}
	for _, provider := range providers {
		info := provider.SubscriptionInfo()
		providerStatus := &OutboundProviderStatus{
			Tag:      provider.Tag(),
			Type:     provider.Type(),
			Updating: provider.Updating(),
			SubscriptionInfo: &OutboundProviderSubscriptionInfo{
				Upload: info.Upload, Download: info.Download, Total: info.Total,
			},
		}
		if updatedAt := provider.UpdatedAt(); !updatedAt.IsZero() {
			providerStatus.UpdatedAt = updatedAt.Unix()
		}
		if !info.ExpireTime.IsZero() {
			providerStatus.SubscriptionInfo.ExpireTime = info.ExpireTime.Unix()
		}
		if lastError := provider.LastError(); lastError != nil {
			providerStatus.LastError = lastError.Error()
		}
		for _, outbound := range provider.Outbounds() {
			item := &OutboundProviderItem{Tag: outbound.Tag(), Type: outbound.Type()}
			if history := instance.urlTestHistoryStorage.LoadURLTestHistory(outbound.Tag()); history != nil {
				item.UrlTestTime = history.Time.Unix()
				item.UrlTestDelay = int32(history.Delay)
			}
			providerStatus.Items = append(providerStatus.Items, item)
		}
		update.Providers = append(update.Providers, providerStatus)
	}
	return update
}

var _ ExtendedServiceServer = (*ExtendedService)(nil)
