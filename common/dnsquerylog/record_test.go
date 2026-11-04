package dnsquerylog

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"

	"github.com/stretchr/testify/require"
)

func TestLookupStatusIgnoresRuleOfOneHalf(t *testing.T) {
	aaaa := RuleMatch{Rule: "query_type=AAAA", Action: "predefined(NOERROR)", ActionType: C.RuleActionTypePredefined}
	blocked := RuleMatch{Rule: "domain_suffix=ads.test", Action: "reject", ActionType: C.RuleActionTypeReject}
	for _, testCase := range []struct {
		name     string
		entry    Entry
		expected string
	}{
		{name: "lookup answered by the other half", entry: Entry{Method: MethodLookup, Rules: []RuleMatch{aaaa}, Answers: []string{"1.2.3.4"}, Cached: true}, expected: StatusCached},
		{name: "lookup without addresses", entry: Entry{Method: MethodLookup, Rules: []RuleMatch{aaaa}}, expected: StatusPredefined},
		{name: "lookup blocked", entry: Entry{Method: MethodLookup, Rules: []RuleMatch{blocked}}, expected: StatusRejected},
		{name: "exchange", entry: Entry{Method: MethodExchange, Rules: []RuleMatch{aaaa}}, expected: StatusPredefined},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, computeStatus(&testCase.entry, false, nil))
		})
	}
}
