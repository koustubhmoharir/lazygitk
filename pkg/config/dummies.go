package config

import (
	"gopkg.in/yaml.v3"
)

// NewDummyAppConfig creates a new dummy AppConfig for testing
func NewDummyAppConfig() *AppConfig {
	userConfig := GetDefaultConfig()
	userConfig.Keybinding.MergeLegacyAltKeybindings()
	appConfig := &AppConfig{
		name:                   "lazygit",
		version:                "unversioned",
		debug:                  false,
		userConfig:             userConfig,
		appState:               &AppState{},
		githubPullRequestCache: newGithubPullRequestCache(""),
	}
	_ = yaml.Unmarshal([]byte{}, appConfig.appState)
	return appConfig
}

// NewDummyAppConfigWithKeybindingFile is NewDummyAppConfig with the config file at
// path loaded through the normal loader, so the keys it sets are explicit, as they
// are for a user's config. Key settings are honoured only when the caller sets
// HonourKeyConfigEnvVar.
func NewDummyAppConfigWithKeybindingFile(path string) (*AppConfig, error) {
	userConfig, err := loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
	if err != nil {
		return nil, err
	}
	appConfig := NewDummyAppConfig()
	appConfig.userConfig = userConfig
	return appConfig, nil
}
