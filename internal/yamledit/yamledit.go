// Package yamledit provides minimal, comment-preserving surgical edits to
// YAML documents (gopkg.in/yaml.v3's Node tree), shared by cmd/linuxctl
// (local-only mcpd user/token administration) and cmd/mcpd (runtime UID
// pinning - see cmd/mcpd/uid_pin.go). A plain struct unmarshal/marshal round
// trip would silently drop every comment in daemon.yaml/mcp-sudo.yaml (both
// files carry maintenance comments that matter - see CLAUDE.md/ARCHITECTURE.md);
// editing the Node tree in place and touching only the keys that actually
// change avoids that.
package yamledit

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// MapGet returns the value node for key in a YAML mapping node, or nil.
func MapGet(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// MapSet sets key to value in a YAML mapping node, replacing any existing
// entry in place (preserving its position) or appending a new one.
func MapSet(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
}

// MapDelete removes key from a YAML mapping node, if present.
func MapDelete(mapping *yaml.Node, key string) bool {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return true
		}
	}
	return false
}

func ScalarNode(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v}
}

func EmptyMapNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode}
}

// LoadDoc parses a YAML file into its full document Node tree.
func LoadDoc(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// SaveDoc re-encodes a document Node tree, using 2-space indentation to
// match this project's existing YAML style (yaml.v3's Marshal defaults to
// 4, which would otherwise reformat every line of the file on every edit -
// pure diff noise unrelated to the actual change being made).
func SaveDoc(path string, doc *yaml.Node) error {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		enc.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0644)
}
