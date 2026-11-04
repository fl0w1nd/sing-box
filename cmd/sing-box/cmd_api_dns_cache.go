package main

import (
	E "github.com/sagernet/sing/common/exceptions"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
)

var commandAPIDNSCache = &cobra.Command{
	Use:   "dns-cache",
	Short: "Manage the DNS cache",
}

var commandAPIDNSCacheClear = &cobra.Command{
	Use:   "clear",
	Short: "Clear the DNS cache",
	Long: "Clear the DNS cache in memory and in the cache file, the optimistic cache, " +
		"the platform and system DNS caches and the DNS reverse mapping.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAPIDNSCacheClear()
	},
}

func init() {
	commandAPIDNSCache.AddCommand(commandAPIDNSCacheClear)
	commandAPIRoot.AddCommand(commandAPIDNSCache)
}

func runAPIDNSCacheClear() error {
	clientConn, client, err := createExtendedAPIClient()
	if err != nil {
		return err
	}
	defer clientConn.Close()
	_, err = client.ClearDNSCache(globalCtx, &emptypb.Empty{})
	if err != nil {
		return E.Cause(err, "clear DNS cache")
	}
	writeStderrLine("DNS cache cleared")
	return nil
}
