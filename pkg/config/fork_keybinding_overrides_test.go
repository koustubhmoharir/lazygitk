package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestKeybindingExplicitProvenanceFromLoadedFiles(t *testing.T) {
	t.Run("same default value on supported platforms", func(t *testing.T) {
		for _, platform := range []string{"linux", "darwin", "windows"} {
			t.Run(platform, func(t *testing.T) {
				t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", platform)
				defaultKey := GetDefaultConfigForPlatform(platform).Keybinding.Universal.Push[0]
				path := filepath.Join(t.TempDir(), "config.yml")
				content := "keybinding:\n  universal:\n    pushFiles: [" + defaultKey + "]\n"
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				loaded, err := loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
				if err != nil {
					t.Fatal(err)
				}
				if !loaded.KeybindingIsExplicit("universal.pushFiles") || !reflect.DeepEqual(loaded.Keybinding.Universal.Push, GetDefaultConfigForPlatform(platform).Keybinding.Universal.Push) {
					t.Fatal("same-as-default value lost provenance or changed")
				}
				if loaded.KeybindingIsExplicit("universal.pullFiles") {
					t.Fatal("omitted field reported as explicit")
				}
			})
		}
	})
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{name: "empty disabled and null migration", files: []string{"keybinding:\n  files:\n    commitChanges: []\n    commitChangesWithoutHook: ''\n    amendLastCommit: null\n"}, want: []string{"files.commitChanges", "files.commitChangesWithoutHook", "files.amendLastCommit"}},
		{name: "legacy alt stays separate", files: []string{"keybinding:\n  universal:\n    prevItem-alt: [x]\n"}, want: []string{"universal.prevItem-alt"}},
		{name: "unknown string keys do not mask known field", files: []string{"keybinding:\n  universal:\n    pushFiles: [x]\n    futureField:\n      nested: value\n  futureGroup:\n    pushFiles: [ignored]\n"}, want: []string{"universal.pushFiles"}},
		{name: "unknown numeric keys do not mask known field", files: []string{"keybinding:\n  universal:\n    pushFiles: [x]\n    17: ignored\n  23:\n    pushFiles: ignored\n"}, want: []string{"universal.pushFiles"}},
		{name: "inline merge is explicit", files: []string{"keybinding: {universal: {<<: {pushFiles: [x]}}}\n"}, want: []string{"universal.pushFiles"}},
		{name: "inline merge at outer levels", files: []string{"<<: {keybinding: {<<: {universal: {pushFiles: [x]}}}}\n"}, want: []string{"universal.pushFiles"}},
		{name: "inline merge explicit value wins", files: []string{"keybinding: {universal: {<<: {pushFiles: [x]}, pushFiles: []}}\n"}, want: []string{"universal.pushFiles"}},
		{name: "union presence and latest value", files: []string{"keybinding:\n  universal:\n    pushFiles: [x]\n    pullFiles: [z]\n", "keybinding:\n  universal:\n    pushFiles: []\n"}, want: []string{"universal.pushFiles", "universal.pullFiles"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := make([]*ConfigFile, 0, len(tt.files))
			for _, content := range tt.files {
				path := filepath.Join(t.TempDir(), "config.yml")
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				files = append(files, &ConfigFile{Path: path, Policy: ConfigFilePolicyErrorIfMissing})
			}
			loaded, err := loadUserConfigWithDefaults(files, false)
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range tt.want {
				if !loaded.KeybindingIsExplicit(path) {
					t.Errorf("%s not explicit", path)
				}
			}
			if loaded.KeybindingIsExplicit("files.nonexistent") || loaded.KeybindingIsExplicit("keybinding.universal.pushFiles") {
				t.Error("unexpected path recognized")
			}
			if tt.name == "union presence and latest value" && (len(loaded.Keybinding.Universal.Push) != 0 || !reflect.DeepEqual(loaded.Keybinding.Universal.Pull, Keybinding{"z"})) {
				t.Errorf("latest push/pull values = %#v / %#v", loaded.Keybinding.Universal.Push, loaded.Keybinding.Universal.Pull)
			}
			if tt.name == "legacy alt stays separate" {
				if !slices.Contains(loaded.Keybinding.Universal.PrevItem, "x") {
					t.Error("explicit legacy alias missing from effective primary keys")
				}
				if loaded.KeybindingIsExplicit("universal.prevItem") {
					t.Error("legacy alias incorrectly made omitted primary explicit")
				}
			}
			if tt.name == "inline merge explicit value wins" && len(loaded.Keybinding.Universal.Push) != 0 {
				t.Error("inline merge replaced the explicit empty binding")
			}
			if loaded.KeybindingIsExplicit("universal.futureField") || loaded.KeybindingIsExplicit("futureGroup.pushFiles") {
				t.Error("unknown field reported as a recognized override")
			}
			if tt.name == "empty disabled and null migration" {
				if len(loaded.Keybinding.Files.CommitChanges) != 0 || len(loaded.Keybinding.Files.CommitChangesWithoutHook) != 0 || len(loaded.Keybinding.Files.AmendLastCommit) != 0 {
					t.Errorf("loaded values after empty/null migration: commit=%#v withoutHook=%#v amend=%#v", loaded.Keybinding.Files.CommitChanges, loaded.Keybinding.Files.CommitChangesWithoutHook, loaded.Keybinding.Files.AmendLastCommit)
				}
			}
		})
	}
}

func TestKeybindingExplicitProvenanceAliasMergeStillRejectedByMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	content := "defaults: &defaults\n  universal:\n    pushFiles: [x]\nkeybinding:\n  <<: *defaults\n  universal:\n    pushFiles: []\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
	if err == nil || !strings.Contains(err.Error(), "Alias nodes are not supported") {
		t.Fatalf("expected existing alias migration error, got %v", err)
	}
	actual, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(actual) != content {
		t.Fatal("provenance processing rewrote the config")
	}
}

func TestKeybindingExplicitProvenanceFreshReloadAndMissingOptional(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("keybinding:\n  universal:\n    pushFiles: [x]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
	if err != nil || !loaded.KeybindingIsExplicit("universal.pushFiles") {
		t.Fatalf("load: %v", err)
	}
	if err := os.WriteFile(path, []byte("keybinding:\n  universal:\n    pullFiles: [y]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.KeybindingIsExplicit("universal.pushFiles") || !loaded.KeybindingIsExplicit("universal.pullFiles") {
		t.Fatal("fresh load did not replace provenance")
	}
	optional := &ConfigFile{Path: filepath.Join(t.TempDir(), "absent.yml"), Policy: ConfigFilePolicySkipIfMissing}
	loaded, err = loadUserConfigWithDefaults([]*ConfigFile{optional}, false)
	if err != nil || loaded.KeybindingIsExplicit("universal.pushFiles") {
		t.Fatalf("missing optional file: %v", err)
	}
}

func TestKeybindingExplicitMetadataNotSerialized(t *testing.T) {
	plain := GetDefaultConfig()
	with := GetDefaultConfig()
	with.keybindingExplicitFields = map[string]struct{}{"universal.pushFiles": {}}
	plainYAML, err := yaml.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	withYAML, err := yaml.Marshal(with)
	if err != nil {
		t.Fatal(err)
	}
	plainJSON, err := json.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	withJSON, err := json.Marshal(with)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plainYAML, withYAML) || !reflect.DeepEqual(plainJSON, withJSON) {
		t.Fatal("metadata affected serialized config")
	}
	zero := UserConfig{}
	if zero.KeybindingIsExplicit("universal.pushFiles") {
		t.Fatal("zero-value config reports explicit path")
	}
}
