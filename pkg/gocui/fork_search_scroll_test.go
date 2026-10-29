package gocui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Fork: a view with KeepOriginXOnSearch selects a match that is past its right edge
// without scrolling sideways, so a list keeps its leading columns.
func TestSelectSearchResultKeepsOriginXWhenKeepOriginXOnSearch(t *testing.T) {
	v := NewView("name", 0, 0, 13, 5, OutputNormal) // InnerWidth 12
	v.KeepOriginXOnSearch = true

	v.writeString("0123456789abcdefghijklmnopqrstuvwxyz\n")

	v.UpdateSearchResults("uv", nil)
	v.SelectSearchResult(0)
	assert.Equal(t, 0, v.OriginX(), "u is past the right edge, but the origin stays")
	assert.Equal(t, 0, v.oy, "the match is on the first line, so the view doesn't scroll down")
	_, cy := v.Cursor()
	assert.Equal(t, 0, cy, "the match's line is still selected")
}

// Fork: a view that is already scrolled sideways keeps its origin when a match is
// selected, even one that is off screen, instead of being moved to the match.
func TestSelectSearchResultLeavesAScrolledOriginXAlone(t *testing.T) {
	v := NewView("name", 0, 0, 13, 5, OutputNormal) // InnerWidth 12
	v.KeepOriginXOnSearch = true

	v.writeString("0123456789abcdefghijklmnopqrstuvwxyz\n")
	v.SetOriginX(20)

	v.UpdateSearchResults("ab", nil)
	v.SelectSearchResult(0)
	assert.Equal(t, 20, v.OriginX(), "a is off screen to the left, and the origin is not reset")
}

// Fork: without KeepOriginXOnSearch, a view keeps the upstream sideways scroll.
func TestSelectSearchResultScrollsSidewaysByDefault(t *testing.T) {
	v := NewView("name", 0, 0, 13, 5, OutputNormal) // InnerWidth 12

	v.writeString("0123456789abcdefghijklmnopqrstuvwxyz\n")

	v.UpdateSearchResults("uv", nil)
	v.SelectSearchResult(0)
	assert.Equal(t, 26, v.OriginX())
}
