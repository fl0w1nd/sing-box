package dnsquerylog

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

type testRuleAction struct{}

func (testRuleAction) Type() string   { return C.RuleActionTypePredefined }
func (testRuleAction) String() string { return "predefined(rcode=NOERROR)" }

type testDNSRule struct {
	adapter.DNSRule
	targeted bool
}

func (r testDNSRule) Action() adapter.RuleAction { return testRuleAction{} }
func (r testDNSRule) String() string             { return "query_type=AAAA" }
func (r testDNSRule) TargetsDestination() bool   { return r.targeted }

func TestRulePolicy(t *testing.T) {
	store := newTestStore(t, option.DNSQueryLogOptions{MemoryOnly: true})
	for _, targeted := range []bool{false, true} {
		ctx, rec := store.beginRecord(context.Background(), MethodExchange, "a.test.", "AAAA")
		Rule(ctx, 3, testDNSRule{targeted: targeted})
		response := new(dns.Msg)
		response.SetQuestion("a.test.", dns.TypeAAAA)
		rec.finish(response, nil, nil)
	}
	entries, _, err := store.Query(context.Background(), Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	// Newest first.
	require.Equal(t, StatusPredefined, entries[1].Status)
	require.Equal(t, []RuleMatch{{Index: 3, Rule: "query_type=AAAA", Action: "predefined(rcode=NOERROR)", ActionType: C.RuleActionTypePredefined, Policy: true}}, entries[1].Rules)
	require.False(t, entries[0].Rules[0].Policy)
}

func TestLookupStatusIgnoresPolicyHalf(t *testing.T) {
	policy := RuleMatch{Rule: "query_type=AAAA", Action: "predefined(NOERROR)", ActionType: C.RuleActionTypePredefined, Policy: true}
	blocked := RuleMatch{Rule: "domain_suffix=ads.test", Action: "reject", ActionType: C.RuleActionTypeReject}
	for _, testCase := range []struct {
		name     string
		entry    Entry
		expected string
	}{
		{name: "lookup answered by the other half", entry: Entry{Method: MethodLookup, Rules: []RuleMatch{policy}, Answers: []string{"1.2.3.4"}, Cached: true}, expected: StatusCached},
		{name: "lookup without addresses", entry: Entry{Method: MethodLookup, Rules: []RuleMatch{policy}}, expected: StatusPredefined},
		{name: "lookup blocked by domain", entry: Entry{Method: MethodLookup, Rules: []RuleMatch{blocked}, Answers: []string{"1.2.3.4"}}, expected: StatusRejected},
		{name: "exchange", entry: Entry{Method: MethodExchange, Rules: []RuleMatch{policy}}, expected: StatusPredefined},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, computeStatus(&testCase.entry, false, nil))
		})
	}
}
