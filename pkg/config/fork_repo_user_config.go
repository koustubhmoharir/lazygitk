package config

import "slices"

// Fork: a repo switch loads the new repo's user config without making it
// current, so the switch can check it before publishing anything (see
// onNewRepo in pkg/gui/gui.go). CommitRepoUserConfig then makes it current,
// as ReloadUserConfigForRepo does in one step.

// PreparedRepoUserConfig is a repo's user config, loaded but not yet current.
type PreparedRepoUserConfig struct {
	userConfig  *UserConfig
	configFiles []*ConfigFile
}

// UserConfig returns the loaded config.
func (p *PreparedRepoUserConfig) UserConfig() *UserConfig {
	return p.userConfig
}

// PrepareRepoUserConfig loads the user config for a repo, with the same files
// that ReloadUserConfigForRepo would use. It changes nothing in the AppConfig.
func (c *AppConfig) PrepareRepoUserConfig(repoConfigFiles []*ConfigFile) (*PreparedRepoUserConfig, error) {
	configFiles := append(slices.Clone(c.globalUserConfigFiles), repoConfigFiles...)
	userConfig, err := loadUserConfigWithDefaults(configFiles, true)
	if err != nil {
		return nil, err
	}
	return &PreparedRepoUserConfig{userConfig: userConfig, configFiles: configFiles}, nil
}

// CommitRepoUserConfig makes a prepared repo user config the current one.
func (c *AppConfig) CommitRepoUserConfig(prepared *PreparedRepoUserConfig) {
	c.userConfig = prepared.userConfig
	c.userConfigFiles = prepared.configFiles
}
