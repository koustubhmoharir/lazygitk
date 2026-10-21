//go:build ignore

// Run from the repository root with: go run new-features/inventory/generate.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
)

type record struct {
	Kind      string            `json:"kind"`
	File      string            `json:"file"`
	Line      int               `json:"line"`
	Owner     string            `json:"owner,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Condition string            `json:"enclosingCondition,omitempty"`
	Evidence  string            `json:"classificationEvidence,omitempty"`
}

type inventory struct {
	Notice          string                 `json:"notice"`
	Factories       []record               `json:"factories"`
	LowLevelCalls   []record               `json:"lowLevelKeyCalls"`
	HardcodedCases  []record               `json:"hardcodedDispatchCases"`
	PlatformDefault map[string]interface{} `json:"platformDefaults"`
}

var wanted = map[string]bool{"Keys": true, "Handler": true, "OnPress": true, "Description": true, "Label": true, "LabelColumns": true, "GetDisabledReason": true, "DisabledReason": true, "Tag": true, "OpensMenu": true, "DisplayOnScreen": true, "ViewName": true, "CommandID": true, "CommandIDFunc": true, "Verb": true, "VerbFunc": true}

func main() {
	inv := inventory{Notice: "Static source inventory only. Locations are audit references, not persistent action IDs. Dynamic/context-dependent factory results and runtime expansion are not guessed.", PlatformDefault: map[string]interface{}{}}
	for _, platform := range []string{"linux", "darwin", "windows"} {
		inv.PlatformDefault[platform] = config.GetDefaultConfigForPlatform(platform).Keybinding
	}
	fs := token.NewFileSet()
	for _, root := range []string{"pkg/gui", "pkg/gocui"} {
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			collect(fs, path, f, &inv)
			return nil
		}); err != nil {
			fatal(err)
		}
	}
	sort.Slice(inv.Factories, func(i, j int) bool { return less(inv.Factories[i], inv.Factories[j]) })
	sort.Slice(inv.LowLevelCalls, func(i, j int) bool { return less(inv.LowLevelCalls[i], inv.LowLevelCalls[j]) })
	sort.Slice(inv.HardcodedCases, func(i, j int) bool { return less(inv.HardcodedCases[i], inv.HardcodedCases[j]) })
	b, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		fatal(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile("new-features/inventory/inventory.json", b, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote inventory: %d factories, %d low-level calls, %d hardcoded dispatch cases\n", len(inv.Factories), len(inv.LowLevelCalls), len(inv.HardcodedCases))
}

func collect(fs *token.FileSet, path string, f *ast.File, inv *inventory) {
	parents := map[ast.Node]ast.Node{}
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		if len(stack) > 0 {
			parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})
	ownerAt := func(pos token.Pos) string {
		owner := ""
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if ok && fn.Pos() <= pos && pos <= fn.End() {
				owner = fn.Name.Name
				if fn.Recv != nil {
					owner = receiverName(fn) + "." + owner
				}
			}
		}
		return owner
	}
	conditionAt := func(node ast.Node) string {
		var ancestors []ast.Node
		for parent := parents[node]; parent != nil; parent = parents[parent] {
			ancestors = append(ancestors, parent)
		}
		var conditions []string
		for i := len(ancestors) - 1; i >= 0; i-- {
			if branch, ok := ancestors[i].(*ast.IfStmt); ok {
				child := node
				if i > 0 {
					child = ancestors[i-1]
				}
				condition := formatExpr(fs, branch.Cond)
				if child == branch.Body {
					conditions = append(conditions, "("+condition+")")
				} else if branch.Else != nil && child == branch.Else {
					conditions = append(conditions, "!("+condition+")")
				}
			}
		}
		return strings.Join(conditions, " && ")
	}
	seen := map[token.Pos]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if s, ok := c.Fun.(*ast.SelectorExpr); ok && (s.Sel.Name == "SetKeybinding" || s.Sel.Name == "SetEditKeybindings") {
				inv.LowLevelCalls = append(inv.LowLevelCalls, record{Kind: s.Sel.Name, File: path, Line: fs.Position(c.Pos()).Line, Owner: ownerAt(c.Pos()), Condition: conditionAt(c), Fields: map[string]string{"arguments": formatExpr(fs, c)}})
			}
		}
		if c, ok := n.(*ast.CaseClause); ok {
			for _, e := range c.List {
				inv.HardcodedCases = append(inv.HardcodedCases, record{Kind: "case", File: path, Line: fs.Position(e.Pos()).Line, Owner: ownerAt(e.Pos()), Condition: conditionAt(c), Fields: map[string]string{"key": formatExpr(fs, e)}})
			}
		}
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		kind := ""
		if isTargetType(cl.Type, "Binding") {
			kind = "Binding"
		} else if isTargetType(cl.Type, "MenuItem") {
			kind = "MenuItem"
		} else {
			p := parents[cl]
			if outer, ok := p.(*ast.CompositeLit); ok {
				if arr, ok := outer.Type.(*ast.ArrayType); ok && arr.Len == nil && isTargetType(arr.Elt, "Binding") {
					kind = "Binding"
				}
				if arr, ok := outer.Type.(*ast.ArrayType); ok && arr.Len == nil && isTargetType(arr.Elt, "MenuItem") {
					kind = "MenuItem"
				}
			}
		}
		if kind != "Binding" && kind != "MenuItem" {
			return true
		}
		if seen[cl.Pos()] {
			return true
		}
		seen[cl.Pos()] = true
		fields := map[string]string{}
		for _, elt := range cl.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				if id, ok := kv.Key.(*ast.Ident); ok && wanted[id.Name] {
					fields[id.Name] = formatExpr(fs, kv.Value)
				}
			}
		}
		inv.Factories = append(inv.Factories, record{Kind: kind, File: path, Line: fs.Position(cl.Pos()).Line, Owner: ownerAt(cl.Pos()), Fields: fields, Condition: conditionAt(cl), Evidence: "explicit types.Binding/types.MenuItem or elided keyed literal in typed collection"})
		return true
	})
}

func typeName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return typeName(x.X) + "." + x.Sel.Name
	case *ast.StarExpr:
		return typeName(x.X)
	case *ast.ArrayType:
		return typeName(x.Elt)
	case *ast.CallExpr:
		return typeName(x.Fun)
	}
	return ""
}

func isTargetType(e ast.Expr, name string) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "types"
}

func receiverName(f *ast.FuncDecl) string {
	if f.Recv == nil || len(f.Recv.List) == 0 {
		return ""
	}
	return typeName(f.Recv.List[0].Type)
}

func formatExpr(fs *token.FileSet, e ast.Node) string {
	var b bytes.Buffer
	if e == nil {
		return ""
	}
	if err := formatNode(&b, fs, e); err != nil {
		return ""
	}
	return strings.TrimSpace(b.String())
}

func formatNode(b *bytes.Buffer, fs *token.FileSet, n ast.Node) error { return format.Node(b, fs, n) }

func less(a, b record) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	return a.Line < b.Line
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
