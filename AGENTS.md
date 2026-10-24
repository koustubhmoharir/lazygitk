# Working on this fork

Start with [Tracker.md](new-features/Tracker.md) for the current item and remaining
tasks, and [Roadmap.md](new-features/Roadmap.md) for scope and agreed behavior.
Follow the tracker's workflow: plan only the current item, remove completed
tasks in their implementation commits, and advance the tracker and roadmap
together when an item is finished.

- Keep upstream merges feasible. Prefer adding code in fork-specific modules
  and minimize the existing locations that need modification. When a fork
  codepath replaces an upstream codepath, comment at the integration point to
  explain what it replaces and why. Actively maintain
  [MergeGuide.md](new-features/MergeGuide.md) in the same commits as changes to
  fork integration points, replacements, or merge requirements.
- There is no hurry. Aim for excellent code quality; always prefer a correct,
  difficult solution to a hack. Keep designs clear and maintainable.
- Focus on usability: this fork exists to solve upstream usability problems.
- The features should be designed to keep the application user in control.
  Present meaningful choices in modal flows with sensible defaults;
  do not let the application silently choose behavior for the user. Respect
  choices already recorded in the roadmap and tracker.
- For any design or implementation question, first consult TortoiseGit's
  behavior, documentation, or source as a guide. Present unresolved choices
  to the user before implementing them.
- Use the `justfile` recipes for generation, formatting, builds, tests and lint.
  Keep commits small and verified, including their tracker and merge-guide
  updates. Commit completed work; refine existing commits with reviewable
  `fixup!` commits and leave them unsquashed.
- Do not create pull requests.
