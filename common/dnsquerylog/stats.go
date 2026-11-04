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
	KindCounts
	// Bucket is the bucket length used.
	Bucket time.Duration
	// Buckets covers the range from since, or the oldest entry, to until, or
	// now, with empty buckets included. Buckets start at a multiple of the
	// bucket length in Unix time.
	Buckets []StatsBucket
	// TopDomains ranks the domains of exchanges: the DNS queries the core
	// received from clients.
	TopDomains []Count
	// TopInternalDomains ranks the domains of lookups and background queries:
	// the ones the core made itself, such as to resolve the destination or
	// the server of an outbound for a connection, or to refresh the cache.
	TopInternalDomains []Count
	// TopBlockedDomains ranks the domains of blocked entries. Policy rules,
	// such as one that answers every AAAA query, apply to every domain and
	// are left to TopRules, as are rewrites.
	TopBlockedDomains []Count
	// TopRules ranks the reject and predefined rules by the queries they
	// answered, of every kind.
	TopRules []RuleCount
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
	KindCounts
}

// KindCounts splits the entries answered by a reject or predefined rule by
// the kind of the rule.
type KindCounts struct {
	Blocked   int64
	Policy    int64
	Rewritten int64
}

func (c *KindCounts) add(kind string) {
	switch kind {
	case RuleKindBlock:
		c.Blocked++
	case RuleKindPolicy:
		c.Policy++
	case RuleKindRewrite:
		c.Rewritten++
	}
}

const (
	// RuleKindBlock is a reject rule, or a predefined rule answering without
	// records, that matched by domain, IP CIDR or rule-set.
	RuleKindBlock = "block"
	// RuleKindPolicy is a reject or predefined rule without such a condition,
	// such as one answering every AAAA query.
	RuleKindPolicy = "policy"
	// RuleKindRewrite is a predefined rule answering with records, like
	// hosts.
	RuleKindRewrite = "rewrite"
)

// answeringRule returns the reject or predefined rule that answered entry,
// and its kind, or nil when the entry was not answered by one.
func answeringRule(entry *Entry) (*RuleMatch, string) {
	if (entry.Status != StatusRejected && entry.Status != StatusPredefined) || len(entry.Rules) == 0 {
		return nil, ""
	}
	rule := &entry.Rules[len(entry.Rules)-1]
	switch {
	case rule.Policy:
		return rule, RuleKindPolicy
	case entry.Status == StatusPredefined && len(entry.Answers) > 0:
		return rule, RuleKindRewrite
	default:
		return rule, RuleKindBlock
	}
}

type Count struct {
	Key   string
	Count int64
}

// RuleCount counts the queries a rule answered. Rules are told apart by
// their conditions and action, since the index of a rule changes with the
// configuration; Index is of the newest entry.
type RuleCount struct {
	Index      int
	Rule       string
	Action     string
	ActionType string
	// Kind is of the newest entry.
	Kind  string
	Count int64
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
		rule, kind := answeringRule(&entry)
		if rule != nil {
			key := ruleKey{rule: rule.Rule, action: rule.Action}
			count := rules[key]
			if count == nil {
				// Entries come newest first.
				count = &RuleCount{
					Index:      rule.Index,
					Rule:       rule.Rule,
					Action:     rule.Action,
					ActionType: rule.ActionType,
					Kind:       kind,
				}
				rules[key] = count
			}
			count.Count++
			stats.add(kind)
			item.add(kind)
		}
		if entry.Domain != "" {
			if entry.Method == MethodExchange {
				domains[entry.Domain]++
			} else {
				internalDomains[entry.Domain]++
			}
			if kind == RuleKindBlock {
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
	stats.TopRules = rankRules(rules, top)
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
		merged.Blocked += item.Blocked
		merged.Policy += item.Policy
		merged.Rewritten += item.Rewritten
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

func sourceAddress(source string) string {
	if source == "" {
		return ""
	}
	if addrPort, err := netip.ParseAddrPort(source); err == nil {
		return addrPort.Addr().Unmap().String()
	}
	return source
}
