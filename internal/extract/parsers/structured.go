package parsers

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

// JSONParser extracts top-level and nested keys from JSON files.
type JSONParser struct{}

func (p *JSONParser) Extensions() []string { return []string{"json"} }

func (p *JSONParser) Parse(path string, content []byte) (*ParseResult, error) {
	result := &ParseResult{Language: "json"}

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	keys := extractKeys("", data, 3)
	for _, k := range keys {
		result.Exports = append(result.Exports, Export{Name: k, Kind: "key"})
	}

	result.Structure = FormatStructure(result)
	return result, nil
}

// YAMLParser extracts top-level and nested keys from YAML files.
type YAMLParser struct{}

func (p *YAMLParser) Extensions() []string { return []string{"yaml", "yml"} }

func (p *YAMLParser) Parse(path string, content []byte) (*ParseResult, error) {
	result := &ParseResult{Language: "yaml"}

	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	keys := extractKeys("", data, 3)
	for _, k := range keys {
		result.Exports = append(result.Exports, Export{Name: k, Kind: "key"})
	}

	result.Structure = FormatStructure(result)
	return result, nil
}

// TOMLParser extracts sections and keys from TOML files.
// Uses a simple regex approach since TOML structure is line-oriented.
type TOMLParser struct{}

func (p *TOMLParser) Extensions() []string { return []string{"toml"} }

func (p *TOMLParser) Parse(path string, content []byte) (*ParseResult, error) {
	result := &ParseResult{Language: "toml"}

	currentSection := ""
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header [section] or [[array]]
		if strings.HasPrefix(line, "[") {
			section := strings.Trim(line, "[]")
			section = strings.TrimSpace(section)
			currentSection = section
			result.Exports = append(result.Exports, Export{Name: section, Kind: "key"})
			continue
		}

		// Key = value
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			fullKey := key
			if currentSection != "" {
				fullKey = currentSection + "." + key
			}
			result.Exports = append(result.Exports, Export{Name: fullKey, Kind: "key"})
		}
	}

	result.Structure = FormatStructure(result)
	return result, nil
}

// extractKeys recursively extracts key paths from a nested structure.
func extractKeys(prefix string, data interface{}, maxDepth int) []string {
	if maxDepth <= 0 {
		return nil
	}

	var keys []string
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			fullKey := k
			if prefix != "" {
				fullKey = prefix + "." + k
			}
			keys = append(keys, fullKey)
			keys = append(keys, extractKeys(fullKey, val, maxDepth-1)...)
		}
	case []interface{}:
		// Don't enumerate array items, just note it's an array
	}
	return keys
}
