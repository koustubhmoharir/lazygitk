# Keymap audit snapshot

Generate `bindings.json` from the repository root with:

```sh
env GOMAXPROCS=2 go run new-features/keymap/generate.go > new-features/keymap/bindings.json
```

The generator uses `go run new-features/keymap/generate.go`. The
generator instantiates Lazygit's exported GUI cheatsheet binding path for
Linux, macOS (`darwin`), and Windows. These are instantiated default
snapshots, not exhaustive coverage of every mode, dynamic menu, or live
availability state.

Rows retain declaration order and record static command ID, verb, description,
view, tag, direct default key labels, and complete sequential command paths.
Direct keys and every key in each path include their raw `Name` integer, `Str`,
and `Mod` tuple because labels such as `unknown` can represent distinct keys.
Dynamic callback presence and disabled-reason guard presence are flags; the
callbacks and guards are never invoked.

The cleared-key section clears configured legacy keybinding leaves and adds an
explicit empty `commandBindings` override for every projected static command ID.
This prevents catalogue defaults from reappearing after legacy fields are
cleared, while retaining each action's identity and descriptive metadata.
Cleared rows are matched back by view, static ID, static verb, and ordinal
within that identity; they do not rely on declaration indexes staying aligned
when rows disappear.
The output does not infer which configuration field supplied a key. Record
field provenance in the separate manual audit when that attribution is needed.

Collision key folding is deliberately narrow: ASCII letters in a plain
single-character key or Shift plus one ASCII letter fold to lowercase in the
raw tuple used for comparison.
Punctuation, control combinations, custom key labels, and native key labels
remain unchanged. Collision groups compare complete paths, including
sequential paths. Two paths sharing only an initial key are not collisions.
Groups preserve each origin and declaration order.
They are audit candidates, not assertions that every overlap is a bug.

Each collision has an explicit `scopeKind` (`view`, `global`, `local-global`,
or `local-parent`) and a structured raw key tuple for every path step. The
initialized GUI records the actual `ParentView` edges from `menuFilter` and `menuFilterFrame` to
`menu`; these are filter-mode overlap candidates and are inactive by default.
Collision candidates are generated from those observed edges, rather than from
a duplicated hardcoded edge list.
Logical context-parent chains are recorded separately and do not imply key
bubbling. Parent overlap candidates are not claims of simultaneous runtime
availability.

Context entries contain the initialized context's key, kind, and input view
for the matching registered view. GUI state files are created under a private
`MkdirTemp` directory and removed after each snapshot. No focus mutation or
runtime handler call is part of generation.
