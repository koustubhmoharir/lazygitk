package helpers

type TagsHelper struct {
	c             *HelperCommon
	commitsHelper *CommitsHelper
	gpg           *GpgHelper
}

func NewTagsHelper(c *HelperCommon, commitsHelper *CommitsHelper, gpg *GpgHelper) *TagsHelper {
	return &TagsHelper{
		c:             c,
		commitsHelper: commitsHelper,
		gpg:           gpg,
	}
}

// Fork: replace live repository/ref lookups in queued tag creation with a
// captured source, exact overwrite review and guarded worker/terminal launch.
func (self *TagsHelper) OpenCreateTagPrompt(ref string, onCreate func()) error {
	return self.openCapturedCreateTagPrompt(ref, onCreate)
}
