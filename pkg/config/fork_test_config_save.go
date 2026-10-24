package config

import (
	"reflect"
	"regexp"

	"gopkg.in/yaml.v3"
)

var legacyKeybindingAliasSuffix = regexp.MustCompile(`-alt[0-9]*$`)

// marshalForkTestConfig keeps the integration harness's effective configuration
// while preserving only intentional legacy keybinding overrides on disk.
func marshalForkTestConfig(current, original *UserConfig) ([]byte, error) {
	var document yaml.Node
	if err := document.Encode(current); err != nil {
		return nil, err
	}
	root := &document
	if document.Kind == yaml.DocumentNode {
		root = document.Content[0]
	}
	keybindings := mappingValue(root, "keybinding")
	if keybindings == nil {
		return yaml.Marshal(current)
	}
	constructor := GetDefaultConfigForPlatform(KeybindingPlatform()).Keybinding
	canonical := constructor
	canonical.MergeLegacyAltKeybindings()
	currentKeybinding := reflect.ValueOf(current.Keybinding)
	originalKeybinding := reflect.ValueOf(original.Keybinding)
	constructorKeybinding := reflect.ValueOf(constructor)
	canonicalKeybinding := reflect.ValueOf(canonical)
	constructorProvenance := current.keybindingExplicitFields == nil
	groups := keybindingFieldTags(reflect.TypeOf(KeybindingConfig{}))
	for groupName, leaves := range groups {
		groupNode := mappingValue(keybindings, groupName)
		if groupNode == nil || groupNode.Kind != yaml.MappingNode {
			continue
		}
		groupField, ok := reflectFieldByYAMLTag(currentKeybinding, groupName)
		originalGroupField, originalOK := reflectFieldByYAMLTag(originalKeybinding, groupName)
		constructorGroup, constructorOK := reflectFieldByYAMLTag(constructorKeybinding, groupName)
		canonicalGroup, canonicalOK := reflectFieldByYAMLTag(canonicalKeybinding, groupName)
		if !ok || !originalOK || !constructorOK || !canonicalOK {
			continue
		}
		for leaf := range leaves {
			path := groupName + "." + leaf
			value := mappingValue(groupNode, leaf)
			field, found := reflectFieldByYAMLTag(groupField, leaf)
			baseline, baselineFound := reflectFieldByYAMLTag(originalGroupField, leaf)
			constructorField, constructorFieldOK := reflectFieldByYAMLTag(constructorGroup, leaf)
			canonicalField, canonicalFieldOK := reflectFieldByYAMLTag(canonicalGroup, leaf)
			if !found {
				continue
			}
			if current.KeybindingIsExplicit(path) {
				if field.Len() == 0 {
					setMappingValue(groupNode, leaf, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<disabled>"})
				}
				continue
			}
			if constructorProvenance {
				if constructorFieldOK && canonicalFieldOK && (reflect.DeepEqual(field.Interface(), constructorField.Interface()) || reflect.DeepEqual(field.Interface(), canonicalField.Interface())) {
					removeMappingValue(groupNode, leaf)
				}
				continue
			}
			if value == nil || !baselineFound {
				continue
			}
			if reflect.DeepEqual(field.Interface(), baseline.Interface()) {
				removeMappingValue(groupNode, leaf)
			}
		}

		// Legacy -alt fields are folded into their primary fields by the loader.
		// If a primary was narrowed, neutralize unchanged implicit aliases so
		// default alias keys cannot reappear during the next load.
		for alias := range leaves {
			parent := legacyKeybindingAliasSuffix.ReplaceAllString(alias, "")
			if parent == alias {
				continue
			}
			primary, primaryOK := reflectFieldByYAMLTag(groupField, parent)
			originalPrimary, originalPrimaryOK := reflectFieldByYAMLTag(originalGroupField, parent)
			aliasValue, aliasOK := reflectFieldByYAMLTag(groupField, alias)
			originalAlias, originalAliasOK := reflectFieldByYAMLTag(originalGroupField, alias)
			if !primaryOK || !aliasOK {
				continue
			}
			constructorAlias, constructorAliasOK := reflectFieldByYAMLTag(constructorGroup, alias)
			primaryChanged := false
			if constructorProvenance {
				constructorPrimary, constructorPrimaryOK := reflectFieldByYAMLTag(constructorGroup, parent)
				canonicalPrimary, canonicalPrimaryOK := reflectFieldByYAMLTag(canonicalGroup, parent)
				primaryChanged = constructorPrimaryOK && canonicalPrimaryOK && !reflect.DeepEqual(primary.Interface(), constructorPrimary.Interface()) && !reflect.DeepEqual(primary.Interface(), canonicalPrimary.Interface())
			} else {
				primaryChanged = originalPrimaryOK && !reflect.DeepEqual(primary.Interface(), originalPrimary.Interface())
			}
			if !primaryOK || !originalPrimaryOK || !aliasOK || !originalAliasOK || !primaryChanged {
				continue
			}
			aliasPath := groupName + "." + alias
			aliasUnchanged := reflect.DeepEqual(aliasValue.Interface(), originalAlias.Interface())
			if constructorProvenance && constructorAliasOK {
				aliasUnchanged = reflect.DeepEqual(aliasValue.Interface(), constructorAlias.Interface())
			}
			if current.KeybindingIsExplicit(aliasPath) || !aliasUnchanged {
				continue
			}
			setMappingValue(groupNode, alias, &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"})
		}
		if len(groupNode.Content) == 0 {
			removeMappingValue(keybindings, groupName)
		}
	}
	if len(keybindings.Content) == 0 {
		removeMappingValue(root, "keybinding")
	}
	return yaml.Marshal(&document)
}

func reflectFieldByYAMLTag(value reflect.Value, tag string) (reflect.Value, bool) {
	for i := 0; i < value.NumField(); i++ {
		if yamlFieldName(value.Type().Field(i)) == tag {
			return value.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func removeMappingValue(node *yaml.Node, key string) {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}

func setMappingValue(node *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1] = value
			return
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	node.Content = append(node.Content, keyNode, value)
}
