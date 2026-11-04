package dnsquerylog

import (
	"cmp"
	"context"
	"net/netip"
	"slices"
	"strings"
	"time"
)

const (
	defaultStatsBucket = time.Hour
	minStatsBucket     = time.Second
	maxStatsBuckets    = 1000
	defaultStatsTop    = 10
	maxStatsTop        = 100
)

type StatsOptions struct {
	Filter Filter
	// Bucket is the length of a time bucket; zero picks one hour. It grows
	// when the range would need more than 1000 buckets.
	Bucket time.Duration
	// Top is the number of entries in each ranking; zero picks 10, at most 100.
	Top int
}

// Stats summarizes the entries matching a filter.
type Stats struct {
	Total    int64
	Statuses map[string]int64
	// Methods counts entries by method.
	Methods map[string]int64
	// Bucket is the bucket length used.
	Bucket time.Duration
	// Buckets covers the range from since, or the oldest entry, to until, or
	// now, with empty buckets included. Buckets start at a multiple of the
	// bucket length in Unix time.
	Buckets []StatsBucket
	// TopDomains ranks the domains of exchanges: the DNS queries the core
	// received from clients. Reverse lookups are left out and counted in
	// ReverseQueries, since every address is a domain of its own and periodic
	// lookups of the addresses of a network would crowd out the ranking.
	TopDomains []Count
	// TopInternalDomains ranks the domains of lookups and background queries:
	// the ones the core made itself, such as to resolve the destination or
	// the server of an outbound for a connection, or to refresh the cache.
	// Reverse lookups are left out and counted in InternalReverseQueries.
	TopInternalDomains []Count
	// ReverseQueries and InternalReverseQueries count the exchanges, and the
	// lookups and background queries, of domains under in-addr.arpa or
	// ip6.arpa.
	ReverseQueries         int64
	InternalReverseQueries int64
	// TopBlockedDomains and TopBlockedRules rank the domains of the entries
	// answered by a reject rule, and those rules. Predefined answers, such as
	// hosts or an empty answer to every AAAA query, are not blocks and count
	// only by status.
	TopBlockedDomains []Count
	TopBlockedRules   []RuleCount
	// TopClients ranks the source addresses, without the port, of exchanges:
	// the DNS queries the core received. Lookups are left out, since the core
	// makes them, such as to resolve the server of an outbound for a
	// connection, even when they carry the source of that connection.
	TopClients []Count
	// QueryTypes counts every query type of an entry, so a lookup of A and
	// AAAA counts for both.
	QueryTypes []Count
	Upstreams  []UpstreamStats
	// Duration is of all entries, UpstreamDuration of entries with the
	// upstream status.
	Duration         Duration
	UpstreamDuration Duration
}

type StatsBucket struct {
	// Time is the start of the bucket in Unix milliseconds.
	Time     int64
	Total    int64
	Statuses map[string]int64
}

type Count struct {
	Key   string
	Count int64
}

// RuleCount counts the queries a rule answered. Rules are told apart by
// their conditions and action, since the index of a rule changes with the
// configuration; Index is of the newest entry.
type RuleCount struct {
	Index  int
	Rule   string
	Action string
	Count  int64
}

// UpstreamStats summarizes the exchanges with one DNS server.
type UpstreamStats struct {
	Server string
	Type   string
	// Address and Detour are of the newest exchange, since the configuration
	// may have changed while the log was kept.
	Address   string
	Detour    string
	Exchanges int64
	Failed    int64
	Rejected  int64
	// Duration is of the exchanges that did not fail.
	Duration Duration
}

// Duration is in microseconds.
type Duration struct {
	Average int64
	P95     int64
}

type upstreamAggregate struct {
	stats     UpstreamStats
	durations []int64
}

// Stats summarizes the entries matching the filter of options.
func (s *Store) Stats(ctx context.Context, options StatsOptions) (*Stats, error) {
	bucket := options.Bucket
	if bucket <= 0 {
		bucket = defaultStatsBucket
	}
	bucket = max(bucket, minStatsBucket)
	top := options.Top
	if top <= 0 {
		top = defaultStatsTop
	}
	top = min(top, maxStatsTop)
	var (
		stats             = &Stats{Statuses: make(map[string]int64), Methods: make(map[string]int64)}
		buckets           = make(map[int64]*StatsBucket)
		domains           = make(map[string]int64)
		internalDomains   = make(map[string]int64)
		blockedDomains    = make(map[string]int64)
		rules             = make(map[ruleKey]*RuleCount)
		clients           = make(map[string]int64)
		queryTypes        = make(map[string]int64)
		upstreams         = make(map[string]*upstreamAggregate)
		durations         []int64
		upstreamDurations []int64
		oldest            int64
	)
	bucketMillis := bucket.Milliseconds()
	err := s.walk(ctx, options.Filter, 0, func(entry Entry) bool {
		stats.Total++
		stats.Statuses[entry.Status]++
		stats.Methods[entry.Method]++
		// Entries come in the order they finished, not by Time.
		if oldest == 0 || entry.Time < oldest {
			oldest = entry.Time
		}
		bucketTime := entry.Time - entry.Time%bucketMillis
		item := buckets[bucketTime]
		if item == nil {
			item = &StatsBucket{Time: bucketTime, Statuses: make(map[string]int64)}
			buckets[bucketTime] = item
		}
		item.Total++
		item.Statuses[entry.Status]++
		blocked := entry.Status == StatusRejected && len(entry.Rules) > 0
		if blocked {
			rule := entry.Rules[len(entry.Rules)-1]
			key := ruleKey{rule: rule.Rule, action: rule.Action}
			count := rules[key]
			if count == nil {
				// Entries come newest first.
				count = &RuleCount{Index: rule.Index, Rule: rule.Rule, Action: rule.Action}
				rules[key] = count
			}
			count.Count++
		}
		if entry.Domain != "" {
			reverse := isReverseDomain(entry.Domain)
			switch {
			case entry.Method == MethodExchange && reverse:
				stats.ReverseQueries++
			case entry.Method == MethodExchange:
				domains[entry.Domain]++
			case reverse:
				stats.InternalReverseQueries++
			default:
				internalDomains[entry.Domain]++
			}
			if blocked {
				blockedDomains[entry.Domain]++
			}
		}
		if entry.Method == MethodExchange {
			if client := sourceAddress(entry.Source); client != "" {
				clients[client]++
			}
		}
		for queryType := range strings.SplitSeq(entry.QueryType, ",") {
			if queryType != "" {
				queryTypes[queryType]++
			}
		}
		for _, upstream := range entry.Upstreams {
			aggregate := upstreams[upstream.Server]
			if aggregate == nil {
				// Entries come newest first.
				aggregate = &upstreamAggregate{stats: UpstreamStats{
					Server:  upstream.Server,
					Type:    upstream.Type,
					Address: upstream.Address,
					Detour:  upstream.Detour,
				}}
				upstreams[upstream.Server] = aggregate
			}
			aggregate.stats.Exchanges++
			if upstream.Rejected {
				aggregate.stats.Rejected++
			}
			if upstream.Error != "" {
				aggregate.stats.Failed++
			} else {
				aggregate.durations = append(aggregate.durations, upstream.Duration)
			}
		}
		durations = append(durations, entry.Duration)
		if entry.Status == StatusUpstream {
			upstreamDurations = append(upstreamDurations, entry.Duration)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	stats.Bucket, stats.Buckets = fillBuckets(buckets, bucket, options.Filter, oldest, stats.Total > 0)
	stats.TopDomains = rankCounts(domains, top)
	stats.TopInternalDomains = rankCounts(internalDomains, top)
	stats.TopBlockedDomains = rankCounts(blockedDomains, top)
	stats.TopBlockedRules = rankRules(rules, top)
	stats.TopClients = rankCounts(clients, top)
	stats.QueryTypes = rankCounts(queryTypes, 0)
	for _, aggregate := range upstreams {
		aggregate.stats.Duration = summarizeDurations(aggregate.durations)
		stats.Upstreams = append(stats.Upstreams, aggregate.stats)
	}
	slices.SortFunc(stats.Upstreams, func(a, b UpstreamStats) int {
		return cmp.Or(cmp.Compare(b.Exchanges, a.Exchanges), strings.Compare(a.Server, b.Server))
	})
	stats.Duration = summarizeDurations(durations)
	stats.UpstreamDuration = summarizeDurations(upstreamDurations)
	return stats, nil
}

// fillBuckets lists the buckets of the range in time order, merging them into
// longer ones when there would be too many.
func fillBuckets(buckets map[int64]*StatsBucket, bucket time.Duration, filter Filter, oldest int64, hasEntries bool) (time.Duration, []StatsBucket) {
	start := filter.Since
	if start <= 0 {
		if !hasEntries {
			return bucket, nil
		}
		start = oldest
	}
	end := filter.Until
	if end <= 0 {
		end = time.Now().UnixMilli()
	}
	if end < start {
		return bucket, nil
	}
	bucketMillis := bucket.Milliseconds()
	if count := (end-start)/bucketMillis + 1; count > maxStatsBuckets {
		bucketMillis *= (count + maxStatsBuckets - 1) / maxStatsBuckets
		bucket = time.Duration(bucketMillis) * time.Millisecond
	}
	start -= start % bucketMillis
	result := make([]StatsBucket, 0, (end-start)/bucketMillis+1)
	for bucketTime := start; bucketTime <= end; bucketTime += bucketMillis {
		result = append(result, StatsBucket{Time: bucketTime, Statuses: make(map[string]int64)})
	}
	for _, item := range buckets {
		index := (item.Time - start) / bucketMillis
		if index < 0 || index >= int64(len(result)) {
			continue
		}
		merged := &result[index]
		merged.Total += item.Total
		for status, count := range item.Statuses {
			merged.Statuses[status] += count
		}
	}
	return bucket, result
}

// rankCounts sorts counts by count, then key, keeping the first limit when
// limit is positive.
func rankCounts(counts map[string]int64, limit int) []Count {
	result := make([]Count, 0, len(counts))
	for key, count := range counts {
		result = append(result, Count{Key: key, Count: count})
	}
	slices.SortFunc(result, func(a, b Count) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Key, b.Key))
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

type ruleKey struct {
	rule   string
	action string
}

// rankRules sorts rules by count, then index, keeping the first limit.
func rankRules(rules map[ruleKey]*RuleCount, limit int) []RuleCount {
	result := make([]RuleCount, 0, len(rules))
	for _, rule := range rules {
		result = append(result, *rule)
	}
	slices.SortFunc(result, func(a, b RuleCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Index, b.Index), strings.Compare(a.Rule, b.Rule), strings.Compare(a.Action, b.Action))
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

// summarizeDurations returns the average and the 95th percentile, by the
// nearest rank, of durations, which it sorts.
func summarizeDurations(durations []int64) Duration {
	if len(durations) == 0 {
		return Duration{}
	}
	slices.Sort(durations)
	var sum int64
	for _, duration := range durations {
		sum += duration
	}
	return Duration{
		Average: sum / int64(len(durations)),
		P95:     durations[(len(durations)*95+99)/100-1],
	}
}

// isReverseDomain reports whether domain, without the trailing dot, is under
// in-addr.arpa or ip6.arpa.
func isReverseDomain(domain string) bool {
	for _, zone := range []string{"in-addr.arpa", "ip6.arpa"} {
		if len(domain) >= len(zone) && strings.EqualFold(domain[len(domain)-len(zone):], zone) &&
			(len(domain) == len(zone) || domain[len(domain)-len(zone)-1] == '.') {
			return true
		}
	}
	return false
}

func sourceAddress(source string) string {
	if source == "" {
		return ""
	}
	if addrPort, err := netip.ParseAddrPort(source); err == nil {
		return addrPort.Addr().Unmap().String()
	}
	return source
}
