package components

import (
	"fmt"
	"time"
)

// EventuallyContent polls the view buffer on the UI thread for content changes
// caused by background events. It never drives a refresh itself.
func (self *ViewDriver) EventuallyContent(matcher *TextMatcher) *ViewDriver {
	deadline := time.Now().Add(10 * time.Second)
	for {
		var content string
		self.t.gui.OnUIThreadAndWait(func() {
			content = self.getView().Buffer()
		})

		ok, message := matcher.context(fmt.Sprintf("%s: Unexpected content.", self.context)).test(content)
		if ok {
			return self
		}

		if !time.Now().Before(deadline) {
			self.t.fail(message)
			return self
		}
		time.Sleep(20 * time.Millisecond)
	}
}
