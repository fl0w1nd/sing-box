package subscription_test

import (
	stdJSON "encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/provider/remote/subscription"
	"github.com/sagernet/sing/common/json"
)

// TestConvertedOutboundsMatchSingBoxOptions checks that every outbound
// converted from the curated test cases decodes, with unknown fields rejected,
// as an outbound of this sing-box. After merging sing-box upstream, a renamed,
// removed or retyped field that Sub-Store produces fails here. Random cases are
// skipped, since they contain invalid values on purpose.
func TestConvertedOutboundsMatchSingBoxOptions(t *testing.T) {
	ctx := include.Context(t.Context())
	inputs, err := filepath.Glob("testdata/cases/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		name := filepath.Base(input)
		if strings.HasPrefix(name, "random-") {
			continue
		}
		content, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		converted, err := subscription.Convert(content)
		if err != nil {
			continue
		}
		var document struct {
			Outbounds []stdJSON.RawMessage `json:"outbounds"`
		}
		err = stdJSON.Unmarshal(converted, &document)
		if err != nil {
			t.Fatal(err)
		}
		for _, outbound := range document.Outbounds {
			_, err = json.UnmarshalExtendedContext[option.Outbound](ctx, outbound)
			if err != nil {
				t.Errorf("%s: %s\n%s", name, err, outbound)
			}
		}
	}
}
