package dnsquerylog

import (
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
		{Time: base + 60_000, Domain: "ads.test", QueryType: "A", Method: MethodExchange, Source: "192.168.1.3:6001", Status: StatusRejected, Duration: 5},
		{Time: base + 61_000, Domain: "c.test", QueryType: "HTTPS", Method: MethodExchange, Source: "[::ffff:192.168.1.2]:5002", Status: StatusPredefined, Duration: 5},
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
			stats, err := store.Stats(StatsOptions{
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

			require.Equal(t, []Count{{"a.test", 2}, {"ads.test", 1}}, stats.TopDomains)
			require.Equal(t, []Count{{"ads.test", 1}, {"c.test", 1}}, stats.TopBlockedDomains)
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
	stats, err := store.Stats(StatsOptions{Filter: Filter{Status: "UPSTREAM"}})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Total)
	require.Equal(t, []Count{{"a.test", 1}, {"b.test", 1}}, stats.TopDomains)
	// Without since, the buckets start at the oldest entry and run to now.
	require.Equal(t, defaultStatsBucket, stats.Bucket)
	require.Equal(t, base, stats.Buckets[0].Time)
	require.Equal(t, int64(2), stats.Buckets[0].Total)
	require.Greater(t, stats.Buckets[len(stats.Buckets)-1].Time, base)
}

func TestStatsBucketLimit(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	base := time.Now().Add(-time.Hour).Truncate(time.Hour).UnixMilli()
	addStatsEntries(store, base)
	stats, err := store.Stats(StatsOptions{
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
	stats, err := store.Stats(StatsOptions{})
	require.NoError(t, err)
	require.Zero(t, stats.Total)
	require.Empty(t, stats.Buckets)
	require.Empty(t, stats.TopDomains)
	require.Equal(t, Duration{}, stats.Duration)
}
