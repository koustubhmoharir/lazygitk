package config

import (
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// KeybindingIsExplicit reports whether a keybinding YAML leaf appeared in a
// loaded config file, independent of its value or the platform default.
// Paths use YAML group/field tags, e.g. "universal.pushFiles". Deprecated aliases
// retain their own provenance even after their values are merged into a primary.
func (config *UserConfig) KeybindingIsExplicit(path string) bool {
	if config == nil {
		return false
	}
	_, ok := config.keybindingExplicitFields[path]
	return ok
}

// recordExplicitKeybindingFields unions the recognized keybinding leaves
// present in one successfully loaded YAML document.
// Inspect migrated content, matching the normal typed loader. Decoding mappings
// into Nodes resolves inline merges without interpreting unknown field values.
func recordExplicitKeybindingFields(config *UserConfig, content []byte) error {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return err
	}
	if len(document.Content) == 0 {
		return nil
	}
	var root map[string]yaml.Node
	if err := document.Content[0].Decode(&root); err != nil {
		return err
	}
	keybinding, ok := root["keybinding"]
	if !ok || keybinding.Kind != yaml.MappingNode {
		return nil
	}
	var keybindingFields map[string]yaml.Node
	if err := keybinding.Decode(&keybindingFields); err != nil {
		return err
	}
	groups := keybindingFieldTags(reflect.TypeOf(KeybindingConfig{}))
	if config.keybindingExplicitFields == nil {
		config.keybindingExplicitFields = make(map[string]struct{})
	}
	for group, fields := range groups {
		values, ok := keybindingFields[group]
		if !ok || values.Kind != yaml.MappingNode {
			continue
		}
		var leaves map[string]yaml.Node
		if err := values.Decode(&leaves); err != nil {
			return err
		}
		for leaf := range fields {
			if _, present := leaves[leaf]; present {
				config.keybindingExplicitFields[group+"."+leaf] = struct{}{}
			}
		}
	}
	return nil
}

func keybindingFieldTags(t reflect.Type) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{}, t.NumField())
	for i := range t.NumField() {
		groupField := t.Field(i)
		group := yamlFieldName(groupField)
		if group == "" || groupField.Type.Kind() != reflect.Struct {
			continue
		}
		leaves := make(map[string]struct{})
		for j := range groupField.Type.NumField() {
			leaf := yamlFieldName(groupField.Type.Field(j))
			if leaf != "" {
				leaves[leaf] = struct{}{}
			}
		}
		result[group] = leaves
	}
	return result
}

func yamlFieldName(field reflect.StructField) string {
	tag := strings.Split(field.Tag.Get("yaml"), ",")[0]
	if tag == "" || tag == "-" {
		return ""
	}
	return tag
}
