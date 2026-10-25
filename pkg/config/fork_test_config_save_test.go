package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestForkSaveGlobalUserConfigRoundTripsKeybindingIntent(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "1")
	tests := []struct {
		name       string
		initial    string
		mutate     func(*UserConfig)
		wantKeys   []string
		wantAbsent []string
	}{
		{
			name:       "implicit defaults are not made explicit",
			initial:    "gui:\n  showIcons: true\n",
			wantAbsent: []string{"universal.quit:", "universal.prevItem:"},
		},
		{
			name:    "explicit default and null remain explicit",
			initial: "keybinding:\n  universal:\n    quit: q\n    confirmMenu: null\n",
		},
		{
			name:    "programmatic narrowing is kept",
			initial: "keybinding:\n  universal:\n    prevItem: [<up>, <k>]\n",
			mutate:  func(c *UserConfig) { c.Keybinding.Universal.PrevItem = Keybinding{"<up>"} },
		},
		{
			name:    "unrelated config remains and command bindings survive",
			initial: "gui:\n  showIcons: true\n",
			mutate: func(c *UserConfig) {
				c.CommandBindings = map[string]CommandBindingPaths{"diff.commit.worktree": {{"d", "w"}}}
			},
			wantKeys:   []string{"showIcons: true", "commandBindings:", "diff.commit.worktree:"},
			wantAbsent: []string{"universal.quit:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			if err := os.WriteFile(path, []byte(tt.initial), 0o600); err != nil {
				t.Fatal(err)
			}
			files := []*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}
			current, err := loadUserConfigWithDefaults(files, false)
			if err != nil {
				t.Fatal(err)
			}
			if tt.mutate != nil {
				tt.mutate(current)
			}
			app := &AppConfig{globalUserConfigFiles: files, userConfig: current}
			app.SaveGlobalUserConfig()
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.wantKeys {
				if !strings.Contains(string(saved), want) {
					t.Errorf("saved config missing %q:\n%s", want, saved)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(string(saved), absent) {
					t.Errorf("saved config contains %q:\n%s", absent, saved)
				}
			}
			if err := app.ReloadUserConfigForRepo(nil); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(tt.initial, "confirmMenu: null") {
				if !app.userConfig.KeybindingIsExplicit("universal.confirmMenu") {
					t.Error("confirmMenu should remain explicit")
				}
				if len(app.userConfig.Keybinding.Universal.ConfirmMenu) != 0 {
					t.Errorf("null should retain disabled semantics, got %v", app.userConfig.Keybinding.Universal.ConfirmMenu)
				}
			}
			if strings.Contains(tt.initial, "quit: q") && !app.userConfig.KeybindingIsExplicit("universal.quit") {
				t.Error("explicit default quit binding lost provenance")
			}
			if tt.name == "implicit defaults are not made explicit" {
				if app.userConfig.KeybindingIsExplicit("universal.quit") || app.userConfig.KeybindingIsExplicit("universal.prevItem") {
					t.Errorf("implicit defaults became explicit; saved contains keybinding=%v", strings.Contains(string(saved), "keybinding:"))
				}
			}
			if tt.name == "programmatic narrowing is kept" {
				if !reflect.DeepEqual(Keybinding{"<up>"}, app.userConfig.Keybinding.Universal.PrevItem) {
					t.Errorf("unexpected prevItem: %v", app.userConfig.Keybinding.Universal.PrevItem)
				}
				if !app.userConfig.KeybindingIsExplicit("universal.prevItem") {
					t.Error("changed prevItem should be explicit")
				}
			}
		})
	}
}

func TestForkTestConfigSaveConstructorResetAcrossPlatforms(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "1")
	for _, platform := range []string{"linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", platform)
			path := filepath.Join(t.TempDir(), "config.yml")
			if err := os.WriteFile(path, []byte("keybinding:\n  universal:\n    quit: q\n    prevItem: [<up>, <k>]\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			files := []*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}
			current, err := loadUserConfigWithDefaults(files, false)
			if err != nil {
				t.Fatal(err)
			}
			*current = *GetDefaultConfigForPlatform(platform)
			current.Keybinding.Universal.PrevItem = Keybinding{"r"}
			current.Keybinding.Universal.ConfirmMenu = Keybinding{"e"}
			current.Keybinding.Universal.Return = Keybinding{"x"}
			app := &AppConfig{globalUserConfigFiles: files, userConfig: current}
			app.SaveGlobalUserConfig()
			if err := app.ReloadUserConfigForRepo(nil); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"universal.prevItem", "universal.confirmMenu", "universal.return"} {
				if !app.userConfig.KeybindingIsExplicit(path) {
					t.Errorf("customized %s was not retained", path)
				}
			}
			for _, path := range []string{"universal.quit", "universal.newWorktree", "commits.pasteCommits"} {
				if app.userConfig.KeybindingIsExplicit(path) {
					t.Errorf("constructor default %s became explicit", path)
				}
			}
			if !reflect.DeepEqual(Keybinding{"r"}, app.userConfig.Keybinding.Universal.PrevItem) || !reflect.DeepEqual(Keybinding{"e"}, app.userConfig.Keybinding.Universal.ConfirmMenu) || !reflect.DeepEqual(Keybinding{"x"}, app.userConfig.Keybinding.Universal.Return) {
				t.Errorf("constructor overrides were not preserved: prevItem=%v confirmMenu=%v return=%v", app.userConfig.Keybinding.Universal.PrevItem, app.userConfig.Keybinding.Universal.ConfirmMenu, app.userConfig.Keybinding.Universal.Return)
			}
		})
	}
}

func TestForkTestConfigSaveDoesNotMutateInputs(t *testing.T) {
	current, err := loadUserConfigWithDefaults([]*ConfigFile{}, false)
	if err != nil {
		t.Fatal(err)
	}
	original, err := loadUserConfigWithDefaults([]*ConfigFile{}, false)
	if err != nil {
		t.Fatal(err)
	}
	current.Keybinding.Universal.PrevItem = Keybinding{"<up>"}
	currentBefore := *current
	originalBefore := *original
	currentBefore.Keybinding = cloneKeybindingConfig(current.Keybinding)
	originalBefore.Keybinding = cloneKeybindingConfig(original.Keybinding)
	if _, err := marshalForkTestConfig(current, original); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current, &currentBefore) {
		t.Error("current config was mutated")
	}
	if !reflect.DeepEqual(original, &originalBefore) {
		t.Error("original config was mutated")
	}
}

func cloneKeybindingConfig(in KeybindingConfig) KeybindingConfig {
	out := in
	value := reflect.ValueOf(in)
	outValue := reflect.ValueOf(&out).Elem()
	for i := range value.NumField() {
		group := value.Field(i)
		groupCopy := reflect.New(group.Type()).Elem()
		for j := range group.NumField() {
			field := group.Field(j)
			if field.Kind() == reflect.Slice && !field.IsNil() {
				copyField := reflect.MakeSlice(field.Type(), field.Len(), field.Len())
				reflect.Copy(copyField, field)
				groupCopy.Field(j).Set(copyField)
			} else {
				groupCopy.Field(j).Set(field)
			}
		}
		outValue.Field(i).Set(groupCopy)
	}
	return out
}

func TestForkSaveGlobalUserConfigKeepsExplicitLegacyAlias(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "1")
	path := filepath.Join(t.TempDir(), "config.yml")
	initial := "keybinding:\n  universal:\n    prevItem: [<up>, <k>]\n    prevItem-alt: [x]\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	files := []*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}
	current, err := loadUserConfigWithDefaults(files, false)
	if err != nil {
		t.Fatal(err)
	}
	current.Keybinding.Universal.PrevItem = Keybinding{"<up>"}
	app := &AppConfig{globalUserConfigFiles: files, userConfig: current}
	app.SaveGlobalUserConfig()
	if err := app.ReloadUserConfigForRepo(nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(Keybinding{"<up>", "x"}, app.userConfig.Keybinding.Universal.PrevItem) {
		t.Errorf("alias did not remain additive: %v", app.userConfig.Keybinding.Universal.PrevItem)
	}
	if !app.userConfig.KeybindingIsExplicit("universal.prevItem-alt") {
		t.Error("explicit alias provenance was lost")
	}
}
