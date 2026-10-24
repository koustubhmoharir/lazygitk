package git_commands

import (
	"strings"
	"testing"
)

func combineOID(n byte) string { return strings.Repeat(string(n), 40) }

func combineHistory() []CombineHistoryCommit {
	return []CombineHistoryCommit{
		{OID: combineOID('a'), Parents: []string{combineOID('0')}, Message: "first\n\nfirst body"},
		{OID: combineOID('b'), Parents: []string{combineOID('a')}, Message: "selected two\nbody"},
		{OID: combineOID('c'), Parents: []string{combineOID('b')}, Message: "keep this"},
		{OID: combineOID('d'), Parents: []string{combineOID('c')}, Message: "selected four"},
		{OID: combineOID('e'), Parents: []string{combineOID('d')}, Message: "head"},
	}
}

func TestPlanLinearCombineBuildsAncestorOrderedTodo(t *testing.T) {
	history := combineHistory()
	plan, err := PlanLinearCombine(history, []string{combineOID('d'), combineOID('a'), combineOID('b')})
	if err != nil {
		t.Fatal(err)
	}
	if plan.KeeperOID() != combineOID('a') || plan.BaseOID() != combineOID('0') || plan.HeadOID() != combineOID('e') {
		t.Fatalf("unexpected plan endpoints: keeper=%s base=%s head=%s", plan.KeeperOID(), plan.BaseOID(), plan.HeadOID())
	}
	wantTodo := strings.Join([]string{
		"reword " + combineOID('a'), "fixup " + combineOID('b'),
		"fixup " + combineOID('d'), "pick " + combineOID('c'), "pick " + combineOID('e'),
	}, "\n") + "\n"
	if plan.Todo() != wantTodo {
		t.Fatalf("todo mismatch\n got: %q\nwant: %q", plan.Todo(), wantTodo)
	}
	wantDescription := "first\n\nfirst body\n\nselected two\nbody\n\nselected four"
	if plan.Summary() != "first" || plan.Description() != wantDescription {
		t.Fatalf("unexpected messages: summary=%q description=%q", plan.Summary(), plan.Description())
	}
	if got := plan.SelectedOIDs(); len(got) != 3 || got[0] != combineOID('a') || got[1] != combineOID('b') || got[2] != combineOID('d') {
		t.Fatalf("selected OIDs are not ancestor ordered: %v", got)
	}
}

func TestPlanLinearCombinePlacesAllFixupsBeforeUnselectedPicks(t *testing.T) {
	history := []CombineHistoryCommit{
		{OID: combineOID('a'), Parents: []string{combineOID('0')}, Message: "A"},
		{OID: combineOID('b'), Parents: []string{combineOID('a')}, Message: "B"},
		{OID: combineOID('c'), Parents: []string{combineOID('b')}, Message: "C"},
		{OID: combineOID('d'), Parents: []string{combineOID('c')}, Message: "D"},
	}
	plan, err := PlanLinearCombine(history, []string{combineOID('a'), combineOID('c')})
	if err != nil {
		t.Fatal(err)
	}
	want := "reword " + combineOID('a') + "\nfixup " + combineOID('c') + "\npick " + combineOID('b') + "\npick " + combineOID('d') + "\n"
	if plan.Todo() != want {
		t.Fatalf("todo mismatch\n got: %q\nwant: %q", plan.Todo(), want)
	}
}

func TestPlanLinearCombineRootBaseAndPreservesEmptyMessages(t *testing.T) {
	history := []CombineHistoryCommit{
		{OID: combineOID('a'), Message: ""},
		{OID: combineOID('b'), Parents: []string{combineOID('a')}, Message: "second\n\nbody\n"},
	}
	plan, err := PlanLinearCombine(history, []string{combineOID('b'), combineOID('a')})
	if err != nil {
		t.Fatal(err)
	}
	if plan.BaseOID() != "" || plan.Summary() != "" || plan.Description() != "\n\nsecond\n\nbody\n" {
		t.Fatalf("root/messages not preserved: base=%q summary=%q description=%q", plan.BaseOID(), plan.Summary(), plan.Description())
	}
}

func TestPlanLinearCombineMergeRules(t *testing.T) {
	history := combineHistory()
	history[0].Parents = []string{combineOID('0'), combineOID('9')}
	if _, err := PlanLinearCombine(history, []string{combineOID('b'), combineOID('d')}); err != nil {
		t.Fatalf("merge before keeper should be allowed: %v", err)
	}
	history[3].Parents = []string{combineOID('c'), combineOID('9')}
	if _, err := PlanLinearCombine(history, []string{combineOID('b'), combineOID('d')}); err == nil {
		t.Fatal("merge in rewritten suffix accepted")
	}
}

func TestPlanLinearCombineRejectsInvalidHistoryAndSelection(t *testing.T) {
	cases := map[string]func([]CombineHistoryCommit) ([]CombineHistoryCommit, []string){
		"bad history oid": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			h[0].OID = "bad"
			return h, []string{combineOID('a'), combineOID('b')}
		},
		"non-hex history oid": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			h[0].OID = strings.Repeat("g", 40)
			return h, []string{combineOID('a'), combineOID('b')}
		},
		"mixed history format": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			h[1].OID = strings.Repeat("b", 64)
			return h, []string{combineOID('a'), combineOID('b')}
		},
		"duplicate history": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			h[1].OID = h[0].OID
			return h, []string{combineOID('a'), combineOID('b')}
		},
		"broken adjacency": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			h[2].Parents[0] = combineOID('a')
			return h, []string{combineOID('a'), combineOID('b')}
		},
		"unreachable selection": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			return h, []string{combineOID('a'), combineOID('9')}
		},
		"duplicate selection": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			return h, []string{combineOID('a'), combineOID('a')}
		},
		"mixed selection format": func(h []CombineHistoryCommit) ([]CombineHistoryCommit, []string) {
			return h, []string{combineOID('a'), strings.Repeat("b", 64)}
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			history, selected := build(combineHistory())
			if _, err := PlanLinearCombine(history, selected); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestPlanLinearCombineCopiesInputsAndOutputs(t *testing.T) {
	history := combineHistory()
	plan, err := PlanLinearCombine(history, []string{combineOID('a'), combineOID('b')})
	if err != nil {
		t.Fatal(err)
	}
	wantTodo := plan.Todo()
	history[0].OID = combineOID('z')
	history[0].Parents[0] = combineOID('y')
	oids := plan.SelectedOIDs()
	oids[0] = combineOID('x')
	if plan.KeeperOID() != combineOID('a') || plan.Todo() != wantTodo || plan.SelectedOIDs()[0] != combineOID('a') {
		t.Fatal("input/output mutation changed plan")
	}
}

func TestPlanLinearCombineCanonicalizesObjectIdentity(t *testing.T) {
	history := combineHistory()
	history[0].OID = strings.ToUpper(history[0].OID)
	plan, err := PlanLinearCombine(history, []string{strings.ToUpper(combineOID('a')), combineOID('d')})
	if err != nil {
		t.Fatal(err)
	}
	if plan.KeeperOID() != combineOID('a') {
		t.Fatalf("keeper = %s", plan.KeeperOID())
	}
	if _, err := PlanLinearCombine(history, []string{combineOID('a'), strings.ToUpper(combineOID('a'))}); err == nil {
		t.Fatal("same object in different case was treated as two commits")
	}
}
