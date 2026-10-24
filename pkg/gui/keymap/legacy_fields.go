//go:generate go run generate_legacy.go

package keymap

// LegacyFields records configuration provenance, never guesses it from key
// values. A few shared IDs use different fields in different views.
func LegacyFields(commandID, view string) []string {
	switch commandID {
	case "tag.create-at-selection":
		if view == "localBranches" {
			return []string{"Branches.CreateTag"}
		}
		return []string{"Commits.CreateTag"}
	case "commit-editor.confirm":
		if view == "commitMessage" {
			return []string{"Universal.SubmitEditorText"}
		}
		return []string{"Universal.ConfirmInEditor"}
	case "config.edit-menu":
		if view == "status" {
			return []string{"Universal.Edit"}
		}
		return []string{"Universal.EditConfig"}
	case "repo.switch-menu":
		if view == "status" {
			return []string{"Status.RecentRepos"}
		}
		return []string{"Universal.OpenRecentRepos"}
	}
	return append([]string(nil), legacyFields[commandID]...)
}

// LegacyAlternates keeps deprecated field provenance separate from the primary
// value into which the loader folds its keys.
func LegacyAlternates(field string) []string {
	return append([]string(nil), legacyAlternates[field]...)
}
