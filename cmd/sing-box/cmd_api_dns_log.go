package main

import (
	"context"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing/common/byteformats"
	E "github.com/sagernet/sing/common/exceptions"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	commandAPIDNSLogFlagSearch  string
	commandAPIDNSLogFlagType    string
	commandAPIDNSLogFlagStatus  string
	commandAPIDNSLogFlagServer  string
	commandAPIDNSLogFlagInbound string
	commandAPIDNSLogFlagSince   time.Duration
	commandAPIDNSLogFlagLimit   int32
	commandAPIDNSLogFlagBefore  uint64
	commandAPIDNSLogFlagFollow  bool
	commandAPIDNSLogFlagJSON    bool
)

var commandAPIDNSLog = &cobra.Command{
	Use:   "dns-log",
	Short: "Print the DNS query log",
	Long: "Print the latest entries of the DNS query log.\n\n" +
		"Statuses: upstream, cached, optimistic, fakeip, rejected, predefined, strategy, local, error.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIDNSLog()
	},
}

var commandAPIDNSLogStatus = &cobra.Command{
	Use:   "status",
	Short: "Show the DNS query log status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIDNSLogStatus()
	},
}

var commandAPIDNSLogClear = &cobra.Command{
	Use:   "clear",
	Short: "Remove all DNS query log entries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIDNSLogClear()
	},
}

func init() {
	flags := commandAPIDNSLog.Flags()
	addDNSLogFilterFlags(commandAPIDNSLog, "Print")
	flags.Int32VarP(&commandAPIDNSLogFlagLimit, "limit", "n", 50, "Print at most this many entries")
	flags.Uint64Var(&commandAPIDNSLogFlagBefore, "before", 0, "Print entries before this ID, for the next page")
	flags.BoolVarP(&commandAPIDNSLogFlagFollow, "follow", "f", false, "Keep printing new entries until interrupted")
	flags.BoolVar(&commandAPIDNSLogFlagJSON, "json", false, "Print entries as JSON lines")
	commandAPIDNSLog.AddCommand(commandAPIDNSLogStatus, commandAPIDNSLogClear)
	commandAPIRoot.AddCommand(commandAPIDNSLog)
}

// addDNSLogFilterFlags adds the flags that select entries to command,
// described with verb, such as Print.
func addDNSLogFilterFlags(command *cobra.Command, verb string) {
	flags := command.Flags()
	flags.StringVar(&commandAPIDNSLogFlagSearch, "search", "", verb+" entries whose domain, client, process or answers contain this text, case-insensitive")
	flags.StringVar(&commandAPIDNSLogFlagType, "type", "", verb+" entries of this query type, such as A or AAAA")
	flags.StringVar(&commandAPIDNSLogFlagStatus, "status", "", verb+" entries with this status")
	flags.StringVar(&commandAPIDNSLogFlagServer, "server", "", verb+" entries exchanged with this DNS server tag")
	flags.StringVar(&commandAPIDNSLogFlagInbound, "inbound", "", verb+" entries from this inbound tag")
	flags.DurationVar(&commandAPIDNSLogFlagSince, "since", 0, verb+" entries from this long ago, such as 10m")
}

func createExtendedAPIClient() (*grpc.ClientConn, daemon.ExtendedServiceClient, error) {
	clientConn, _, err := createAPIClient()
	if err != nil {
		return nil, nil, err
	}
	return clientConn, daemon.NewExtendedServiceClient(clientConn), nil
}

func newDNSLogRequest() *daemon.DNSQueryLogRequest {
	request := &daemon.DNSQueryLogRequest{
		Search:    commandAPIDNSLogFlagSearch,
		QueryType: commandAPIDNSLogFlagType,
		Status:    commandAPIDNSLogFlagStatus,
		Server:    commandAPIDNSLogFlagServer,
		Inbound:   commandAPIDNSLogFlagInbound,
		Before:    commandAPIDNSLogFlagBefore,
		Limit:     commandAPIDNSLogFlagLimit,
	}
	if commandAPIDNSLogFlagSince > 0 {
		request.Since = time.Now().Add(-commandAPIDNSLogFlagSince).UnixMilli()
	}
	return request
}

func runAPIDNSLog() error {
	clientConn, client, err := createExtendedAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	ctx, cancel := signal.NotifyContext(globalCtx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	request := newDNSLogRequest()
	if commandAPIDNSLogFlagFollow {
		return followAPIDNSLog(ctx, client, request)
	}
	page, err := client.QueryDNSQueryLog(ctx, request)
	if err != nil {
		return err
	}
	if commandAPIDNSLogFlagJSON {
		var output strings.Builder
		for _, entry := range page.GetEntries() {
			content, marshalErr := protojson.Marshal(entry)
			if marshalErr != nil {
				return marshalErr
			}
			output.Write(content)
			output.WriteString("\n")
		}
		os.Stdout.WriteString(output.String())
	} else {
		table := tableWriter{
			header:       dnsLogHeader,
			emptyMessage: "no DNS queries",
		}
		entries := page.GetEntries()
		// Oldest first, so the newest entry is next to the prompt.
		for _, entry := range slices.Backward(entries) {
			table.addRow(formatDNSLogRow(entry)...)
		}
		table.flush()
	}
	if page.GetNext() != 0 {
		writeStderrLine("more entries: --before " + strconv.FormatUint(page.GetNext(), 10))
	}
	return nil
}

func followAPIDNSLog(ctx context.Context, client daemon.ExtendedServiceClient, request *daemon.DNSQueryLogRequest) error {
	stream, err := client.SubscribeDNSQueryLog(ctx, request)
	if err != nil {
		return err
	}
	if !commandAPIDNSLogFlagJSON {
		writeStderrLine(strings.Join(dnsLogHeader, "  "))
	}
	for {
		entry, recvErr := stream.Recv()
		if recvErr != nil {
			if ctx.Err() != nil {
				return nil
			}
			return recvErr
		}
		if commandAPIDNSLogFlagJSON {
			content, marshalErr := protojson.Marshal(entry)
			if marshalErr != nil {
				return marshalErr
			}
			os.Stdout.Write(append(content, '\n'))
			continue
		}
		separator := "  "
		if !stdoutIsTerminal {
			separator = "\t"
		}
		os.Stdout.WriteString(strings.Join(formatDNSLogRow(entry), separator) + "\n")
	}
}

var dnsLogHeader = []string{"ID", "TIME", "CLIENT", "TYPE", "DOMAIN", "STATUS", "SERVER", "DURATION", "ANSWER"}

func formatDNSLogRow(entry *daemon.DNSQueryLogEntry) []string {
	return []string{
		strconv.FormatUint(entry.GetId(), 10),
		formatDNSLogTime(entry.GetTime()),
		formatDNSLogClient(entry),
		entry.GetQueryType(),
		entry.GetDomain(),
		entry.GetStatus(),
		formatDNSLogServer(entry),
		formatDNSLogDuration(entry.GetDuration()),
		formatDNSLogAnswer(entry),
	}
}

func formatDNSLogTime(unixMilli int64) string {
	queryTime := time.UnixMilli(unixMilli)
	now := time.Now()
	if queryTime.Year() == now.Year() && queryTime.YearDay() == now.YearDay() {
		return queryTime.Format("15:04:05")
	}
	return queryTime.Format("01-02 15:04:05")
}

func formatDNSLogClient(entry *daemon.DNSQueryLogEntry) string {
	if entry.GetProcess() != "" {
		process := entry.GetProcess()
		if index := strings.LastIndexAny(process, `/\`); index >= 0 {
			process = process[index+1:]
		}
		return process
	}
	if source := entry.GetSource(); source != "" {
		if index := strings.LastIndex(source, ":"); index > 0 {
			return strings.Trim(source[:index], "[]")
		}
		return source
	}
	if entry.GetOutbound() != "" {
		return "outbound/" + entry.GetOutbound()
	}
	return entry.GetMethod()
}

func formatDNSLogServer(entry *daemon.DNSQueryLogEntry) string {
	upstreams := entry.GetUpstreams()
	for _, upstream := range slices.Backward(upstreams) {
		if upstream.GetError() == "" && !upstream.GetRejected() {
			return upstream.GetServer()
		}
	}
	if len(upstreams) > 0 {
		return upstreams[len(upstreams)-1].GetServer()
	}
	if rules := entry.GetRules(); len(rules) > 0 {
		return rules[len(rules)-1].GetAction()
	}
	return ""
}

func formatDNSLogDuration(microseconds int64) string {
	if microseconds < 1000 {
		return "<1ms"
	}
	return strconv.FormatInt(microseconds/1000, 10) + "ms"
}

func formatDNSLogAnswer(entry *daemon.DNSQueryLogEntry) string {
	if entry.GetError() != "" {
		return entry.GetError()
	}
	answers := entry.GetAnswers()
	if len(answers) == 0 {
		return entry.GetRcode()
	}
	answer := answers[0]
	if len(answers) > 1 {
		answer += " (+" + strconv.Itoa(len(answers)-1) + ")"
	}
	return answer
}

func runAPIDNSLogStatus() error {
	clientConn, client, err := createExtendedAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	queryLogStatus, err := client.GetDNSQueryLogStatus(globalCtx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	var block blockWriter
	if !queryLogStatus.GetEnabled() {
		block.addLine("Enabled", "false")
		block.flush()
		return nil
	}
	block.addLine("Enabled", "true")
	block.addLine("Persistent", strconv.FormatBool(queryLogStatus.GetPersistent()))
	if queryLogStatus.GetPersistent() {
		block.addLine("Path", queryLogStatus.GetPath())
		block.addLine("Retention", (time.Duration(queryLogStatus.GetRetention()) * time.Second).String())
		block.addLine("Max size", byteformats.FormatMemoryBytes(uint64(queryLogStatus.GetMaxSize())))
		block.addLine("Disk usage", byteformats.FormatMemoryBytes(uint64(queryLogStatus.GetDiskUsage())))
		block.addLine("Segments", strconv.Itoa(int(queryLogStatus.GetSegments())))
		block.addLine("Pending", strconv.Itoa(int(queryLogStatus.GetPendingEntries())))
	}
	block.addLine("In memory", strconv.Itoa(int(queryLogStatus.GetMemoryEntries())))
	block.flush()
	return nil
}

func runAPIDNSLogClear() error {
	clientConn, client, err := createExtendedAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	_, err = client.ClearDNSQueryLog(globalCtx, &emptypb.Empty{})
	if err != nil {
		return E.Cause(err, "clear DNS query log")
	}
	writeStderrLine("DNS query log cleared")
	return nil
}
