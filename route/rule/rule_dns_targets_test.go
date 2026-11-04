package rule

import (
	"context"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestDNSRuleTargetsDestination(t *testing.T) {
	t.Parallel()
	predefined := option.DNSRuleAction{Action: C.RuleActionTypePredefined}
	for _, testCase := range []struct {
		name     string
		options  option.RawDefaultDNSRule
		expected bool
	}{
		{name: "query type", options: option.RawDefaultDNSRule{
			QueryType: badoption.Listable[option.DNSQueryType]{option.DNSQueryType(dns.TypeAAAA)},
		}, expected: false},
		{name: "domain suffix", options: option.RawDefaultDNSRule{
			DomainSuffix: badoption.Listable[string]{"ads.test"},
		}, expected: true},
		{name: "query type and domain", options: option.RawDefaultDNSRule{
			QueryType:    badoption.Listable[option.DNSQueryType]{option.DNSQueryType(dns.TypeHTTPS)},
			DomainSuffix: badoption.Listable[string]{"example.test"},
		}, expected: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rule, err := NewDNSRule(context.Background(), log.NewNOPFactory().NewLogger("dns"), option.DNSRule{
				Type:           C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{RawDefaultDNSRule: testCase.options, DNSRuleAction: predefined},
			}, true, false)
			require.NoError(t, err)
			targeted, isTargeted := rule.(interface{ TargetsDestination() bool })
			require.True(t, isTargeted)
			require.Equal(t, testCase.expected, targeted.TargetsDestination())
		})
	}
}
