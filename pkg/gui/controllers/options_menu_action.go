package controllers

type OptionsMenuAction struct {
	c *ControllerCommon
}

// Fork: the palette replaces upstream's menu body; the palette is assembled in
// openPalette (fork_palette_menu.go).
func (self *OptionsMenuAction) Call() error {
	return self.openPalette()
}
