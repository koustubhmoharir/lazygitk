# Working on this fork

The fork's documents live in [docs-master/fork/](docs-master/fork/):

- [Tracker.md](docs-master/fork/Tracker.md): the current work, its agreed
  scope and remaining tasks.
- [Design.md](docs-master/fork/Design.md): the UX principles that govern
  every change.
- [Guide.md](docs-master/fork/Guide.md): every key and how behaviour differs
  from upstream lazygit, from the user's side.
- [MergeGuide.md](docs-master/fork/MergeGuide.md): where the fork changes
  upstream code and what to check when merging upstream.

Start with the tracker and follow its workflow.

- Keep upstream merges feasible. Prefer adding code in fork-specific modules
  and minimize the existing locations that need modification. When a fork
  codepath replaces an upstream codepath, comment at the integration point to
  explain what it replaces and why. Update the merge guide in the same commits
  as changes to fork integration points, replacements, or merge requirements.
- Update the user guide in the same commits as changes to keys or
  user-visible behaviour, and the design principles when a decision sets or
  changes a rule.
- There is no hurry. Aim for excellent code quality; always prefer a correct,
  difficult solution to a hack. Keep designs clear and maintainable.
- Focus on usability: this fork exists to solve upstream usability problems.
- The features should be designed to keep the application user in control.
  Present meaningful choices in modal flows with sensible defaults;
  do not let the application silently choose behavior for the user. Respect
  choices already recorded in the design principles, user guide and tracker.
- For any design or implementation question, first consult TortoiseGit's
  behavior, documentation, or source as a guide. Present unresolved choices
  to the user before implementing them.
- Use the `justfile` recipes for generation, formatting, builds, tests and lint.
  Keep commits small and verified, including their tracker and document
  updates. Commit completed work; refine existing commits with reviewable
  `fixup!` commits and leave them unsquashed.
- Do not create pull requests.
