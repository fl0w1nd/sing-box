package dnsquerylog

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/option"

	"github.com/stretchr/testify/require"
)

func readBackward(t *testing.T, content string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "segment")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	var lines []string
	require.NoError(t, readLinesBackward(context.Background(), path, int64(len(content)), func(line []byte) bool {
		lines = append(lines, string(line))
		return true
	}))
	return lines
}

func TestReadLinesBackward(t *testing.T) {
	require.Empty(t, readBackward(t, ""))
	require.Equal(t, []string{"c", "b", "a"}, readBackward(t, "a\nb\n\nc\n"))
	// A last line without a newline, as left by a crash, is read too.
	require.Equal(t, []string{"c", "b", "a"}, readBackward(t, "a\nb\nc"))
	// Lines across blocks, and one longer than a block.
	long := strings.Repeat("x", 3*readBlockSize+17)
	var expected []string
	var content strings.Builder
	for i := range 2000 {
		line := strings.Repeat(string(rune('a'+i%26)), i%97+1)
		if i == 1000 {
			line = long
		}
		content.WriteString(line + "\n")
		expected = append([]string{line}, expected...)
	}
	require.Equal(t, expected, readBackward(t, content.String()))
}

func TestReadLinesBackwardSkipsLongLine(t *testing.T) {
	long := strings.Repeat("x", maxLineSize+readBlockSize)
	require.Equal(t, []string{"c", "a"}, readBackward(t, "a\n"+long+"\nc\n"))
	require.Equal(t, []string{"c"}, readBackward(t, long+"\nc\n"))
}

func TestReadLinesBackwardSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "segment")
	require.NoError(t, os.WriteFile(path, []byte("a\nb\nc\n"), 0o644))
	var lines []string
	// Only what was written when the query started is read, and a size past
	// the end of a truncated file is cut to it.
	for _, size := range []int64{4, 100} {
		lines = lines[:0]
		require.NoError(t, readLinesBackward(context.Background(), path, size, func(line []byte) bool {
			lines = append(lines, string(line))
			return true
		}))
		if size == 4 {
			require.Equal(t, []string{"b", "a"}, lines)
		} else {
			require.Equal(t, []string{"c", "b", "a"}, lines)
		}
	}
}

func TestReadLinesBackwardCanceled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "segment")
	require.NoError(t, os.WriteFile(path, []byte(strings.Repeat("line\n", 10*cancelCheckLines)), 0o644))
	ctx, cancel := context.WithCancel(context.Background())
	var count int
	err := readLinesBackward(ctx, path, 50*cancelCheckLines, func(line []byte) bool {
		count++
		if count == 10 {
			cancel()
		}
		return true
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, count, 2*cancelCheckLines)
}

func TestContainsFoldASCII(t *testing.T) {
	require.True(t, containsFoldASCII([]byte(`"domain":"WWW.Example.com"`), []byte("example.com")))
	require.True(t, containsFoldASCII([]byte("abc"), []byte("")))
	require.False(t, containsFoldASCII([]byte("exampl"), []byte("example")))
	require.False(t, containsFoldASCII([]byte("EXAMPLE"), []byte("examplf")))
}

// The raw filter only skips lines that the full match would reject, so
// queries return the same entries with it.
func TestRawFilterMatchesFull(t *testing.T) {
	entries := []Entry{
		{Domain: "Ads.Example.com", Status: StatusRejected, Method: MethodExchange, Inbound: "tun-in", Source: "10.0.0.2:5353"},
		{Domain: "a<b>&c.test", Status: StatusUpstream, Method: MethodLookup, Process: `C:\Program Files\App "x".exe`, Upstreams: []Upstream{{Server: "remote"}}},
		{Domain: "münchen.test", Status: StatusCached, Method: MethodBackground, Answers: []string{"A 1.2.3.4"}},
		{Domain: "plain.test", Status: StatusPredefined, Method: MethodExchange, Inbound: "dns-in", Upstreams: []Upstream{{Server: "local"}}},
	}
	filters := []Filter{
		{Search: "example"},
		{Search: "EXAMPLE.COM"},
		{Search: "<b>&"},
		{Search: `"x"`},
		{Search: `program files\app`},
		{Search: "MÜNCHEN"},
		{Search: "1.2.3"},
		{Search: "10.0.0.2"},
		{Search: "rejected"},
		{Status: "rejected"},
		{Status: "Cached"},
		{Method: "lookup"},
		{Inbound: "dns-in"},
		{Server: "remote"},
		{Server: "local", Status: "predefined"},
	}
	store := newTestStore(t, option.DNSQueryLogOptions{})
	for _, entry := range entries {
		store.add(entry)
	}
	require.NoError(t, store.flush())
	for _, filter := range filters {
		var expected []uint64
		prepared := filter
		prepared.Prepare()
		for i, entry := range slices.Backward(entries) {
			entry.ID = uint64(i + 1)
			if prepared.Match(&entry) {
				expected = append(expected, entry.ID)
			}
		}
		result, _, err := store.Query(context.Background(), filter, 0, 10)
		require.NoError(t, err)
		var ids []uint64
		for _, entry := range result {
			ids = append(ids, entry.ID)
		}
		require.Equal(t, expected, ids, "%+v", filter)
	}
}
