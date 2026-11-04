package dnsquerylog

import (
	"context"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"

	"github.com/stretchr/testify/require"
)

func addStatsEntries(store *Store, base int64) {
	upstream := func(server string, duration int64, failed bool, rejected bool) Upstream {
		item := Upstream{Server: server, Type: "https", QueryType: "A", Duration: duration, Rejected: rejected}
		if failed {
			item.Error = "timeout"
		}
		return item
	}
	entries := []Entry{
		{Time: base, Domain: "a.test", QueryType: "A", Method: MethodExchange, Source: "192.168.1.2:5000", Status: StatusUpstream, Duration: 100, Upstreams: []Upstream{upstream("remote", 90, false, false)}},
		{Time: base + 1000, Domain: "a.test", QueryType: "AAAA", Method: MethodExchange, Source: "192.168.1.2:5001", Status: StatusCached, Duration: 10},
		{Time: base + 2000, Domain: "b.test", QueryType: "A,AAAA", Method: MethodLookup, Source: "192.168.1.3:6000", Status: StatusUpstream, Duration: 300, Upstreams: []Upstream{upstream("remote", 280, false, true), upstream("local", 20, false, false)}},
		{Time: base + 60_000, Domain: "ads.test", QueryType: "A", Method: MethodExchange, Source: "192.168.1.3:6001", Status: StatusRejected, Duration: 5, Rules: []RuleMatch{{Index: 2, Rule: "domain_suffix=ads.test", Action: "reject", ActionType: "reject"}}},
		{Time: base + 61_000, Domain: "c.test", QueryType: "HTTPS", Method: MethodExchange, Source: "[::ffff:192.168.1.2]:5002", Status: StatusPredefined, Duration: 5, Rules: []RuleMatch{{Index: 0, Rule: "query_type=HTTPS", Action: "predefined(rcode=NOERROR)", ActionType: "predefined", Policy: true}}},
		{Time: base + 120_000, Domain: "d.test", QueryType: "A", Method: MethodBackground, Status: StatusError, Duration: 5000, Upstreams: []Upstream{upstream("remote", 5000, true, false)}},
	}
	for _, entry := range entries {
		store.add(entry)
	}
}

func TestStats(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options option.DNSQueryLogOptions
		flush   bool
	}{
		{name: "memory", options: option.DNSQueryLogOptions{MemoryOnly: true}},
		{name: "persistent", options: option.DNSQueryLogOptions{}, flush: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := newTestStore(t, testCase.options)
			base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
			addStatsEntries(store, base)
			if testCase.flush {
				require.NoError(t, store.flush())
			}
			stats, err := store.Stats(context.Background(), StatsOptions{
				Filter: Filter{Since: base, Until: base + 179_999},
				Bucket: time.Minute,
				Top:    2,
			})
			require.NoError(t, err)
			require.Equal(t, int64(6), stats.Total)
			require.Equal(t, map[string]int64{
				StatusUpstream: 2, StatusCached: 1, StatusRejected: 1, StatusPredefined: 1, StatusError: 1,
			}, stats.Statuses)

			require.Equal(t, time.Minute, stats.Bucket)
			require.Len(t, stats.Buckets, 3)
			require.Equal(t, base, stats.Buckets[0].Time)
			require.Equal(t, []int64{3, 2, 1}, []int64{stats.Buckets[0].Total, stats.Buckets[1].Total, stats.Buckets[2].Total})
			require.Equal(t, int64(2), stats.Buckets[0].Statuses[StatusUpstream])

			require.Equal(t, map[string]int64{MethodExchange: 4, MethodLookup: 1, MethodBackground: 1}, stats.Methods)
			// Domains of lookups and background entries rank on their own.
			require.Equal(t, []Count{{"a.test", 2}, {"ads.test", 1}}, stats.TopDomains)
			require.Equal(t, []Count{{"b.test", 1}, {"d.test", 1}}, stats.TopInternalDomains)
			// The policy rule answers c.test, which counts for the rule only.
			require.Equal(t, []Count{{"ads.test", 1}}, stats.TopBlockedDomains)
			require.Equal(t, []RuleCount{
				{Index: 0, Rule: "query_type=HTTPS", Action: "predefined(rcode=NOERROR)", ActionType: "predefined", Kind: RuleKindPolicy, Count: 1},
				{Index: 2, Rule: "domain_suffix=ads.test", Action: "reject", ActionType: "reject", Kind: RuleKindBlock, Count: 1},
			}, stats.TopRules)
			require.Equal(t, KindCounts{Blocked: 1, Policy: 1}, stats.KindCounts)
			require.Equal(t, KindCounts{Blocked: 1, Policy: 1}, stats.Buckets[1].KindCounts)
			// The IPv4-mapped address counts as the IPv4 one. The lookup,
			// though it has a source, and the background entry are left out.
			require.Equal(t, []Count{{"192.168.1.2", 3}, {"192.168.1.3", 1}}, stats.TopClients)
			require.Equal(t, []Count{{"A", 4}, {"AAAA", 2}, {"HTTPS", 1}}, stats.QueryTypes)

			require.Equal(t, []UpstreamStats{
				{Server: "remote", Type: "https", Exchanges: 3, Failed: 1, Rejected: 1, Duration: Duration{Average: 185, P95: 280}},
				{Server: "local", Type: "https", Exchanges: 1, Duration: Duration{Average: 20, P95: 20}},
			}, stats.Upstreams)
			require.Equal(t, Duration{Average: (100 + 10 + 300 + 5 + 5 + 5000) / 6, P95: 5000}, stats.Duration)
			require.Equal(t, Duration{Average: 200, P95: 300}, stats.UpstreamDuration)
		})
	}
}

func TestStatsFilter(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
	addStatsEntries(store, base)
	stats, err := store.Stats(context.Background(), StatsOptions{Filter: Filter{Status: "UPSTREAM"}})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Total)
	require.Equal(t, []Count{{"a.test", 1}}, stats.TopDomains)
	require.Equal(t, []Count{{"b.test", 1}}, stats.TopInternalDomains)
	// Without since, the buckets start at the oldest entry and run to now.
	require.Equal(t, defaultStatsBucket, stats.Bucket)
	require.Equal(t, base, stats.Buckets[0].Time)
	require.Equal(t, int64(2), stats.Buckets[0].Total)
	require.Greater(t, stats.Buckets[len(stats.Buckets)-1].Time, base)
}

func TestStatsMethodFilter(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
	addStatsEntries(store, base)
	stats, err := store.Stats(context.Background(), StatsOptions{Filter: Filter{Method: "Lookup"}})
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.Total)
	require.Equal(t, map[string]int64{MethodLookup: 1}, stats.Methods)
	require.Empty(t, stats.TopDomains)
	require.Equal(t, []Count{{"b.test", 1}}, stats.TopInternalDomains)
}

func TestStatsBlockRules(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
	policy := RuleMatch{Rule: "query_type=AAAA", Action: "predefined(rcode=NOERROR)", ActionType: "predefined", Policy: true}
	// Entries come newest first; the index of the newest one is kept.
	for i, domain := range []string{"a.test", "b.test", "c.test"} {
		policy.Index = 5 - i
		store.add(Entry{Time: base + int64(i), Domain: domain, QueryType: "AAAA", Method: MethodExchange, Status: StatusPredefined, Rules: []RuleMatch{policy}})
	}
	// Only the final rule counts.
	store.add(Entry{Time: base + 10, Domain: "a.test", QueryType: "A", Method: MethodExchange, Status: StatusRejected, Rules: []RuleMatch{
		{Index: 0, Rule: "inbound=dns-in", Action: "route-options", ActionType: "route-options", Policy: true},
		{Index: 1, Rule: "rule_set=ads", Action: "reject", ActionType: "reject"},
	}})
	stats, err := store.Stats(context.Background(), StatsOptions{})
	require.NoError(t, err)
	require.Equal(t, []Count{{"a.test", 1}}, stats.TopBlockedDomains)
	require.Equal(t, []RuleCount{
		{Index: 3, Rule: "query_type=AAAA", Action: "predefined(rcode=NOERROR)", ActionType: "predefined", Kind: RuleKindPolicy, Count: 3},
		{Index: 1, Rule: "rule_set=ads", Action: "reject", ActionType: "reject", Kind: RuleKindBlock, Count: 1},
	}, stats.TopRules)
	require.Equal(t, KindCounts{Blocked: 1, Policy: 3}, stats.KindCounts)
}

func TestStatsRewrite(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
	hosts := RuleMatch{Index: 0, Rule: "domain=nas.lan", Action: "predefined(NOERROR)", ActionType: "predefined"}
	block := RuleMatch{Index: 1, Rule: "domain_suffix=ads.test", Action: "predefined(NXDOMAIN)", ActionType: "predefined"}
	store.add(Entry{Time: base, Domain: "nas.lan", QueryType: "A", Method: MethodExchange, Status: StatusPredefined, Rules: []RuleMatch{hosts}, Answers: []string{"A 192.168.1.10"}})
	store.add(Entry{Time: base + 1, Domain: "ads.test", QueryType: "A", Method: MethodExchange, Status: StatusPredefined, Rules: []RuleMatch{block}, Rcode: "NXDOMAIN"})
	stats, err := store.Stats(context.Background(), StatsOptions{})
	require.NoError(t, err)
	require.Equal(t, KindCounts{Blocked: 1, Rewritten: 1}, stats.KindCounts)
	require.Equal(t, []Count{{"ads.test", 1}}, stats.TopBlockedDomains)
	require.Equal(t, []RuleCount{
		{Index: 0, Rule: "domain=nas.lan", Action: "predefined(NOERROR)", ActionType: "predefined", Kind: RuleKindRewrite, Count: 1},
		{Index: 1, Rule: "domain_suffix=ads.test", Action: "predefined(NXDOMAIN)", ActionType: "predefined", Kind: RuleKindBlock, Count: 1},
	}, stats.TopRules)
}

func TestStatsBucketLimit(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
	addStatsEntries(store, base)
	stats, err := store.Stats(context.Background(), StatsOptions{
		Filter: Filter{Since: base, Until: base + 2*time.Hour.Milliseconds()},
		Bucket: time.Second,
	})
	require.NoError(t, err)
	require.LessOrEqual(t, len(stats.Buckets), maxStatsBuckets)
	require.Equal(t, 8*time.Second, stats.Bucket)
	var total int64
	for _, item := range stats.Buckets {
		total += item.Total
	}
	require.Equal(t, int64(6), total)
}

func TestStatsEmpty(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	stats, err := store.Stats(context.Background(), StatsOptions{})
	require.NoError(t, err)
	require.Zero(t, stats.Total)
	require.Empty(t, stats.Buckets)
	require.Empty(t, stats.TopDomains)
	require.Equal(t, Duration{}, stats.Duration)
}
