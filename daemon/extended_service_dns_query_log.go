package daemon

import (
	"context"
	"time"

	"github.com/sagernet/sing-box/common/dnsquerylog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *ExtendedService) dnsQueryLog(ctx context.Context) (*dnsquerylog.Store, error) {
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
	return instance.dnsQueryLog, nil
}

func (s *ExtendedService) GetDNSQueryLogStatus(ctx context.Context, _ *emptypb.Empty) (*DNSQueryLogStatus, error) {
	store, err := s.dnsQueryLog(ctx)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return &DNSQueryLogStatus{}, nil
	}
	storeStatus := store.Status()
	return &DNSQueryLogStatus{
		Enabled:        true,
		Persistent:     storeStatus.Persistent,
		Path:           storeStatus.Path,
		Retention:      int64(storeStatus.Retention.Seconds()),
		MaxSize:        storeStatus.MaxSize,
		DiskUsage:      storeStatus.DiskUsage,
		Segments:       int32(storeStatus.Segments),
		MemoryEntries:  int32(storeStatus.MemoryEntries),
		PendingEntries: int32(storeStatus.Pending),
	}, nil
}

func (s *ExtendedService) QueryDNSQueryLog(ctx context.Context, request *DNSQueryLogRequest) (*DNSQueryLogPage, error) {
	store, err := s.dnsQueryLog(ctx)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, status.Error(codes.FailedPrecondition, "DNS query log is disabled")
	}
	entries, next, err := store.Query(newDNSQueryLogFilter(request), request.Before, int(request.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	page := &DNSQueryLogPage{
		Entries: make([]*DNSQueryLogEntry, 0, len(entries)),
		Next:    next,
	}
	for i := range entries {
		page.Entries = append(page.Entries, newDNSQueryLogEntry(&entries[i]))
	}
	return page, nil
}

func (s *ExtendedService) GetDNSQueryLogStats(ctx context.Context, request *DNSQueryLogStatsRequest) (*DNSQueryLogStats, error) {
	store, err := s.dnsQueryLog(ctx)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, status.Error(codes.FailedPrecondition, "DNS query log is disabled")
	}
	options := dnsquerylog.StatsOptions{
		Bucket: time.Duration(request.Bucket) * time.Second,
		Top:    int(request.Top),
	}
	if request.Filter != nil {
		options.Filter = newDNSQueryLogFilter(request.Filter)
	}
	stats, err := store.Stats(options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	message := &DNSQueryLogStats{
		Total:             stats.Total,
		Statuses:          stats.Statuses,
		Bucket:            int64(stats.Bucket / time.Second),
		Buckets:           make([]*DNSQueryLogStatsBucket, 0, len(stats.Buckets)),
		TopDomains:        newDNSQueryLogStatsCounts(stats.TopDomains),
		TopBlockedDomains: newDNSQueryLogStatsCounts(stats.TopBlockedDomains),
		TopClients:        newDNSQueryLogStatsCounts(stats.TopClients),
		QueryTypes:        newDNSQueryLogStatsCounts(stats.QueryTypes),
		Upstreams:         make([]*DNSQueryLogUpstreamStats, 0, len(stats.Upstreams)),
		Duration:          newDNSQueryLogDuration(stats.Duration),
		UpstreamDuration:  newDNSQueryLogDuration(stats.UpstreamDuration),
	}
	for _, item := range stats.Buckets {
		message.Buckets = append(message.Buckets, &DNSQueryLogStatsBucket{
			Time:     item.Time,
			Total:    item.Total,
			Statuses: item.Statuses,
		})
	}
	for _, upstream := range stats.Upstreams {
		message.Upstreams = append(message.Upstreams, &DNSQueryLogUpstreamStats{
			Server:    upstream.Server,
			Type:      upstream.Type,
			Address:   upstream.Address,
			Detour:    upstream.Detour,
			Exchanges: upstream.Exchanges,
			Failed:    upstream.Failed,
			Rejected:  upstream.Rejected,
			Duration:  newDNSQueryLogDuration(upstream.Duration),
		})
	}
	return message, nil
}

func newDNSQueryLogStatsCounts(counts []dnsquerylog.Count) []*DNSQueryLogStatsCount {
	messages := make([]*DNSQueryLogStatsCount, 0, len(counts))
	for _, count := range counts {
		messages = append(messages, &DNSQueryLogStatsCount{Key: count.Key, Count: count.Count})
	}
	return messages
}

func newDNSQueryLogDuration(duration dnsquerylog.Duration) *DNSQueryLogDuration {
	return &DNSQueryLogDuration{Average: duration.Average, P95: duration.P95}
}

func (s *ExtendedService) SubscribeDNSQueryLog(request *DNSQueryLogRequest, server ExtendedService_SubscribeDNSQueryLogServer) error {
	err := s.started.waitForStarted(server.Context())
	if err != nil {
		return err
	}
	filter := newDNSQueryLogFilter(request)
	filter.Prepare()
	return s.started.followInstance(server.Context(), func(ctx context.Context, instance *Instance) error {
		if instance == nil || instance.dnsQueryLog == nil {
			<-ctx.Done()
			return nil
		}
		subscription := instance.dnsQueryLog.Subscribe()
		defer subscription.Close()
		for {
			select {
			case entry, loaded := <-subscription.Entries():
				if !loaded {
					<-ctx.Done()
					return nil
				}
				if !filter.Match(&entry) {
					continue
				}
				err := server.Send(newDNSQueryLogEntry(&entry))
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return nil
			}
		}
	})
}

func (s *ExtendedService) ClearDNSQueryLog(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	store, err := s.dnsQueryLog(ctx)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, status.Error(codes.FailedPrecondition, "DNS query log is disabled")
	}
	err = store.Clear()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func newDNSQueryLogFilter(request *DNSQueryLogRequest) dnsquerylog.Filter {
	return dnsquerylog.Filter{
		Search:    request.Search,
		QueryType: request.QueryType,
		Status:    request.Status,
		Server:    request.Server,
		Inbound:   request.Inbound,
		Since:     request.Since,
		Until:     request.Until,
	}
}

func newDNSQueryLogEntry(entry *dnsquerylog.Entry) *DNSQueryLogEntry {
	message := &DNSQueryLogEntry{
		Id:          entry.ID,
		Time:        entry.Time,
		Duration:    entry.Duration,
		Method:      entry.Method,
		Domain:      entry.Domain,
		QueryType:   entry.QueryType,
		Inbound:     entry.Inbound,
		InboundType: entry.InboundType,
		Source:      entry.Source,
		User:        entry.User,
		Process:     entry.Process,
		Outbound:    entry.Outbound,
		Destination: entry.Destination,
		Cached:      entry.Cached,
		Optimistic:  entry.Optimistic,
		Rejected:    entry.Rejected,
		Status:      entry.Status,
		Rcode:       entry.Rcode,
		Answers:     entry.Answers,
		Ttl:         entry.TTL,
		Error:       entry.Error,
	}
	for _, rule := range entry.Rules {
		message.Rules = append(message.Rules, &DNSQueryLogRule{
			Index:      int32(rule.Index),
			Rule:       rule.Rule,
			Action:     rule.Action,
			ActionType: rule.ActionType,
		})
	}
	for _, upstream := range entry.Upstreams {
		message.Upstreams = append(message.Upstreams, &DNSQueryLogUpstream{
			Server:    upstream.Server,
			Type:      upstream.Type,
			QueryType: upstream.QueryType,
			Duration:  upstream.Duration,
			Rcode:     upstream.Rcode,
			Answers:   upstream.Answers,
			Error:     upstream.Error,
			Rejected:  upstream.Rejected,
			Address:   upstream.Address,
			Detour:    upstream.Detour,
		})
	}
	return message
}
