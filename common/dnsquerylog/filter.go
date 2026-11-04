package dnsquerylog

import (
	"slices"
	"strings"
)

// Filter selects entries. Empty fields match everything. Call Prepare
// before Match.
type Filter struct {
	// Search matches the domain, source, process or answers, case-insensitive.
	Search    string
	QueryType string
	Status    string
	// Server matches the tag of any DNS server the query was exchanged with.
	Server  string
	Inbound string
	// Since and Until bound the start time in Unix milliseconds.
	Since int64
	Until int64
}

// Prepare normalizes the filter. Store.Query calls it before matching.
func (f *Filter) Prepare() {
	f.Search = strings.ToLower(strings.TrimSpace(f.Search))
	f.QueryType = strings.ToUpper(strings.TrimSpace(f.QueryType))
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
}

func (f *Filter) Match(entry *Entry) bool {
	if f.Since > 0 && entry.Time < f.Since {
		return false
	}
	if f.Until > 0 && entry.Time > f.Until {
		return false
	}
	if f.Status != "" && entry.Status != f.Status {
		return false
	}
	if f.Inbound != "" && entry.Inbound != f.Inbound {
		return false
	}
	if f.QueryType != "" && !slices.Contains(strings.Split(entry.QueryType, ","), f.QueryType) {
		return false
	}
	if f.Server != "" && !matchServer(entry, f.Server) {
		return false
	}
	if f.Search != "" && !matchSearch(entry, f.Search) {
		return false
	}
	return true
}

func matchServer(entry *Entry, server string) bool {
	for _, upstream := range entry.Upstreams {
		if upstream.Server == server {
			return true
		}
	}
	return false
}

func matchSearch(entry *Entry, search string) bool {
	if containsFold(entry.Domain, search) ||
		containsFold(entry.Source, search) ||
		containsFold(entry.Process, search) ||
		containsFold(entry.Destination, search) {
		return true
	}
	for _, answer := range entry.Answers {
		if containsFold(answer, search) {
			return true
		}
	}
	return false
}

// containsFold reports whether s contains the lower case substr.
func containsFold(s string, substr string) bool {
	return s != "" && strings.Contains(strings.ToLower(s), substr)
}
