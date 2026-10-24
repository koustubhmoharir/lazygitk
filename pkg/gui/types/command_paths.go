package types

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// KeyLabels includes complete menu paths without presenting their continuation
// letters as independent shortcuts. It does not evaluate dynamic metadata.
func (b *Binding) KeyLabels() []string {
	if b == nil {
		return nil
	}
	labels := make([]string, 0, len(b.Keys)+len(b.CommandPaths))
	for _, key := range b.Keys {
		labels = append(labels, config.LabelForKey(key))
	}
	for _, path := range b.CommandPaths {
		labels = append(labels, CommandPathLabel(path))
	}
	return labels
}

// KeyLabels returns the visible shortcuts for this item, including complete
// command paths when the item proxies a binding.
func (self *MenuItem) KeyLabels() []string {
	if self == nil {
		return nil
	}
	if self.Binding != nil {
		return self.Binding.KeyLabels()
	}
	labels := make([]string, 0, len(self.Keys))
	for _, key := range self.Keys {
		labels = append(labels, config.LabelForKey(key))
	}
	return labels
}

func CommandPathLabel(path []gocui.Key) string {
	labels := make([]string, len(path))
	for i, key := range path {
		labels[i] = config.LabelForKey(key)
	}
	return strings.Join(labels, " ")
}

// SetCommandPaths copies paths so callers can build independent prefix frames.
func (b *Binding) SetCommandPaths(paths [][]gocui.Key) {
	b.Keys = nil
	b.CommandPaths = nil
	for _, path := range paths {
		if len(path) == 1 {
			b.Keys = append(b.Keys, path[0])
		} else if len(path) > 1 {
			b.CommandPaths = append(b.CommandPaths, append([]gocui.Key(nil), path...))
		}
	}
}
