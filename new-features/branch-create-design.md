# Captured branch and tag creation (current item 7)

Retain the existing create-and-checkout action. Pointer-only create/move remains
its separate action; do not turn a failed checkout into a ref move or stash.
[TortoiseGit](https://tortoisegit.org/docs/tortoisegit/tgit-dug-branchtag.html)
provides explicit checkout and three tracking choices: configured policy,
no tracking, and direct tracking. Preserve these choices in a modal review.

Copy the selected full ref (or immutable commit OID), original backend and UI
repository generation before inspecting the source. Capture native source tip,
current HEAD, effective configuration and destination state. Name entry is
literal, without sanitizing user input. Existing names fail normal creation;
reviewed overwrite remains the separate create/move or checkout-as-branch action.

For branch creation, expose tracking policy instead of silently choosing it
from whether the entered name matches a suggestion. The matching suggestion may
select configured policy initially; another name may select no tracking, as in
the upstream behavior, but show that choice before execution. Native no-track
creation uses the exact captured commit. Native configured/direct tracking needs
the original full branch start-point so Git can infer its native tracking and
inheritance policy. Recheck its exact tip, HEAD and effective configuration
immediately before launch; native ref resolution after that check uses Git's
normal concurrency model. Do not simulate native policy by parsing a subset of
settings, create temporary refs, or add application locks, retries or rollback.
For an OID source, configured policy naturally has no branch metadata to infer;
direct tracking is unavailable. Retain native errors for unsupported sources.
“Do not set up upstream tracking” suppresses automatic setup; as in native Git,
it does not remove pre-existing orphan tracking configuration for that name.
Checkout protects ignored files with `--no-overwrite-ignore` and carries harmless
dirty/untracked files normally. Refresh actual state even if a hook fails after
checkout. No automatic stash is offered.

For tags, the source command uses the captured commit OID. Retain lightweight
versus annotated selection from the description and native signing settings,
with terminal signing support. Capture the exact old tag object, not only its
peeled commit, for overwrite approval; refuse symbolic destinations. Revalidate
source/current HEAD/configuration/destination at worker and terminal-launch
boundaries. Never reinterpret a changed tag as permission to overwrite it.

Every queued UI, command and refresh completion stays bound to the original
backend and repository generation. Repeated approval must not launch twice.
These are read-only preparations followed by native Git commands, with native
errors and actual partial state. See [the native probe](branch-create-probes.md)
for the verified start-point/tracking distinction. Native creation plans and
focused checks are implemented and the captured GUI hooks are integrated.
Nine branch/commit/stash terminal scenarios and the tag terminal group pass;
final item 7 acceptance remains in the tracker.
