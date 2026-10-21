# Static key and menu factory inventory

Regenerate from the repository root with:

```sh
go run new-features/inventory/generate.go
```

`inventory.json` is a mechanical Go AST inventory of non-test source under
`pkg/gui` and `pkg/gocui`. It records explicit `Binding` and `MenuItem`
composite literals, plus elided keyed literals directly inside typed slices
such as `[]*types.Binding` and `[]*types.MenuItem`. An entry's `file` and
`line` are audit references only; they are not persistent action IDs and do
not imply a semantic verb or target.

For each factory, `fields` contains raw Go expressions for the requested
fields when those fields appear directly in the literal. `enclosingCondition`
records the enclosing `if` branch expressions; an else branch is represented
as `!(condition)`, including nested branches in outer-to-inner order. The
generator does not evaluate expressions, follow variables, expand loops, invoke helper
functions, resolve runtime context, or infer values returned by dynamic
factories. Consequently this static factory list is not exhaustive runtime
expansion. Main review must inspect dynamic/context-dependent targets and
confirm behavior, aliases, availability, semantic verb, and dispatch
precedence before assigning identities.

The inventory separately lists direct `SetKeybinding` and
`SetEditKeybindings` calls. `hardcodedDispatchCases` contains all Go `case`
labels in the scanned source, including labels that are not key dispatch;
these are intentionally broad leads for review, not semantic classification.
Other hardcoded keys may be expressed through conditionals or helper code
without a case label. The low-level call inventory includes raw call
arguments, not the effective runtime registrations.

The three platform defaults serialize `config.GetDefaultConfigForPlatform`
`Keybinding` values for Linux, macOS, and Windows. They show defaults only,
not loaded user/repository configuration. JSON output is deterministically
ordered by source path and line for source records and by key for maps. It has
no timestamps. Factory classification is based on explicit `types.Binding` or
`types.MenuItem` types, or on keyed elided literals directly in slices of
those types. Classification evidence is included per record so that the
matching rule is visible.

Run the standalone tool tests explicitly (their build tags exclude them from
the application):

```sh
go test new-features/inventory/generate.go new-features/inventory/generate_test.go
go test new-features/inventory/check_annotations.go new-features/inventory/check_annotations_test.go
```

`check_annotations.go -base <commit> <paths...>` compares baseline/current ASTs
after removing `CommandID`, `CommandIDFunc`, `Verb` and `VerbFunc` keyed fields
from composite literals. It ignores comments, formatting and positions, not
handlers, keys, guards or descriptions. Use it only for annotation-only paths;
review intentional integration changes separately.
