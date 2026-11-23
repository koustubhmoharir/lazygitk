package types

// Fork: LoadedStringTask shows a title and text that Load produces. Load runs on the
// render's own goroutine, not on the UI thread, so it may run git; the view is set up
// from what it returns once it has returned (see fork_loaded_string.go). Key identifies
// what is shown, as the key of a string render does.
type LoadedStringTask struct {
	Key  string
	Load func() (title string, text string)
}

func (t *LoadedStringTask) IsUpdateTask() {}

func NewLoadedStringTask(key string, load func() (title string, text string)) *LoadedStringTask {
	return &LoadedStringTask{Key: key, Load: load}
}
