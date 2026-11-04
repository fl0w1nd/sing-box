package daemon

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const DNSCacheExtensionVersion = 1

// ClearDNSCache clears the DNS cache as the Clash API /cache/dns/flush does:
// the memory cache, including the optimistic cache, the cache file, the
// platform and system DNS caches, and the reverse mapping.
func (s *ExtendedService) ClearDNSCache(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
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
	dnsRouter := service.FromContext[adapter.DNSRouter](instance.ctx)
	if dnsRouter == nil {
		return nil, status.Error(codes.Unavailable, "DNS router is unavailable")
	}
	dnsRouter.ClearCache()
	return &emptypb.Empty{}, nil
}
