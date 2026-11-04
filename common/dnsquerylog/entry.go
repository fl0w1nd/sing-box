package dnsquerylog

const (
	MethodExchange   = "exchange"
	MethodLookup     = "lookup"
	MethodBackground = "background"
)

const (
	StatusUpstream   = "upstream"
	StatusCached     = "cached"
	StatusOptimistic = "optimistic"
	StatusFakeIP     = "fakeip"
	StatusRejected   = "rejected"
	StatusPredefined = "predefined"
	// StatusStrategy is an empty answer for a query type the domain strategy
	// excludes, such as AAAA with ipv4_only.
	StatusStrategy = "strategy"
	// StatusLocal is any other answer the core made itself without a rule,
	// such as for resolver discovery queries.
	StatusLocal = "local"
	StatusError = "error"
)

// Entry is one DNS query handled by the DNS router.
type Entry struct {
	// ID increases in the order entries are recorded.
	ID uint64 `json:"id"`
	// Time is the start of the query in Unix milliseconds.
	Time int64 `json:"time"`
	// Duration is in microseconds.
	Duration int64  `json:"duration"`
	Method   string `json:"method"`

	Domain    string `json:"domain"`
	QueryType string `json:"query_type,omitempty"`

	Inbound     string `json:"inbound,omitempty"`
	InboundType string `json:"inbound_type,omitempty"`
	Source      string `json:"source,omitempty"`
	User        string `json:"user,omitempty"`
	Process     string `json:"process,omitempty"`
	// Outbound and Destination describe the connection a lookup resolves for.
	Outbound    string `json:"outbound,omitempty"`
	Destination string `json:"destination,omitempty"`

	Rules     []RuleMatch `json:"rules,omitempty"`
	Upstreams []Upstream  `json:"upstreams,omitempty"`

	Cached     bool `json:"cached,omitempty"`
	Optimistic bool `json:"optimistic,omitempty"`
	// Rejected reports that a response was rejected by a rule's address
	// limit and the router went on to the next rule.
	Rejected bool `json:"rejected,omitempty"`

	Status  string   `json:"status"`
	Rcode   string   `json:"rcode,omitempty"`
	Answers []string `json:"answers,omitempty"`
	TTL     uint32   `json:"ttl,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type RuleMatch struct {
	Index      int    `json:"index"`
	Rule       string `json:"rule,omitempty"`
	Action     string `json:"action"`
	ActionType string `json:"action_type"`
}

// Upstream is one exchange with a DNS server.
type Upstream struct {
	Server string `json:"server"`
	Type   string `json:"type"`
	// Address is the configured address of the server, such as
	// https://dns.google/dns-query, or the addresses a local or dhcp server
	// uses at the time. Detour is the configured detour, or the endpoint of a
	// tailscale server.
	Address   string `json:"address,omitempty"`
	Detour    string `json:"detour,omitempty"`
	QueryType string `json:"query_type"`
	// Duration is in microseconds.
	Duration int64    `json:"duration"`
	Rcode    string   `json:"rcode,omitempty"`
	Answers  []string `json:"answers,omitempty"`
	Error    string   `json:"error,omitempty"`
	Rejected bool     `json:"rejected,omitempty"`
}
