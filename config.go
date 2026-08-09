package projectspec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// configAttrKeys are the attribute keys that make a config mapping a leaf (a
// single setting) rather than a section holding sub-entries.
var configAttrKeys = map[string]bool{
	"type": true, "default": true, "control": true, "options": true,
	"min": true, "max": true, "disabled": true, "platforms": true,
}

var knownConfigTypes = map[string]bool{"bool": true, "int": true, "string": true}

var knownConfigControls = map[string]bool{
	"": true, "text": true, "number": true, "switch": true, "select": true,
}

// knownConfigPlatforms are the GOOS values accepted in an entry's "platforms"
// list: the desktop targets the app may run on.
var knownConfigPlatforms = map[string]bool{"linux": true, "windows": true, "darwin": true}

// loadConfig flattens the document's "config" section into document-ordered
// entries. The section is walked as a yaml.Node tree (not unmarshalled into a
// map) so entry order is preserved. Placeholders are not resolved here:
// config defaults are literals.
func (v *validator) loadConfig(doc *yaml.Node) []ConfigEntry {
	root := doc
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return nil
		}
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "config" {
			continue
		}
		section := root.Content[i+1]
		if section.Kind != yaml.MappingNode {
			v.problemf("config: expected a mapping")
			return nil
		}
		var entries []ConfigEntry
		v.walkConfig(section, nil, map[string]bool{}, &entries)
		return entries
	}
	return nil
}

// walkConfig flattens one config mapping: sections (mappings without
// attribute keys) are recursed into, leaves are parsed as entries.
func (v *validator) walkConfig(node *yaml.Node, prefix []string, seen map[string]bool, out *[]ConfigEntry) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		path := append(append([]string{}, prefix...), key)
		joined := strings.Join(path, ".")
		if val.Kind != yaml.MappingNode {
			v.problemf("config.%s: expected a mapping", joined)
			continue
		}
		attrs := 0
		for j := 0; j+1 < len(val.Content); j += 2 {
			if configAttrKeys[val.Content[j].Value] {
				attrs++
			}
		}
		if attrs == 0 {
			v.walkConfig(val, path, seen, out)
			continue
		}
		if attrs*2 != len(val.Content) {
			v.problemf("config.%s: cannot be both an entry and a section", joined)
			continue
		}
		if seen[joined] {
			v.problemf("config.%s: duplicate path", joined)
			continue
		}
		seen[joined] = true
		*out = append(*out, v.parseConfigEntry(joined, path, val))
	}
}

// parseConfigEntry decodes and validates one leaf mapping into a ConfigEntry.
func (v *validator) parseConfigEntry(path string, segs []string, node *yaml.Node) ConfigEntry {
	e := ConfigEntry{Path: segs}
	var typeNode, defNode, controlNode, optionsNode, minNode, maxNode, disabledNode, platformsNode *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "type":
			typeNode = node.Content[i+1]
		case "default":
			defNode = node.Content[i+1]
		case "control":
			controlNode = node.Content[i+1]
		case "options":
			optionsNode = node.Content[i+1]
		case "min":
			minNode = node.Content[i+1]
		case "max":
			maxNode = node.Content[i+1]
		case "disabled":
			disabledNode = node.Content[i+1]
		case "platforms":
			platformsNode = node.Content[i+1]
		}
	}

	validType := false
	if typeNode == nil {
		v.problemf("config.%s.type: required", path)
	} else if typeNode.Kind != yaml.ScalarNode {
		v.problemf("config.%s.type: must be a string", path)
	} else {
		e.Type = typeNode.Value
		if !knownConfigTypes[e.Type] {
			v.problemf("config.%s.type: unknown %q (known: %s)", path, e.Type, keys(knownConfigTypes))
		} else {
			validType = true
		}
	}

	if defNode == nil {
		v.problemf("config.%s.default: required", path)
	} else if validType {
		e.Default = v.decodeTyped(fmt.Sprintf("config.%s.default", path), defNode, e.Type)
	}

	if controlNode != nil {
		if controlNode.Kind != yaml.ScalarNode {
			v.problemf("config.%s.control: must be a string", path)
		} else {
			e.Control = controlNode.Value
			if !knownConfigControls[e.Control] {
				v.problemf("config.%s.control: unknown %q (known: %s)", path, e.Control, keys(knownConfigControls))
			} else if validType {
				switch e.Control {
				case "number":
					if e.Type != "int" {
						v.problemf("config.%s.control: %q only valid for type int", path, e.Control)
					}
				case "switch":
					if e.Type != "bool" {
						v.problemf("config.%s.control: %q only valid for type bool", path, e.Control)
					}
				case "text":
					if e.Type != "string" {
						v.problemf("config.%s.control: %q only valid for type string", path, e.Control)
					}
				}
			}
		}
	}

	if optionsNode != nil {
		if optionsNode.Kind != yaml.SequenceNode {
			v.problemf("config.%s.options: must be a list", path)
		} else if validType {
			for i, el := range optionsNode.Content {
				e.Options = append(e.Options, v.decodeTyped(fmt.Sprintf("config.%s.options[%d]", path, i), el, e.Type))
			}
		}
	}
	if e.Control == "select" && len(e.Options) == 0 {
		v.problemf("config.%s.options: select control requires at least one option", path)
	}

	e.Min = v.decodeMinMax(path, "min", minNode, e.Type, validType)
	e.Max = v.decodeMinMax(path, "max", maxNode, e.Type, validType)
	if e.Min != nil && e.Max != nil && *e.Min > *e.Max {
		v.problemf("config.%s: min %d > max %d", path, *e.Min, *e.Max)
	}
	if d, ok := e.Default.(int); ok {
		if e.Min != nil && d < *e.Min {
			v.problemf("config.%s.default: %d below min %d", path, d, *e.Min)
		}
		if e.Max != nil && d > *e.Max {
			v.problemf("config.%s.default: %d above max %d", path, d, *e.Max)
		}
	}

	if disabledNode != nil {
		if disabledNode.Kind != yaml.ScalarNode || disabledNode.Tag != "!!bool" {
			v.problemf("config.%s.disabled: must be a bool", path)
		} else if err := disabledNode.Decode(&e.Disabled); err != nil {
			v.problemf("config.%s.disabled: must be a bool", path)
		}
	}

	if platformsNode != nil {
		if platformsNode.Kind != yaml.SequenceNode {
			v.problemf("config.%s.platforms: must be a list", path)
		} else {
			for i, el := range platformsNode.Content {
				if el.Kind != yaml.ScalarNode || el.Tag != "!!str" {
					v.problemf("config.%s.platforms[%d]: must be a string", path, i)
					continue
				}
				p := el.Value
				if !knownConfigPlatforms[p] {
					v.problemf("config.%s.platforms[%d]: unknown %q (known: %s)", path, i, p, keys(knownConfigPlatforms))
					continue
				}
				e.Platforms = append(e.Platforms, p)
			}
		}
	}
	return e
}

// decodeTyped decodes a scalar node as the entry type, reporting a problem
// when the yaml kind does not match (e.g. a string default for an int).
func (v *validator) decodeTyped(path string, n *yaml.Node, typ string) any {
	wantTags := map[string]string{"bool": "!!bool", "int": "!!int", "string": "!!str"}
	if n.Kind != yaml.ScalarNode || n.Tag != wantTags[typ] {
		v.problemf("%s: must be %s", path, article(typ))
		return nil
	}
	switch typ {
	case "bool":
		var b bool
		if err := n.Decode(&b); err != nil {
			v.problemf("%s: must be %s", path, article(typ))
			return nil
		}
		return b
	case "int":
		var i int
		if err := n.Decode(&i); err != nil {
			v.problemf("%s: must be %s", path, article(typ))
			return nil
		}
		return i
	default: // string
		return n.Value
	}
}

// decodeMinMax decodes a min/max bound. Bounds are only valid on int
// entries and must be yaml ints.
func (v *validator) decodeMinMax(path, name string, n *yaml.Node, typ string, validType bool) *int {
	if n == nil {
		return nil
	}
	if validType && typ != "int" {
		v.problemf("config.%s.%s: only valid for type int", path, name)
		return nil
	}
	if n.Kind != yaml.ScalarNode || n.Tag != "!!int" {
		v.problemf("config.%s.%s: must be an int", path, name)
		return nil
	}
	var i int
	if err := n.Decode(&i); err != nil {
		v.problemf("config.%s.%s: must be an int", path, name)
		return nil
	}
	return &i
}

func article(typ string) string {
	if typ == "int" {
		return "an int"
	}
	return "a " + typ
}
