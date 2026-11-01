package subscription

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestConvertMatchesSubStore compares the port with Sub-Store: every
// testdata/cases/*.json was produced by Sub-Store from the *.txt next to it,
// see testdata/sync.sh.
func TestConvertMatchesSubStore(t *testing.T) {
	inputs, err := filepath.Glob("testdata/cases/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no test cases")
	}
	removed := make(map[string]bool)
	for _, input := range inputs {
		name := strings.TrimSuffix(filepath.Base(input), ".txt")
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			expectedContent, err := os.ReadFile(strings.TrimSuffix(input, ".txt") + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var expected struct {
				Outbounds json.RawMessage `json:"outbounds"`
			}
			err = json.Unmarshal(expectedContent, &expected)
			if err != nil {
				t.Fatal(err)
			}
			var expectedError struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(expected.Outbounds, &expectedError) == nil && expectedError.Error != "" {
				_, err = Convert(content)
				if err == nil {
					t.Fatalf("Sub-Store failed with %q, the port did not", expectedError.Error)
				}
				return
			}
			converted, err := Convert(content)
			if err != nil {
				t.Fatal(err)
			}
			var actual struct {
				Outbounds json.RawMessage `json:"outbounds"`
			}
			err = json.Unmarshal(converted, &actual)
			if err != nil {
				t.Fatal(err)
			}
			compareOutbounds(t, expected.Outbounds, actual.Outbounds, removed)
		})
	}
	for outboundType, fields := range unsupportedFields {
		for _, field := range fields {
			if !removed[outboundType+"."+field] {
				t.Errorf("Sub-Store no longer produces %s.%s in the test cases: remove it from unsupportedFields or add a case", outboundType, field)
			}
		}
	}
}

func compareOutbounds(t *testing.T, expectedContent, actualContent json.RawMessage, removed map[string]bool) {
	t.Helper()
	var expected, actual []map[string]any
	if err := json.Unmarshal(expectedContent, &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(actualContent, &actual); err != nil {
		t.Fatal(err)
	}
	expectedByTag := make(map[string]map[string]any)
	var expectedTags, actualTags []string
	for _, outbound := range expected {
		outboundType, _ := outbound["type"].(string)
		for _, field := range unsupportedFields[outboundType] {
			if _, loaded := outbound[field]; loaded {
				removed[outboundType+"."+field] = true
				delete(outbound, field)
			}
		}
		tag, _ := outbound["tag"].(string)
		expectedTags = append(expectedTags, tag)
		expectedByTag[tag] = outbound
	}
	for _, outbound := range actual {
		tag, _ := outbound["tag"].(string)
		actualTags = append(actualTags, tag)
		if expectedOutbound, loaded := expectedByTag[tag]; loaded && !reflect.DeepEqual(expectedOutbound, outbound) {
			expectedJSON, _ := json.MarshalIndent(expectedOutbound, "", "  ")
			actualJSON, _ := json.MarshalIndent(outbound, "", "  ")
			t.Errorf("outbound %q differs from Sub-Store\nSub-Store: %s\nport: %s", tag, expectedJSON, actualJSON)
		}
	}
	if !reflect.DeepEqual(expectedTags, actualTags) {
		t.Errorf("outbounds differ from Sub-Store\nSub-Store: %q\nport: %q", expectedTags, actualTags)
	}
}
