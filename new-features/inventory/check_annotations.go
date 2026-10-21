//go:build ignore

// check_annotations compares selected working Go files with a Git baseline,
// ignoring only command identity and verb metadata in composite literals.
// Run from the repository root with:
// go run new-features/inventory/check_annotations.go -base <commit> <files...>
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

var metadataFields = map[string]bool{
	"CommandID": true, "CommandIDFunc": true, "Verb": true, "VerbFunc": true,
}

func main() {
	base := flag.String("base", "", "Git commit used as the baseline")
	flag.Parse()
	if *base == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: check_annotations.go -base <commit> <tracked .go paths...>")
		os.Exit(2)
	}
	failed := false
	for _, path := range flag.Args() {
		if err := check(*base, path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failed = true
		} else {
			fmt.Printf("%s: annotation-only changes (or unchanged)\n", path)
		}
	}
	if failed {
		os.Exit(1)
	}
}

func check(base, path string) error {
	if !strings.HasSuffix(path, ".go") {
		return fmt.Errorf("path is not a .go file")
	}
	tracked, err := exec.Command("git", "ls-files", "--error-unmatch", "--", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("not a tracked path: %s", strings.TrimSpace(string(tracked)))
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read current file: %w", err)
	}
	baseline, err := exec.Command("git", "show", base+":"+path).Output()
	if err != nil {
		return fmt.Errorf("read baseline %s: %w", base, err)
	}
	oldAST, err := parseAndNormalize(path, baseline)
	if err != nil {
		return fmt.Errorf("parse baseline: %w", err)
	}
	newAST, err := parseAndNormalize(path, current)
	if err != nil {
		return fmt.Errorf("parse current file: %w", err)
	}
	if !bytes.Equal(oldAST, newAST) {
		return fmt.Errorf("non-metadata AST change detected (handlers, keys, guards, descriptions, and all other behavior are compared)")
	}
	return nil
}

func parseAndNormalize(filename string, src []byte) ([]byte, error) {
	f, err := parser.ParseFile(token.NewFileSet(), filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		filtered := lit.Elts[:0]
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if ok {
				ident, keyed := kvKey(kv)
				if keyed && metadataFields[ident] {
					continue
				}
			}
			filtered = append(filtered, elt)
		}
		lit.Elts = filtered
		return true
	})
	compactEmptySlices(reflect.ValueOf(f))
	var out bytes.Buffer
	err = ast.Fprint(&out, token.NewFileSet(), f, func(name string, value reflect.Value) bool {
		if value.IsValid() && value.Type() == reflect.TypeOf(token.Pos(0)) {
			return false
		}
		return true
	})
	return out.Bytes(), err
}

func kvKey(kv *ast.KeyValueExpr) (string, bool) {
	if kv == nil {
		return "", false
	}
	id, ok := kv.Key.(*ast.Ident)
	if !ok {
		return "", false
	}
	return id.Name, true
}

// AST field slices that become empty after metadata removal have no structural
// distinction between nil and empty; compact them before printing.
func compactEmptySlices(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			compactEmptySlices(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() {
				compactEmptySlices(v.Field(i))
			}
		}
	case reflect.Slice:
		if v.Len() == 0 && v.CanSet() {
			v.Set(reflect.Zero(v.Type()))
			return
		}
		for i := 0; i < v.Len(); i++ {
			compactEmptySlices(v.Index(i))
		}
	}
}
