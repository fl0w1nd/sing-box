// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// utils/yaml.js

var yamlStrTagRegex = regexp.MustCompile(`!<str>\s*`)

// yamlLoad is safeLoad: it parses a YAML document into the JavaScript model.
func yamlLoad(content string) (any, error) {
	value, err := yamlParse(content)
	if err != nil {
		// Sub-Store retries with "!<str>" values turned into plain strings.
		value, err = yamlParse(yamlStrTagRegex.ReplaceAllString(content, "!!str "))
	}
	return value, err
}

func yamlParse(content string) (any, error) {
	var document yaml.Node
	err := yaml.Unmarshal([]byte(content), &document)
	if err != nil {
		return nil, err
	}
	if document.Kind == 0 {
		return nil, nil
	}
	return yamlValue(&document, 0)
}

const maximumYAMLDepth = 64

func yamlValue(node *yaml.Node, depth int) (any, error) {
	if depth > maximumYAMLDepth {
		return nil, jsError("YAML document is too deep")
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, nil
		}
		return yamlValue(node.Content[0], depth+1)
	case yaml.AliasNode:
		return yamlValue(node.Alias, depth+1)
	case yaml.SequenceNode:
		result := make([]any, 0, len(node.Content))
		for _, item := range node.Content {
			value, err := yamlValue(item, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case yaml.MappingNode:
		o := newObject()
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode, valueNode := node.Content[i], node.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode && keyNode.Value == "<<" && keyNode.Tag != "!!str" {
				err := yamlMerge(o, valueNode, depth+1)
				if err != nil {
					return nil, err
				}
				continue
			}
			key, err := yamlValue(keyNode, depth+1)
			if err != nil {
				return nil, err
			}
			value, err := yamlValue(valueNode, depth+1)
			if err != nil {
				return nil, err
			}
			o.Set(jsString(key), value)
		}
		return o, nil
	case yaml.ScalarNode:
		return yamlScalar(node), nil
	}
	return nil, nil
}

// yamlMerge applies a merge key: existing keys are not overridden.
func yamlMerge(target *Object, node *yaml.Node, depth int) error {
	var sources []*yaml.Node
	if node.Kind == yaml.SequenceNode {
		sources = node.Content
	} else {
		sources = []*yaml.Node{node}
	}
	for _, source := range sources {
		value, err := yamlValue(source, depth)
		if err != nil {
			return err
		}
		merged, isObject := value.(*Object)
		if !isObject {
			continue
		}
		for _, key := range merged.Keys() {
			if !target.Has(key) {
				target.Set(key, merged.Get(key))
			}
		}
	}
	return nil
}

func yamlScalar(node *yaml.Node) any {
	switch node.ShortTag() {
	case "!!null":
		return jsNull
	case "!!bool":
		var value bool
		if node.Decode(&value) == nil {
			return value
		}
	case "!!int":
		var value int64
		if node.Decode(&value) == nil {
			return float64(value)
		}
		var unsigned uint64
		if node.Decode(&unsigned) == nil {
			return float64(unsigned)
		}
		number, err := strconv.ParseFloat(strings.ReplaceAll(node.Value, "_", ""), 64)
		if err == nil {
			return number
		}
	case "!!float":
		var value float64
		if node.Decode(&value) == nil {
			return value
		}
		if strings.EqualFold(strings.TrimPrefix(node.Value, "."), "nan") {
			return math.NaN()
		}
	}
	return node.Value
}
