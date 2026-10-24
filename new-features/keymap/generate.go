//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type keyRow struct {
	Name  int    `json:"name"`
	Str   string `json:"str"`
	Mod   int    `json:"mod"`
	Label string `json:"label"`
}

type bindingRow struct {
	Declaration        int      `json:"declaration"`
	IdentityOrdinal    int      `json:"identityOrdinal"`
	DefaultDeclaration *int     `json:"defaultDeclaration,omitempty"`
	ID                 string   `json:"id"`
	Verb               string   `json:"verb"`
	Description        string   `json:"description"`
	View               string   `json:"view"`
	Tag                string   `json:"tag"`
	Keys               []string `json:"defaultKeys"`
	RawKeys            []keyRow `json:"rawKeys"`
	DynamicID          bool     `json:"hasDynamicID"`
	DynamicVerb        bool     `json:"hasDynamicVerb"`
	DynamicDesc        bool     `json:"hasDynamicDescription"`
	Guarded            bool     `json:"guarded"`
	Contexts           []string `json:"contexts,omitempty"`
}

type collision struct {
	Platform  string   `json:"platform"`
	ScopeKind string   `json:"scopeKind"`
	Scope     string   `json:"scope"`
	Key       keyRow   `json:"key"`
	Class     string   `json:"class"`
	Origins   []string `json:"origins"`
	IDs       []string `json:"ids"`
	Verbs     []string `json:"verbs"`
	Order     []int    `json:"declarations"`
	Evidence  string   `json:"scopeEvidence,omitempty"`
}

type snapshot struct {
	Platform    string       `json:"platform"`
	Bindings    []bindingRow `json:"bindings"`
	Cleared     []bindingRow `json:"clearedKeyBindings"`
	Collides    []collision  `json:"collisions"`
	ParentEdges []parentEdge `json:"parentViewEdges"`
}

type parentEdge struct {
	Child           string `json:"childView"`
	Parent          string `json:"parentView"`
	Evidence        string `json:"evidence"`
	ActiveByDefault bool   `json:"activeByDefault"`
}

type report struct {
	Description string     `json:"description"`
	Platforms   []snapshot `json:"platforms"`
}

func main() {
	out := report{Description: "Instantiated default snapshots; not all modes, dynamic menus, or live availability.", Platforms: make([]snapshot, 0, 3)}
	for _, platform := range []string{"linux", "darwin", "windows"} {
		guiInstance, cleanup := instantiate(platform, false)
		bindings := guiInstance.GetCheatsheetKeybindings()
		rows := rowsFor(bindings, guiInstance.State.Contexts.Flatten())
		parentEdges := viewParentEdges(guiInstance)
		cleanup()
		clearedGui, cleanup := instantiate(platform, true)
		clearedRows := rowsFor(clearedGui.GetCheatsheetKeybindings(), clearedGui.State.Contexts.Flatten())
		cleanup()
		matchClearedRows(rows, clearedRows)
		out.Platforms = append(out.Platforms, snapshot{Platform: platform, Bindings: rows, Cleared: clearedRows, Collides: analyze(platform, rows, parentEdges), ParentEdges: parentEdges})
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		panic(err)
	}
}

func viewParentEdges(instance *gui.Gui) []parentEdge {
	views := []*gocui.View{instance.Views.MenuFilterFrame, instance.Views.MenuFilter}
	rows := make([]parentEdge, 0, len(views))
	for _, child := range views {
		if child != nil && child.ParentView != nil {
			rows = append(rows, parentEdge{Child: child.Name(), Parent: child.ParentView.Name(), Evidence: "initialized Gui.Views ParentView pointer; assignments in pkg/gui/views.go", ActiveByDefault: false})
		}
	}
	return rows
}

func instantiate(platform string, clear bool) (*gui.Gui, func()) {
	tempDir, err := os.MkdirTemp("", "lazygit-keymap-audit-")
	if err != nil {
		panic(err)
	}
	cfg := config.NewDummyAppConfig()
	userConfig := config.GetDefaultConfigForPlatform(platform)
	*cfg.GetUserConfig() = *userConfig
	cfg.GetUserConfig().Keybinding.MergeLegacyAltKeybindings()
	if clear {
		clearLeaves(reflect.ValueOf(&cfg.GetUserConfig().Keybinding))
	}
	cmn := common.NewDummyCommonWithUserConfigAndAppState(cfg.GetUserConfig(), cfg.GetAppState())
	instance, err := gui.NewGui(cmn, cfg, &git_commands.GitVersion{Major: 2, Minor: 51}, nil, false, tempDir, nil)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		panic(err)
	}
	return instance, func() {
		if err := os.RemoveAll(tempDir); err != nil {
			panic(err)
		}
	}
}

func clearLeaves(value reflect.Value) {
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if !value.IsNil() {
			clearLeaves(value.Elem())
		}
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			clearLeaves(value.Field(i))
		}
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Struct {
			for i := 0; i < value.Len(); i++ {
				clearLeaves(value.Index(i))
			}
		} else if value.CanSet() {
			value.Set(reflect.Zero(value.Type()))
		}
	}
}

func rowsFor(bindings []*types.Binding, contexts []types.Context) []bindingRow {
	rows := make([]bindingRow, 0, len(bindings))
	ordinals := map[string]int{}
	for i, b := range bindings {
		identity := b.ViewName + "\x00" + b.CommandID + "\x00" + b.Verb
		row := bindingRow{Declaration: i, IdentityOrdinal: ordinals[identity], ID: b.CommandID, Verb: b.Verb, Description: b.Description, View: b.ViewName, Tag: b.Tag, DynamicID: b.CommandIDFunc != nil, DynamicVerb: b.VerbFunc != nil, DynamicDesc: b.DescriptionFunc != nil, Guarded: b.GetDisabledReason != nil}
		ordinals[identity]++
		for _, key := range b.Keys {
			row.Keys = append(row.Keys, config.LabelForKey(key))
			row.RawKeys = append(row.RawKeys, keyRow{Name: int(key.KeyName()), Str: key.Str(), Mod: int(key.Mod()), Label: config.LabelForKey(key)})
		}
		for _, context := range contexts {
			if context.GetViewName() == b.ViewName || b.ViewName == "" && context.GetKind() == types.GLOBAL_CONTEXT {
				row.Contexts = append(row.Contexts, contextChain(context)...)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func matchClearedRows(defaultRows, clearedRows []bindingRow) {
	indices := make(map[string]int, len(defaultRows))
	for i, row := range defaultRows {
		indices[identityKey(row)] = i
	}
	for i := range clearedRows {
		if index, ok := indices[identityKey(clearedRows[i])]; ok {
			declaration := defaultRows[index].Declaration
			clearedRows[i].DefaultDeclaration = &declaration
		}
	}
}

func identityKey(row bindingRow) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%d", row.View, row.ID, row.Verb, row.IdentityOrdinal)
}

func contextChain(context types.Context) []string {
	var chain []string
	for c := context; c != nil; c = c.GetParentContext() {
		chain = append(chain, fmt.Sprintf("%s|%d|%s", c.GetKey(), c.GetKind(), c.GetInputViewName()))
	}
	return chain
}

func analyze(platform string, rows []bindingRow, edges []parentEdge) []collision {
	return collisionRows(platform, rows, edges)
}

func collisionRows(platform string, rows []bindingRow, parentEdges []parentEdge) []collision {
	type group struct {
		kind, scope, normalized, evidence string
		rows                              []bindingRow
		key                               keyRow
	}
	bases := map[string]*group{}
	baseOrder := []string{}
	add := func(kind, scope, evidence string, row bindingRow, key keyRow) {
		n := normalizeRawKey(key)
		id := kind + "\x00" + scope + "\x00" + n
		g := bases[id]
		if g == nil {
			g = &group{kind: kind, scope: scope, normalized: n, evidence: evidence, key: canonicalKey(key)}
			bases[id] = g
			baseOrder = append(baseOrder, id)
		}
		g.rows = append(g.rows, row)
	}
	for _, row := range rows {
		for _, key := range row.RawKeys {
			if row.View == "" {
				add("global", "", "", row, key)
			} else {
				add("view", row.View, "", row, key)
			}
		}
	}
	all := append([]string(nil), baseOrder...)
	// Create overlap groups independently from immutable base groups. These are
	// candidates, since runtime visibility and focus determine actual dispatch.
	for _, id := range baseOrder {
		local := bases[id]
		if local.kind != "view" {
			continue
		}
		global := bases["global\x00\x00"+local.normalized]
		if global != nil {
			g := &group{kind: "local-global", scope: local.scope, normalized: local.normalized, evidence: "global bindings may be considered after the focused view", key: local.key, rows: append(append([]bindingRow{}, local.rows...), global.rows...)}
			key := "local-global\x00" + g.scope + "\x00" + g.normalized
			bases[key] = g
			all = append(all, key)
		}
	}
	for _, edge := range parentEdges {
		child := map[string][]*group{}
		parent := map[string][]*group{}
		for _, id := range baseOrder {
			g := bases[id]
			if g.kind != "view" {
				continue
			}
			if g.scope == edge.Child {
				child[g.normalized] = append(child[g.normalized], g)
			}
			if g.scope == edge.Parent {
				parent[g.normalized] = append(parent[g.normalized], g)
			}
		}
		for n, cgs := range child {
			pgs := parent[n]
			for _, cg := range cgs {
				for _, pg := range pgs {
					g := &group{kind: "local-parent", scope: edge.Child + "->" + edge.Parent, normalized: n, evidence: edge.Evidence, key: cg.key, rows: append(append([]bindingRow{}, cg.rows...), pg.rows...)}
					id := "local-parent\x00" + g.scope + "\x00" + n
					if _, exists := bases[id]; !exists {
						bases[id] = g
						all = append(all, id)
					}
				}
			}
		}
	}
	out := make([]collision, 0)
	for _, id := range all {
		g := bases[id]
		if len(g.rows) < 2 {
			continue
		}
		orderedRows := append([]bindingRow(nil), g.rows...)
		sort.SliceStable(orderedRows, func(i, j int) bool { return orderedRows[i].Declaration < orderedRows[j].Declaration })
		ids, verbs, origins, order := []string{}, []string{}, []string{}, []int{}
		for _, r := range orderedRows {
			ids = append(ids, r.ID)
			verbs = append(verbs, r.Verb)
			origins = append(origins, r.View)
			order = append(order, r.Declaration)
		}
		class := "different-id-different-verb"
		if allSame(ids) && allSame(verbs) {
			class = "same-id-same-verb-alias"
		} else if !allSame(ids) && allSame(verbs) {
			class = "different-id-shared-verb"
		} else if allSame(ids) {
			class = "same-id-different-verb"
		}
		out = append(out, collision{Platform: platform, ScopeKind: g.kind, Scope: g.scope, Key: g.key, Class: class, Origins: origins, IDs: ids, Verbs: verbs, Order: order, Evidence: g.evidence})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScopeKind != out[j].ScopeKind {
			return out[i].ScopeKind < out[j].ScopeKind
		}
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		if out[i].Key.Name != out[j].Key.Name {
			return out[i].Key.Name < out[j].Key.Name
		}
		if out[i].Key.Str != out[j].Key.Str {
			return out[i].Key.Str < out[j].Key.Str
		}
		return out[i].Key.Mod < out[j].Key.Mod
	})
	return out
}

func canonicalKey(key keyRow) keyRow {
	canonical := gocui.NormalizeCommandKey(gocui.NewKey(gocui.KeyName(key.Name), key.Str, gocui.Modifier(key.Mod)))
	return keyRow{
		Name:  int(canonical.KeyName()),
		Str:   canonical.Str(),
		Mod:   int(canonical.Mod()),
		Label: config.LabelForKey(canonical),
	}
}

func allSame(values []string) bool {
	if len(values) == 0 {
		return true
	}
	for _, v := range values[1:] {
		if v != values[0] {
			return false
		}
	}
	return true
}

func normalizeRawKey(key keyRow) string {
	canonical := canonicalKey(key)
	return fmt.Sprintf("%d\x00%s\x00%d", canonical.Name, canonical.Str, canonical.Mod)
}
