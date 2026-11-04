package dnsquerylog

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"
)

// Filter selects entries. Empty fields match everything. Call Prepare
// before Match.
type Filter struct {
	// Search matches the domain, source, process or answers, case-insensitive.
	Search    string
	QueryType string
	Status    string
	// Method is exchange for the queries of clients, lookup for the lookups
	// of the core, or background.
	Method string
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
	f.Method = strings.ToLower(strings.TrimSpace(f.Method))
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
	if f.Method != "" && entry.Method != f.Method {
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

// rawFilter rejects the lines of entries that cannot match a filter before
// they are decoded, by the encoded fields the filter needs. Lines it passes
// are decoded and matched in full.
type rawFilter struct {
	fields [][]byte
	// search is the lower case search when every entry matching it holds
	// it, ASCII case aside, in the encoded line.
	search []byte
}

func (f *Filter) rawFilter() rawFilter {
	var raw rawFilter
	addField := func(key string, value string) {
		if value == "" {
			return
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return
		}
		raw.fields = append(raw.fields, append([]byte(`"`+key+`":`), encoded...))
	}
	addField("status", f.Status)
	addField("method", f.Method)
	addField("inbound", f.Inbound)
	addField("server", f.Server)
	if f.Search != "" && encodesAsIs(f.Search) {
		raw.search = []byte(f.Search)
	}
	return raw
}

// encodesAsIs reports whether s is ASCII that encoding/json writes
// unchanged, so that ASCII case folding of the line finds it.
func encodesAsIs(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c >= utf8.RuneSelf || c == '"' || c == '\\' || c == '<' || c == '>' || c == '&' {
			return false
		}
	}
	return true
}

func (r *rawFilter) match(line []byte) bool {
	for _, field := range r.fields {
		if !bytes.Contains(line, field) {
			return false
		}
	}
	return r.search == nil || containsFoldASCII(line, r.search)
}

// containsFoldASCII reports whether s contains the lower case ASCII substr,
// ignoring the ASCII case of s.
func containsFoldASCII(s []byte, substr []byte) bool {
	if len(substr) == 0 {
		return true
	}
	first := substr[0]
	upper := first
	if 'a' <= first && first <= 'z' {
		upper -= 'a' - 'A'
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if c := s[i]; c != first && c != upper {
			continue
		}
		matched := true
		for j := 1; j < len(substr); j++ {
			c := s[i+j]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != substr[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
