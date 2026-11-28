package graph

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/xo/terminfo"
)

// Invariant checker and fuzzer for the commit graph.
//
// The pipe sets (GetPipeSets) are checked against these invariants for every
// row i, with commit c_i, pipe set P_i and previous set P_{i-1} (the initial
// START pipe for row 0):
//
//	I1 lanes of STARTS/CONTINUES pipes in a row are distinct
//	I2 no pipe is dropped or invented between P_{i-1} and P_i (pipes into the
//	   empty tree are exempt: they are dropped on purpose after a root)
//	I3 TERMINATES end at c_i's column; pipes into c_i terminate
//	I4 c_i has one STARTS per parent (one to the empty tree for a root), the
//	   first parent's STARTS is vertical at the commit position
//	I5 the commit position is the leftmost pipe into c_i
//	I6 box drawing: a line goes down from row i iff a line comes up into row i+1
//	I7 box drawing: every pipe's horizontal span is drawn
//	I8 horizontal spans that overlap (see checkOverlaps)
//	I9 the same continuity as I6, for branch drawing symbols
//	X1 render sanity: known glyphs, one commit glyph at the commit position of
//	   the right kind (merge iff more than one STARTS), no box fallback glyph
//	   in branch mode (unselected rows only)
//
// Continuity for commit cells is derived from the pipes, as the commit glyph
// does not show its lines.
//
// Each check also runs with a commit selected ([selected] in its key), for
// every commit of a small list, and for a few commits of a real history. The
// selected commit's pipes are highlighted by renderPipeSet, which can hide a
// vertical line under a bare horizontal glyph. Only such a glyph (a cell
// strictly inside the horizontal span of a highlighted pipe from the selected
// commit) has unknown vertical lines; everything else must still be exact.
//
// Fuzzing: the generators run a fixed set of seeds, with the counts kept
// small for test time (about 1 to 2 s). Raising them is the way to fuzz
// harder: GRAPH_FUZZ_SCALE=<n> multiplies every count by n (default 1).
//
// Set GRAPH_REAL_LOGS to a directory of git log dumps (*.txt) to check real
// histories too, produced with e.g.
//
//	git log --topo-order --all --pretty=format:%H%x00%P -2000 > dir/repo.txt
//
// The first 3 commits of each dump and every 100th are also checked as the
// selected commit.

const (
	lineNo      int8 = 0
	lineYes     int8 = 1
	lineUnknown int8 = -1
)

// selectionLimit is the largest number of commits for which every commit is
// also checked as the selected commit in fuzzed lists.
const selectionLimit = 12

// realLogsSelectedFirst and realLogsSelectedEvery pick the commits of a real
// history that are also checked as the selected commit
const (
	realLogsSelectedFirst = 3
	realLogsSelectedEvery = 100
)

var (
	invHorizontalGlyph = []rune(branchDrawingHorizontal)[0]
	invCommitGlyphs    = func() map[rune]bool {
		m := map[rune]bool{CommitSymbol: false, MergeSymbol: true}
		for _, s := range branchDrawingCommitSymbols {
			m[[]rune(s.commit)[0]] = false
			m[[]rune(s.merge)[0]] = true
		}
		return m
	}()
	invBranchLines = func() map[rune]lineCourses {
		m := map[rune]lineCourses{}
		for lc, s := range branchDrawingConnections {
			m[[]rune(s)[0]] = lc
		}
		return m
	}()
	invBoxUp   = map[rune]bool{'│': true, '┴': true, '╯': true, '╰': true, '╵': true, '┤': true, '├': true}
	invBoxDown = map[rune]bool{'│': true, '┬': true, '╮': true, '╭': true, '╷': true, '┤': true, '├': true}
	invBoxNone = map[rune]bool{' ': true, '─': true, '╶': true}
)

type violation struct {
	id       string
	selected bool
	row      int
	msg      string
}

func (v violation) key() string {
	k := v.id
	if v.selected {
		k += " [selected]"
	}
	return k
}

type graphCase struct {
	opts      []models.NewCommitOpts
	commits   []*models.Commit
	pipes     [][]Pipe
	commitPos []int16 // -1 where unknown
}

func newGraphCase(opts []models.NewCommitOpts) *graphCase {
	pool := &utils.StringPool{}
	commits := lo.Map(opts, func(o models.NewCommitOpts, _ int) *models.Commit {
		return models.NewCommit(pool, o)
	})
	getStyle := func(*models.Commit) *style.TextStyle { return &style.FgDefault }
	return &graphCase{opts: opts, commits: commits, pipes: GetPipeSets(commits, getStyle)}
}

func short(p *string) string {
	switch p {
	case nil:
		return "nil"
	case &StartCommitHash:
		return "START"
	case &EmptyTreeCommitHash:
		return "EMPTY"
	}
	if len(*p) > 7 {
		return (*p)[:7]
	}
	return *p
}

func kindName(k PipeKind) string {
	switch k {
	case TERMINATES:
		return "TERMINATES"
	case STARTS:
		return "STARTS"
	default:
		return "CONTINUES"
	}
}

func pipeString(p Pipe) string {
	return fmt.Sprintf("%s(%s:%d->%s:%d)", kindName(p.kind), short(p.fromHash), p.fromPos, short(p.toHash), p.toPos)
}

// The pointers a commit's first STARTS pipe goes to: its parents, or the empty
// tree for a root
func commitTargets(c *models.Commit) []*string {
	if c.IsFirstCommit() {
		return []*string{&EmptyTreeCommitHash}
	}
	return c.ParentPtrs()
}

// sameLane says whether p continues or terminates the lane of q
func sameLane(q, p Pipe) bool {
	return equalHashes(p.fromHash, q.fromHash) && equalHashes(p.toHash, q.toHash) && p.fromPos == q.toPos
}

func (gc *graphCase) checkPipes() []violation {
	var vs []violation
	add := func(id string, row int, format string, args ...any) {
		vs = append(vs, violation{id: id, row: row, msg: fmt.Sprintf(format, args...)})
	}
	gc.commitPos = make([]int16, len(gc.commits))
	if len(gc.commits) == 0 {
		return nil
	}
	initial := []Pipe{{fromPos: 0, toPos: 0, fromHash: &StartCommitHash, toHash: gc.commits[0].HashPtr(), kind: STARTS}}

	for i, c := range gc.commits {
		prev := initial
		if i > 0 {
			prev = gc.pipes[i-1]
		}
		cur := gc.pipes[i]
		gc.commitPos[i] = -1

		// I4: the STARTS pipes of this commit
		var own []Pipe
		for _, p := range cur {
			if p.kind != STARTS {
				continue
			}
			if equalHashes(p.fromHash, c.HashPtr()) {
				own = append(own, p)
			} else {
				add("I4", i, "STARTS pipe %s is not from this commit", pipeString(p))
			}
		}
		targets := commitTargets(c)
		if len(own) != len(targets) {
			add("I4", i, "want %d STARTS pipes from %s, got %d", len(targets), c.Hash(), len(own))
		}
		for _, t := range targets {
			n := 0
			for _, p := range own {
				if equalHashes(p.toHash, t) {
					n++
				}
			}
			if n != 1 {
				add("I4", i, "target %s has %d STARTS pipes (want 1)", short(t), n)
			}
		}
		for _, p := range own {
			if equalHashes(p.toHash, targets[0]) {
				gc.commitPos[i] = p.fromPos
				if p.toPos != p.fromPos {
					add("I4", i, "first-parent STARTS is not vertical: %s", pipeString(p))
				}
			}
		}
		pos := gc.commitPos[i]
		if pos < 0 {
			add("I4", i, "no first-parent STARTS pipe for %s", c.Hash())
		}
		for _, p := range own {
			if pos >= 0 && p.fromPos != pos {
				add("I4", i, "STARTS pipe %s not at commit position %d", pipeString(p), pos)
			}
		}

		// I1: distinct lanes
		seen := map[int16]bool{}
		for _, p := range cur {
			if p.kind == TERMINATES {
				continue
			}
			if seen[p.toPos] {
				add("I1", i, "two STARTS/CONTINUES pipes end at col %d", p.toPos)
			}
			seen[p.toPos] = true
		}

		// I5: the commit position is the leftmost pipe into this commit
		if pos >= 0 {
			leftmost := int16(-1)
			for _, q := range prev {
				if q.kind != TERMINATES && equalHashes(q.toHash, c.HashPtr()) && (leftmost < 0 || q.toPos < leftmost) {
					leftmost = q.toPos
				}
			}
			if leftmost >= 0 && leftmost != pos {
				add("I5", i, "commit %s at col %d, but leftmost pipe into it ends at col %d", c.Hash(), pos, leftmost)
			}
		}

		// I3: terminations
		for _, p := range cur {
			switch p.kind {
			case TERMINATES:
				if !equalHashes(p.toHash, c.HashPtr()) {
					add("I3", i, "TERMINATES into %s, not into this commit: %s", short(p.toHash), pipeString(p))
				} else if pos >= 0 && p.toPos != pos {
					add("I3", i, "TERMINATES at col %d, commit at col %d: %s", p.toPos, pos, pipeString(p))
				}
			case CONTINUES:
				if equalHashes(p.toHash, c.HashPtr()) {
					add("I3", i, "CONTINUES into this commit instead of terminating: %s", pipeString(p))
				}
			}
		}

		// I2: every pipe of the previous row is carried over exactly once, and
		// nothing new appears
		for _, q := range prev {
			if q.kind == TERMINATES || equalHashes(q.toHash, &EmptyTreeCommitHash) {
				continue
			}
			want := CONTINUES
			if equalHashes(q.toHash, c.HashPtr()) {
				want = TERMINATES
			}
			n := 0
			for _, p := range cur {
				if p.kind == want && sameLane(q, p) {
					n++
				}
			}
			if n != 1 {
				add("I2", i, "previous pipe %s has %d matching %s pipes in this row (want 1)", pipeString(q), n, kindName(want))
			}
		}
		for _, p := range cur {
			if p.kind != TERMINATES && p.kind != CONTINUES {
				continue
			}
			found := false
			for _, q := range prev {
				if q.kind != TERMINATES && sameLane(q, p) {
					found = true
					break
				}
			}
			if !found {
				add("I2", i, "%s has no predecessor in the previous row", pipeString(p))
			}
		}
	}
	return vs
}

// checkOverlaps (I8) reports horizontal spans that share a column gap. Two
// pipes that both touch the commit position are allowed to overlap, as they
// all converge into or emerge from the commit. A CONTINUES pipe is always
// reported.
func checkOverlaps(pipes []Pipe, pos int16, row int) []violation {
	var vs []violation
	touches := func(p Pipe) bool { return pos >= 0 && (p.fromPos == pos || p.toPos == pos) }
	for a := range pipes {
		for b := a + 1; b < len(pipes); b++ {
			pa, pb := pipes[a], pipes[b]
			if pa.left() == pa.right() || pb.left() == pb.right() {
				continue
			}
			if max(pa.left(), pb.left()) >= min(pa.right(), pb.right()) {
				continue
			}
			msg := ""
			switch {
			case pa.kind == CONTINUES || pb.kind == CONTINUES:
				msg = "horizontal spans of a CONTINUES pipe overlap: %s and %s"
			case touches(pa) && touches(pb):
				continue
			default:
				msg = "horizontal spans overlap: %s and %s"
			}
			vs = append(vs, violation{id: "I8", row: row, msg: fmt.Sprintf(msg, pipeString(pa), pipeString(pb))})
		}
	}
	return vs
}

type decodedCell struct {
	up, down int8
	right    bool
	commit   bool
	merge    bool
}

func yesNo(b bool) int8 {
	if b {
		return lineYes
	}
	return lineNo
}

// boxLines gives the vertical lines of a box drawing glyph
func boxLines(first rune) (up, down int8, ok bool) {
	if invBoxUp[first] || invBoxDown[first] {
		return yesNo(invBoxUp[first]), yesNo(invBoxDown[first]), true
	}
	if invBoxNone[first] {
		return lineNo, lineNo, true
	}
	return 0, 0, false
}

// isBareHorizontal says whether a first glyph is the plain horizontal line of
// the symbol set. In branch drawing, a box ─ is a fallback and is reported
// as such, so only the branch horizontal glyph counts there.
func isBareHorizontal(first rune, sym SymbolSet) bool {
	if sym == BranchDrawingSymbols {
		return first == invHorizontalGlyph
	}
	return first == '─'
}

// highlightedPipes gives the pipes of the selected commit that renderPipeSet
// highlights in a row. It mirrors the highlight logic of renderPipeSet.
func highlightedPipes(pipes []Pipe, sel *string, prev *models.Commit) []Pipe {
	if prev != nil && equalHashes(prev.HashPtr(), sel) {
		visible := false
		for _, p := range pipes {
			if equalHashes(p.fromHash, sel) && (p.kind != TERMINATES || p.fromPos != p.toPos) {
				visible = true
			}
		}
		if !visible {
			return nil
		}
	}
	var out []Pipe
	for _, p := range pipes {
		if equalHashes(p.fromHash, sel) {
			out = append(out, p)
		}
	}
	return out
}

// exemptCols gives the columns strictly inside the horizontal span of a
// highlighted pipe from the selected commit. A bare horizontal glyph there may
// hide the lane that renderPipeSet draws it over, so they are unknown.
func exemptCols(highlighted []Pipe) map[int]bool {
	cols := map[int]bool{}
	for _, p := range highlighted {
		for c := p.left() + 1; c < p.right(); c++ {
			cols[int(c)] = true
		}
	}
	return cols
}

// decodeRow splits a rendered row (2 runes per column) into cells. In a
// selected row, a bare horizontal glyph in an exempt column has unknown
// vertical lines (see exemptCols).
func decodeRow(text string, sym SymbolSet, exempt map[int]bool) ([]decodedCell, []string) {
	runes := []rune(text)
	var problems []string
	if len(runes)%2 != 0 {
		problems = append(problems, "odd number of runes in row")
	}
	cells := make([]decodedCell, len(runes)/2)
	for c := range cells {
		first, second := runes[2*c], runes[2*c+1]
		cell := &cells[c]
		if merge, ok := invCommitGlyphs[first]; ok {
			cell.commit, cell.merge = true, merge
		} else if isBareHorizontal(first, sym) {
			cell.up, cell.down = lineNo, lineNo
			if exempt[c] {
				cell.up, cell.down = lineUnknown, lineUnknown
			}
		} else if lc, ok := invBranchLines[first]; ok && sym == BranchDrawingSymbols {
			cell.up, cell.down = yesNo(lc.up != noLine), yesNo(lc.down != noLine)
		} else if up, down, ok := boxLines(first); ok {
			cell.up, cell.down = up, down
			// The branch drawing symbols only fall back to box glyphs for
			// combinations the layout never produces
			if sym == BranchDrawingSymbols && first != ' ' {
				problems = append(problems, fmt.Sprintf("box fallback glyph %q in branch drawing at col %d", first, c))
			}
		} else {
			problems = append(problems, fmt.Sprintf("unknown glyph %q at col %d", first, c))
		}
		switch second {
		case '─', invHorizontalGlyph:
			cell.right = true
		case ' ':
		default:
			problems = append(problems, fmt.Sprintf("unknown right glyph %q at col %d", second, c))
		}
	}
	return cells, problems
}

type renderedRow struct {
	text     string
	cells    []decodedCell
	problems []string
}

// commitEdgesFromPipes gives the lines that touch the commit's own cell
func commitEdgesFromPipes(pipes []Pipe, pos int16) (up, down bool) {
	for _, p := range pipes {
		switch p.kind {
		case TERMINATES:
			if p.fromPos == pos && p.toPos == pos && !equalHashes(p.fromHash, &StartCommitHash) {
				up = true
			}
		case CONTINUES:
			if p.fromPos == pos {
				up = true
			}
			if p.toPos == pos {
				down = true
			}
		case STARTS:
			if p.toPos == pos && !equalHashes(p.toHash, &EmptyTreeCommitHash) {
				down = true
			}
		}
	}
	return up, down
}

func symName(sym SymbolSet) string {
	if sym == BranchDrawingSymbols {
		return "branch"
	}
	return "box"
}

// renderRows renders every row with the given selected commit (nil for none)
func (gc *graphCase) renderRows(sel *string, sym SymbolSet) []renderedRow {
	rows := make([]renderedRow, len(gc.commits))
	for k := range gc.commits {
		var prev *models.Commit
		if k > 0 {
			prev = gc.commits[k-1]
		}
		text := utils.Decolorise(renderPipeSet(gc.pipes[k], sel, prev, sym))
		var exempt map[int]bool
		if sel != nil {
			exempt = exemptCols(highlightedPipes(gc.pipes[k], sel, prev))
		}
		cells, problems := decodeRow(text, sym, exempt)
		pos := gc.commitPos[k]
		if pos >= 0 {
			if int(pos) >= len(cells) {
				problems = append(problems, fmt.Sprintf("commit position %d beyond row", pos))
			} else {
				if !cells[pos].commit {
					problems = append(problems, fmt.Sprintf("no commit glyph at commit position %d", pos))
				}
				up, down := commitEdgesFromPipes(gc.pipes[k], pos)
				cells[pos].up, cells[pos].down = yesNo(up), yesNo(down)
				isMerge := 0
				for _, p := range gc.pipes[k] {
					if p.kind == STARTS {
						isMerge++
					}
				}
				if cells[pos].commit && cells[pos].merge != (isMerge > 1) {
					problems = append(problems, fmt.Sprintf("commit glyph merge=%v but %d STARTS pipes", cells[pos].merge, isMerge))
				}
			}
			for col, cell := range cells {
				if cell.commit && col != int(pos) {
					problems = append(problems, fmt.Sprintf("commit glyph at col %d, commit position is %d", col, pos))
				}
			}
		}
		rows[k] = renderedRow{text: text, cells: cells, problems: problems}
	}
	return rows
}

// checkRows checks the continuity of lines between rows (I6 box, I9 branch),
// the horizontal spans (I7, box only) and the render problems (X1, unselected
// only).
func (gc *graphCase) checkRows(rows []renderedRow, sym SymbolSet, selected bool) ([]violation, int) {
	continuityID := "I6"
	if sym == BranchDrawingSymbols {
		continuityID = "I9"
	}
	var vs []violation
	add := func(id string, row int, format string, args ...any) {
		vs = append(vs, violation{id: id, selected: selected, row: row, msg: fmt.Sprintf(format, args...)})
	}
	cellAt := func(row, col int) decodedCell {
		if col < len(rows[row].cells) {
			return rows[row].cells[col]
		}
		return decodedCell{}
	}
	width := 0
	for _, r := range rows {
		width = max(width, len(r.cells))
	}
	skipped := 0

	for i, r := range rows {
		if !selected {
			for _, p := range r.problems {
				add("X1", i, "%s: %s", symName(sym), p)
			}
		}
		if len(gc.commits) > 0 && gc.commits[i].IsFirstCommit() && gc.commitPos[i] >= 0 && int(gc.commitPos[i]) < len(r.cells) {
			if r.cells[gc.commitPos[i]].down == lineYes {
				add(continuityID, i, "%s drawing: root commit has a line down", symName(sym))
			}
		}
	}
	if len(rows) > 0 {
		for c, cell := range rows[0].cells {
			if cell.up == lineYes {
				add(continuityID, 0, "%s drawing: column %d has a line up in the top row", symName(sym), c)
			}
		}
	}
	for i := 0; i+1 < len(rows); i++ {
		for c := 0; c < width; c++ {
			down := cellAt(i, c).down
			up := cellAt(i+1, c).up
			if down == lineUnknown || up == lineUnknown {
				skipped++
				continue
			}
			if (down == lineYes) != (up == lineYes) {
				add(continuityID, i, "%s drawing: col %d: line down from row %d is %v, line up into row %d is %v",
					symName(sym), c, i, down == lineYes, i+1, up == lineYes)
			}
		}
	}
	if sym == BoxDrawingSymbols {
		for i, pipes := range gc.pipes {
			for _, p := range pipes {
				if p.left() == p.right() {
					continue
				}
				for c := p.left(); c < p.right(); c++ {
					if !cellAt(i, int(c)).right {
						add("I7", i, "pipe %s: no horizontal drawn at col %d", pipeString(p), c)
					}
				}
			}
		}
	}
	return vs, skipped
}

// checkAll runs every check. The selection checks run once for each index in
// selected, with that commit selected.
func (gc *graphCase) checkAll(selected []int) ([]violation, int) {
	if len(gc.commits) == 0 {
		return nil, 0
	}
	vs := gc.checkPipes()
	for i, pipes := range gc.pipes {
		vs = append(vs, checkOverlaps(pipes, gc.commitPos[i], i)...)
	}
	skipped := 0
	for _, sym := range []SymbolSet{BoxDrawingSymbols, BranchDrawingSymbols} {
		v, s := gc.checkRows(gc.renderRows(nil, sym), sym, false)
		vs = append(vs, v...)
		skipped += s
	}
	for _, i := range selected {
		c := gc.commits[i]
		for _, sym := range []SymbolSet{BoxDrawingSymbols, BranchDrawingSymbols} {
			v, s := gc.checkRows(gc.renderRows(c.HashPtr(), sym), sym, true)
			skipped += s
			for _, sv := range v {
				sv.msg = fmt.Sprintf("with %s selected: %s", c.Hash(), sv.msg)
				vs = append(vs, sv)
			}
		}
	}
	return vs, skipped
}

// fuzzSelection gives the commits of a fuzzed list that are checked as the
// selected commit: all of them for a small list, none otherwise
func fuzzSelection(n int) []int {
	if n > selectionLimit {
		return nil
	}
	return lo.Range(n)
}

// realLogsSelection gives the commits of a real history that are checked as
// the selected commit
func realLogsSelection(n int) []int {
	var out []int
	for i := range n {
		if i < realLogsSelectedFirst || i%realLogsSelectedEvery == 0 {
			out = append(out, i)
		}
	}
	return out
}

// runCase checks a commit list, with the given commits selected in turn; a
// panic in the code under test is a violation
func runCase(opts []models.NewCommitOpts, selected []int) (vs []violation, skipped int) {
	defer func() {
		if r := recover(); r != nil {
			vs = append(vs, violation{id: "PANIC", row: -1, msg: fmt.Sprint(r)})
		}
	}()
	return newGraphCase(opts).checkAll(selected)
}

func hasKey(vs []violation, key string) bool {
	return slices.ContainsFunc(vs, func(v violation) bool { return v.key() == key })
}

func needsSelection(key string) bool {
	return strings.Contains(key, "[selected]")
}

// formatRepro prints a commit list as NewCommitOpts literals, its violations and
// its box drawing output
func formatRepro(opts []models.NewCommitOpts) (text string) {
	defer func() {
		if r := recover(); r != nil {
			text += fmt.Sprintf("(formatting panicked: %v)\n", r)
		}
	}()
	var b strings.Builder
	b.WriteString("commits:\n")
	for _, o := range opts {
		if len(o.Parents) == 0 {
			fmt.Fprintf(&b, "\t{Hash: %q},\n", o.Hash)
		} else {
			quoted := lo.Map(o.Parents, func(p string, _ int) string { return fmt.Sprintf("%q", p) })
			fmt.Fprintf(&b, "\t{Hash: %q, Parents: []string{%s}},\n", o.Hash, strings.Join(quoted, ", "))
		}
	}
	vs, _ := runCase(opts, fuzzSelection(len(opts)))
	b.WriteString("violations:\n")
	for _, v := range vs {
		row := "-"
		if v.row >= 0 && v.row < len(opts) {
			row = opts[v.row].Hash
		}
		fmt.Fprintf(&b, "\t%s at row %d (%s): %s\n", v.key(), v.row, row, v.msg)
	}
	gc := newGraphCase(opts)
	gc.checkPipes()
	b.WriteString("box:\n")
	for i, r := range gc.renderRows(nil, BoxDrawingSymbols) {
		fmt.Fprintf(&b, "\t%-4s %s\n", opts[i].Hash, r.text)
	}
	return b.String()
}

// shrinkOpts greedily removes commits (in chunks, then one by one) and parent
// links while pred still holds
func shrinkOpts(opts []models.NewCommitOpts, pred func([]models.NewCommitOpts) bool) []models.NewCommitOpts {
	cur := opts
	changed := true
	for changed {
		changed = false
		for size := max(len(cur)/2, 1); size >= 1; size /= 2 {
			for start := 0; start+size <= len(cur); {
				cand := slices.Concat(cur[:start], cur[start+size:])
				if len(cand) > 0 && pred(cand) {
					cur = cand
					changed = true
				} else {
					start += size
				}
			}
		}
		for i := 0; i < len(cur); i++ {
			for j := 0; j < len(cur[i].Parents); {
				parents := slices.Delete(slices.Clone(cur[i].Parents), j, j+1)
				cand := slices.Clone(cur)
				cand[i].Parents = parents
				if pred(cand) {
					cur = cand
					changed = true
				} else {
					j++
				}
			}
		}
	}
	return cur
}

// Generators

// genTopoDAG generates a DAG as a list with children before parents. Commits
// with no children are tips (1 to 3 at the start, more as unrelated roots).
func genTopoDAG(rnd *rand.Rand, size int) []models.NewCommitOpts {
	parentsOf := map[string][]string{}
	var names []string
	var frontier []string
	newCommit := func() string {
		name := fmt.Sprintf("c%d", len(names))
		names = append(names, name)
		frontier = append(frontier, name)
		return name
	}
	for range 1 + rnd.Intn(3) {
		newCommit()
	}
	var order []string
	for len(frontier) > 0 {
		idx := rnd.Intn(len(frontier))
		cur := frontier[idx]
		frontier = slices.Delete(frontier, idx, idx+1)
		order = append(order, cur)
		if len(names) < size && rnd.Float64() < 0.08 {
			newCommit()
		}
		var count int
		switch r := rnd.Float64(); {
		case r < 0.05:
			count = 0
		case r < 0.30:
			count = 2
		case r < 0.33:
			count = 3
		default:
			count = 1
		}
		var parents []string
		for attempt := 0; attempt < 10 && len(parents) < count; attempt++ {
			var p string
			if len(names) < size && (len(frontier) == 0 || rnd.Float64() < 0.4) {
				p = newCommit()
			} else if len(frontier) > 0 {
				p = frontier[rnd.Intn(len(frontier))]
				if slices.Contains(parents, p) {
					continue
				}
			} else {
				break
			}
			parents = append(parents, p)
		}
		parentsOf[cur] = parents
	}
	opts := make([]models.NewCommitOpts, 0, len(order))
	for _, name := range order {
		opts = append(opts, models.NewCommitOpts{Hash: name, Parents: parentsOf[name]})
	}
	return opts
}

func dagForSeed(seed int64) []models.NewCommitOpts {
	rnd := rand.New(rand.NewSource(seed))
	return genTopoDAG(rnd, 3+rnd.Intn(23))
}

// randomTopoOrder emits the DAG in a random valid order (children first)
func randomTopoOrder(rnd *rand.Rand, dag []models.NewCommitOpts) []models.NewCommitOpts {
	return kahn(dag, func(avail []string) int { return rnd.Intn(len(avail)) })
}

// dateOrder emits the DAG like `git log --date-order`: among the commits whose
// children are all emitted, the one with the latest timestamp goes first. The
// timestamps are random but a child is always newer than its parents.
func dateOrder(rnd *rand.Rand, dag []models.NewCommitOpts) []models.NewCommitOpts {
	ts := map[string]int{}
	for _, c := range dag {
		ts[c.Hash] = rnd.Intn(1000)
	}
	for _, c := range dag {
		for _, p := range c.Parents {
			if _, ok := ts[p]; ok && ts[p] >= ts[c.Hash] {
				ts[p] = ts[c.Hash] - 1
			}
		}
	}
	return kahn(dag, func(avail []string) int {
		best := 0
		for i, name := range avail {
			if ts[name] > ts[avail[best]] {
				best = i
			}
		}
		return best
	})
}

// kahn emits the DAG's commits once all their children are emitted, choosing
// among the ready commits with pick
func kahn(dag []models.NewCommitOpts, pick func(avail []string) int) []models.NewCommitOpts {
	byName := map[string]models.NewCommitOpts{}
	childCount := map[string]int{}
	for _, c := range dag {
		byName[c.Hash] = c
	}
	for _, c := range dag {
		for _, p := range c.Parents {
			if _, ok := byName[p]; ok {
				childCount[p]++
			}
		}
	}
	var avail []string
	for _, c := range dag {
		if childCount[c.Hash] == 0 {
			avail = append(avail, c.Hash)
		}
	}
	out := make([]models.NewCommitOpts, 0, len(dag))
	for len(avail) > 0 {
		idx := pick(avail)
		name := avail[idx]
		avail = slices.Delete(avail, idx, idx+1)
		c := byName[name]
		out = append(out, c)
		for _, p := range c.Parents {
			if _, ok := byName[p]; ok {
				childCount[p]--
				if childCount[p] == 0 {
					avail = append(avail, p)
				}
			}
		}
	}
	return out
}

// filterRandom drops each commit with probability 0.4, keeping parent links
func filterRandom(rnd *rand.Rand, dag []models.NewCommitOpts) []models.NewCommitOpts {
	var out []models.NewCommitOpts
	for _, c := range dag {
		if rnd.Float64() < 0.6 {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		out = append(out, dag[rnd.Intn(len(dag))])
	}
	return out
}

// Fuzz runner

type group struct {
	key       string
	cases     int
	instances int
	shrunk    int
	example   string
	repros    []string
	seen      map[string]bool
}

type fuzzRunner struct {
	groups  map[string]*group
	cases   int
	skipped int
}

// shrinkPerKey caps the number of violating cases that are shrunk per key
const shrinkPerKey = 40

func (r *fuzzRunner) run(src string, opts []models.NewCommitOpts) {
	r.cases++
	vs, skipped := runCase(opts, fuzzSelection(len(opts)))
	r.skipped += skipped
	counts := map[string]int{}
	for _, v := range vs {
		counts[v.key()]++
	}
	for key, n := range counts {
		g := r.groups[key]
		if g == nil {
			g = &group{key: key, seen: map[string]bool{}}
			r.groups[key] = g
		}
		g.cases++
		g.instances += n
		if g.shrunk >= shrinkPerKey {
			continue
		}
		g.shrunk++
		minimal := shrinkOpts(opts, func(o []models.NewCommitOpts) bool {
			var selected []int
			if needsSelection(key) {
				selected = fuzzSelection(len(o))
			}
			vs, _ := runCase(o, selected)
			return hasKey(vs, key)
		})
		text := formatRepro(minimal)
		if !g.seen[text] {
			g.seen[text] = true
			g.repros = append(g.repros, text)
			if g.example == "" {
				g.example = src
			}
		}
	}
}

func (r *fuzzRunner) report(t *testing.T) {
	keys := make([]string, 0, len(r.groups))
	for k := range r.groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("cases run: %d; I6/I9 pairs skipped (unknown lines): %d", r.cases, r.skipped)
	for _, k := range keys {
		g := r.groups[k]
		t.Logf("%-32s cases=%d instances=%d distinct minimal repros=%d (first from %s)", k, g.cases, g.instances, len(g.repros), g.example)
	}
	for _, k := range keys {
		g := r.groups[k]
		for _, text := range g.repros {
			t.Errorf("VIOLATION %s\n%s", k, text)
		}
	}
}

func newFuzzRunner() *fuzzRunner {
	return &fuzzRunner{groups: map[string]*group{}}
}

// fuzzScale multiplies the generator counts: GRAPH_FUZZ_SCALE=10 fuzzes ten
// times as many lists. The default is 1.
func fuzzScale(t *testing.T) int {
	s := os.Getenv("GRAPH_FUZZ_SCALE")
	if s == "" {
		return 1
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		t.Fatalf("GRAPH_FUZZ_SCALE must be a positive integer, got %q", s)
	}
	return n
}

// The generator counts are kept small for test time, so that the test runs in
// about a second or two. They are the seed ranges of each generator, and
// fuzzScale multiplies them.
func TestInvariantFuzz(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	defer color.ForceSetColorLevel(oldColorLevel)

	scale := fuzzScale(t)
	r := newFuzzRunner()
	// G1: random DAGs
	for seed := range 500 * scale {
		r.run(fmt.Sprintf("G1 seed=%d", seed), dagForSeed(int64(seed)))
	}
	// G2: the same DAGs in several topo orders, and one date order
	for seed := range 100 * scale {
		dag := dagForSeed(int64(seed))
		rnd := rand.New(rand.NewSource(int64(seed) + 1_000_000))
		for k := range 4 {
			r.run(fmt.Sprintf("G2 seed=%d topo order %d", seed, k), randomTopoOrder(rnd, dag))
		}
		r.run(fmt.Sprintf("G2 seed=%d date order", seed), dateOrder(rnd, dag))
	}
	// G3: truncated lists
	for seed := range 250 * scale {
		dag := dagForSeed(int64(seed))
		rnd := rand.New(rand.NewSource(int64(seed) + 2_000_000))
		r.run(fmt.Sprintf("G3 seed=%d truncated", seed), dag[:1+rnd.Intn(len(dag))])
	}
	// G4: filtered lists
	for seed := range 250 * scale {
		dag := dagForSeed(int64(seed))
		rnd := rand.New(rand.NewSource(int64(seed) + 3_000_000))
		r.run(fmt.Sprintf("G4 seed=%d filtered", seed), filterRandom(rnd, dag))
	}
	r.report(t)
}

// Real logs

func parseGitLog(path string) ([]models.NewCommitOpts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var opts []models.NewCommitOpts
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == "" {
			continue
		}
		hash, parents, _ := strings.Cut(line, "\x00")
		opts = append(opts, models.NewCommitOpts{Hash: hash, Parents: strings.Fields(parents)})
	}
	return opts, nil
}

func dumpRows(gc *graphCase, from, to int) string {
	var b strings.Builder
	rows := gc.renderRows(nil, BoxDrawingSymbols)
	for i := max(from, 0); i <= min(to, len(gc.commits)-1); i++ {
		var pipes []string
		for _, p := range gc.pipes[i] {
			pipes = append(pipes, pipeString(p))
		}
		fmt.Fprintf(&b, "\trow %d %s parents=%v\n\t    %s\n\t    pipes: %s\n", i, short(gc.commits[i].HashPtr()),
			lo.Map(gc.commits[i].ParentPtrs(), func(p *string, _ int) string { return short(p) }),
			rows[i].text, strings.Join(pipes, " "))
	}
	return b.String()
}

func TestInvariantRealLogs(t *testing.T) {
	dir := os.Getenv("GRAPH_REAL_LOGS")
	if dir == "" {
		t.Skip("set GRAPH_REAL_LOGS to a directory of git log dumps to check them")
	}
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelMillions)
	defer color.ForceSetColorLevel(oldColorLevel)

	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		opts, err := parseGitLog(file)
		if err != nil {
			t.Fatal(err)
		}
		gc := newGraphCase(opts)
		vs, skipped := runCase(opts, realLogsSelection(len(opts)))
		t.Logf("%s: %d commits, I6/I9 pairs skipped: %d", filepath.Base(file), len(opts), skipped)
		counts := map[string]int{}
		for _, v := range vs {
			counts[v.key()]++
		}
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			var first violation
			for _, v := range vs {
				if v.key() == k {
					first = v
					break
				}
			}
			msg := fmt.Sprintf("%s: %s instances=%d first at row %d: %s", filepath.Base(file), k, counts[k], first.row, first.msg)
			t.Errorf("VIOLATION %s", msg)
			if first.row >= 0 {
				t.Logf("context:\n%s", dumpRows(gc, first.row-2, first.row+2))
			}
		}
	}
}
