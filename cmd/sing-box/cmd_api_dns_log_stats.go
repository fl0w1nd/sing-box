package main

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sagernet/sing-box/daemon"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	commandAPIDNSLogStatsFlagBucket time.Duration
	commandAPIDNSLogStatsFlagTop    int32
	commandAPIDNSLogStatsFlagJSON   bool
)

var commandAPIDNSLogStats = &cobra.Command{
	Use:   "stats",
	Short: "Summarize the DNS query log",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIDNSLogStats()
	},
}

// dnsLogStatuses is the order statuses are printed in.
var dnsLogStatuses = []string{"upstream", "cached", "optimistic", "fakeip", "rejected", "predefined", "strategy", "local", "error"}

func init() {
	addDNSLogFilterFlags(commandAPIDNSLogStats, "Summarize")
	flags := commandAPIDNSLogStats.Flags()
	flags.DurationVar(&commandAPIDNSLogStatsFlagBucket, "bucket", 0, "Length of a time bucket, one hour by default")
	flags.Int32Var(&commandAPIDNSLogStatsFlagTop, "top", 10, "Entries in each ranking, at most 100")
	flags.BoolVar(&commandAPIDNSLogStatsFlagJSON, "json", false, "Print the summary as JSON")
	commandAPIDNSLog.AddCommand(commandAPIDNSLogStats)
}

func runAPIDNSLogStats() error {
	clientConn, client, err := createExtendedAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	stats, err := client.GetDNSQueryLogStats(globalCtx, &daemon.DNSQueryLogStatsRequest{
		Filter: newDNSLogRequest(),
		Bucket: int64(commandAPIDNSLogStatsFlagBucket / time.Second),
		Top:    commandAPIDNSLogStatsFlagTop,
	})
	if err != nil {
		return err
	}
	if commandAPIDNSLogStatsFlagJSON {
		content, marshalErr := protojson.Marshal(stats)
		if marshalErr != nil {
			return marshalErr
		}
		os.Stdout.Write(append(content, '\n'))
		return nil
	}
	total := stats.GetTotal()
	var block blockWriter
	block.addLine("Total", strconv.FormatInt(total, 10))
	if buckets := stats.GetBuckets(); len(buckets) > 0 {
		bucket := time.Duration(stats.GetBucket()) * time.Second
		block.addLine("Range", formatDNSLogStatsTime(buckets[0].GetTime())+" - "+
			formatDNSLogStatsTime(buckets[len(buckets)-1].GetTime()+bucket.Milliseconds())+", "+bucket.String()+" buckets")
	}
	block.addLine("Duration", formatDNSLogStatsDuration(stats.GetDuration()))
	block.addLine("Upstream duration", formatDNSLogStatsDuration(stats.GetUpstreamDuration()))
	var statuses []string
	for _, status := range dnsLogStatusesOf(stats.GetStatuses()) {
		count := stats.GetStatuses()[status]
		statuses = append(statuses, status+" "+strconv.FormatInt(count, 10)+" ("+formatDNSLogStatsPercent(count, total)+")")
	}
	block.addLine("Statuses", strings.Join(statuses, ", "))
	block.flush()
	if total == 0 {
		return nil
	}
	printDNSLogStatsCounts("Top domains", "DOMAIN", stats.GetTopDomains(), total)
	printDNSLogStatsCounts("Top blocked domains", "DOMAIN", stats.GetTopBlockedDomains(), total)
	printDNSLogStatsCounts("Top clients", "CLIENT", stats.GetTopClients(), total)
	printDNSLogStatsCounts("Query types", "TYPE", stats.GetQueryTypes(), total)

	writeStderrLine("\nUpstreams")
	upstreams := tableWriter{
		header:       []string{"SERVER", "ADDRESS", "DETOUR", "EXCHANGES", "FAILED", "REJECTED", "AVG", "P95"},
		emptyMessage: "no exchanges",
	}
	for _, upstream := range stats.GetUpstreams() {
		// The address starts with the type when the server has one.
		address := upstream.GetAddress()
		if address == "" {
			address = upstream.GetType()
		}
		detour := upstream.GetDetour()
		if detour == "" {
			detour = "-"
		}
		upstreams.addRow(upstream.GetServer(), address, detour,
			strconv.FormatInt(upstream.GetExchanges(), 10),
			strconv.FormatInt(upstream.GetFailed(), 10),
			strconv.FormatInt(upstream.GetRejected(), 10),
			formatDNSLogDuration(upstream.GetDuration().GetAverage()),
			formatDNSLogDuration(upstream.GetDuration().GetP95()))
	}
	upstreams.flush()

	writeStderrLine("\nTimeline")
	columns := dnsLogStatusesOf(stats.GetStatuses())
	timeline := tableWriter{
		header:       append([]string{"TIME", "TOTAL"}, upperStrings(columns)...),
		emptyMessage: "no buckets",
	}
	for _, item := range stats.GetBuckets() {
		row := []string{formatDNSLogStatsTime(item.GetTime()), strconv.FormatInt(item.GetTotal(), 10)}
		for _, status := range columns {
			row = append(row, strconv.FormatInt(item.GetStatuses()[status], 10))
		}
		timeline.addRow(row...)
	}
	timeline.flush()
	return nil
}

func printDNSLogStatsCounts(title string, keyHeader string, counts []*daemon.DNSQueryLogStatsCount, total int64) {
	writeStderrLine("\n" + title)
	table := tableWriter{
		header:       []string{"COUNT", "SHARE", keyHeader},
		emptyMessage: "none",
	}
	for _, count := range counts {
		table.addRow(strconv.FormatInt(count.GetCount(), 10), formatDNSLogStatsPercent(count.GetCount(), total), count.GetKey())
	}
	table.flush()
}

// dnsLogStatusesOf lists the statuses in counts, known ones first.
func dnsLogStatusesOf(counts map[string]int64) []string {
	var statuses []string
	for _, status := range dnsLogStatuses {
		if counts[status] > 0 {
			statuses = append(statuses, status)
		}
	}
	var others []string
	for status := range counts {
		if !slices.Contains(dnsLogStatuses, status) {
			others = append(others, status)
		}
	}
	slices.Sort(others)
	return append(statuses, others...)
}

func upperStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.ToUpper(value))
	}
	return result
}

func formatDNSLogStatsTime(milliseconds int64) string {
	return time.UnixMilli(milliseconds).Format("01-02 15:04:05")
}

func formatDNSLogStatsDuration(duration *daemon.DNSQueryLogDuration) string {
	return "avg " + formatDNSLogDuration(duration.GetAverage()) + ", p95 " + formatDNSLogDuration(duration.GetP95())
}

func formatDNSLogStatsPercent(count int64, total int64) string {
	if total == 0 {
		return "0%"
	}
	return strconv.FormatFloat(float64(count)*100/float64(total), 'f', 1, 64) + "%"
}
