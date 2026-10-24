package keymap

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// ResolvePaths gives command-ID overrides precedence over explicitly configured
// legacy fields, then falls back to the fork catalogue. Merely matching a
// default value is never evidence that a user configured it.
func ResolvePaths(userConfig *config.UserConfig, id, staticID, view string) (paths [][]gocui.Key, explicit, known bool, err error) {
	if configured, ok := userConfig.CommandBindings[id]; ok {
		for _, labels := range configured {
			var path []gocui.Key
			for _, label := range labels {
				key, ok := config.KeyFromLabel(label)
				if !ok || !key.IsSet() {
					return nil, true, true, fmt.Errorf("invalid key %q for command %s", label, id)
				}
				path = append(path, gocui.NormalizeCommandKey(key))
			}
			if len(path) < 1 || len(path) > 3 {
				return nil, true, true, fmt.Errorf("command %s requires paths of one to three keys", id)
			}
			paths = append(paths, path)
		}
		return paths, true, true, nil
	}
	for _, field := range LegacyFields(staticID, view) {
		keys, yamlPath, ok := legacyField(userConfig, field)
		if !ok {
			return nil, false, false, fmt.Errorf("unknown legacy key field %s for command %s", field, id)
		}
		configured := userConfig.KeybindingIsExplicit(yamlPath)
		for _, alternate := range LegacyAlternates(field) {
			_, alternatePath, ok := legacyField(userConfig, alternate)
			configured = configured || ok && userConfig.KeybindingIsExplicit(alternatePath)
		}
		if !configured {
			continue
		}
		explicit = true
		for _, label := range keys {
			key, ok := config.KeyFromLabel(label)
			if !ok {
				return nil, true, true, fmt.Errorf("invalid key %q in %s", label, yamlPath)
			}
			if key.IsSet() {
				paths = append(paths, []gocui.Key{gocui.NormalizeCommandKey(key)})
			}
		}
	}
	if explicit {
		return paths, true, true, nil
	}
	paths, known = DefaultPaths(id)
	return paths, false, known, nil
}

// legacyField returns the effective value and exact YAML field path. Values of
// deprecated aliases are already folded into their primary by the config loader.
func legacyField(userConfig *config.UserConfig, goPath string) (config.Keybinding, string, bool) {
	value := reflect.ValueOf(userConfig.Keybinding)
	var tags []string
	for _, name := range strings.Split(goPath, ".") {
		if value.Kind() != reflect.Struct {
			return nil, "", false
		}
		field, ok := value.Type().FieldByName(name)
		if !ok {
			return nil, "", false
		}
		tags = append(tags, strings.Split(field.Tag.Get("yaml"), ",")[0])
		value = value.FieldByIndex(field.Index)
	}
	keys, ok := value.Interface().(config.Keybinding)
	return keys, strings.Join(tags, "."), ok
}
