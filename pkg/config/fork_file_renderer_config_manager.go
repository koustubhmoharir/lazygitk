package config

import "github.com/jesseduffield/lazygit/pkg/i18n"

// Fork: FileRendererConfigManager keeps the file renderer that the full listing of a
// commit's files shows file contents with, and cycles it. It mirrors
// DiffRendererConfigManager; see fork_file_renderers.go.
type FileRendererConfigManager struct {
	getUserConfig     func() *UserConfig
	fileRendererIndex int
}

func NewFileRendererConfigManager(getUserConfig func() *UserConfig) *FileRendererConfigManager {
	return &FileRendererConfigManager{getUserConfig: getUserConfig}
}

// effectiveFileRenderers returns the list the manager cycles: the user's list as it is,
// or the fork default when it is empty. It is never empty.
func (self *FileRendererConfigManager) effectiveFileRenderers() []FileRendererConfig {
	if userFileRenderers := self.getUserConfig().Git.FileRenderers; len(userFileRenderers) > 0 {
		return userFileRenderers
	}
	return forkDefaultFileRenderers()
}

func (self *FileRendererConfigManager) currentFileRendererConfig() *FileRendererConfig {
	fileRenderers := self.effectiveFileRenderers()

	// The index can be out of range if the user has removed file renderers from
	// their config file while lazygitk is running.
	if self.fileRendererIndex >= len(fileRenderers) {
		self.fileRendererIndex = 0
	}

	return &fileRenderers[self.fileRendererIndex]
}

// CurrentFileRendererType returns the kind of the current file renderer.
func (self *FileRendererConfigManager) CurrentFileRendererType() FileRendererType {
	return self.currentFileRendererConfig().getType()
}

func (self *FileRendererConfigManager) CycleFileRenderers() {
	self.fileRendererIndex = (self.fileRendererIndex + 1) % len(self.effectiveFileRenderers())
}

func (self *FileRendererConfigManager) CycleFileRenderersBackward() {
	n := len(self.effectiveFileRenderers())
	self.fileRendererIndex = (self.fileRendererIndex - 1 + n) % n
}

// CurrentFileRendererIndex returns the position of the current file renderer, and the
// number of file renderers.
func (self *FileRendererConfigManager) CurrentFileRendererIndex() (int, int) {
	return self.fileRendererIndex, len(self.effectiveFileRenderers())
}

// CurrentFileRendererName returns a name for the current file renderer, suitable for
// showing to the user.
func (self *FileRendererConfigManager) CurrentFileRendererName(tr *i18n.TranslationSet) string {
	fileRenderer := self.currentFileRendererConfig()
	if fileRenderer.Name != "" {
		return fileRenderer.Name
	}

	switch fileRenderer.getType() {
	case FileRendererType_Builtin:
		return tr.BuiltinFileRendererName
	case FileRendererType_StdinFilter:
		return firstWord(fileRenderer.Command)
	default:
		return tr.PlainFileRendererName
	}
}
