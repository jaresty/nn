---
name: nn-guide
description: "Use when you need an nn command, flag, syntax, semantics, usage pattern, or command-reference lookup; for iterative or human-driven graph navigation, use nn-navigate instead. Load with `nn skills get nn-guide`."
when_to_use: "When looking up nn command syntax, flags, types, schemas, output semantics, or one-shot usage patterns. Human-driven iterative graph navigation dispatches to nn-navigate."
---

# nn-guide

Reference for `nn` commands, flags, and LLM usage patterns.

## Global flags (all commands)

```
--json          Machine-readable JSON output
--no-color      Disable ANSI color
-q, --quiet     Suppress progress/info output
--notebook      Select a non-default notebook (name from config)
```

## nn new

Create a new note.

```
nn new --title TEXT --type TYPE [--tags TEXT] [--content TEXT] [--no-edit]
       [--link-to ID --annotation TEXT]  # repeatable; --annotation must match --link-to count
       [--applies-when TEXT]
       [--representation ontology|taxonomy|axiom]
       [--from-stdin]
       [--from-file PATH]
       [--expires YYYY-MM-DD]
       [--expires-when "condition text"]
       [--no-suggest]
       [--check]

nn new --quick --title TEXT [--no-edit]
```

- `--type` is required (except with `--quick`): `concept | argument | model | hypothesis | observation | question | protocol`
- `--no-edit` skips `$EDITOR` launch (always use in non-TTY/LLM context)
- `--content TEXT` sets the note body directly
- `--from-stdin` reads the note body from stdin
- `--from-file PATH` scaffolds the note body from `nn ast` output for a source file (sets title to filename if not given)
- `--quick` — shorthand capture: sets type=observation, status=draft, content empty; skips type requirement. Use for fast capture when the note will be refined later.
- `--no-suggest` — skips the link/tag suggestion prompt after creation. Use in non-interactive or batch contexts.
- `--link-to ID --annotation TEXT` — repeatable; add multiple links at creation time. Each `--link-to` must have a matching `--annotation` (paired by position).
- `--check` — opt-in; runs representation graph validation after creation. No-op if the note has no `representation` field.

### Choosing a type

The five types cover the epistemic roles a note can play (after Ahrens, *How to Take Smart Notes*):

| Type | Use when the note… | Example title |
|---|---|---|
| `concept` | defines or explains a single idea, term, or principle | "The Atomicity Principle" |
| `argument` | makes a claim and supports it with reasoning | "Atomicity enables reuse across contexts" |
| `model` | describes a framework, pattern, or mental model | "The Zettelkasten as a second brain" |
| `hypothesis` | states an untested conjecture worth investigating | "Dense linking predicts note longevity" |
| `observation` | records a concrete fact, datum, or empirical finding | "Luhmann produced 90,000 notes over 40 years" |
| `question` | poses an open question that the graph should eventually answer | "Why did Luhmann avoid hierarchical folders?" |
| `protocol` | specifies an imperative procedure the LLM should follow in this notebook | "When creating a hypothesis, immediately link it to its source observation" |

**Decision heuristic:** if you're not sure, ask — *is this a definition (concept), a claim with support (argument), a framework (model), a guess to test (hypothesis), something I witnessed/measured (observation), an open question (question), or an operating instruction for the LLM (protocol)?* If none fit cleanly, the note may not be atomic yet.

## nn show

Print note content to stdout. Accepts a full ID or a title substring.

```
nn show <id-or-title> [--depth N] [--json] [--rules]
nn show --linked-from <id>
nn show --global
```

`--global` shows all global protocol notes (type:protocol with no outgoing `governs` links) in one command, each with the derivation instruction appended. Replaces the two-step `nn list --global --json` + `nn show <id>` pattern. Also appends a `## Reminders` block listing the body of any non-expired notes tagged `reminder`.

`--rules` appends a `## Rule violations` section listing rule violations whose subject is the shown note. This runs the rules engine over the whole notebook, so it is **opt-in** — default `nn show` skips it to stay fast. For a full-notebook violation sweep use `nn rules check` instead.

If the query doesn't match an ID exactly, `nn` searches note titles case-insensitively.
If multiple titles match, the command lists the candidates and exits with an error — use
the full ID to disambiguate.

**Graph neighborhood in plain-text output:** `nn show <id>` now includes two navigation aids in its plain-text output:

- **Resolved link titles:** the `## Links` section renders each outgoing link as `[[ID|Title]]` instead of bare `[[ID]]`, so target note titles are immediately readable without a separate lookup.
- **Backlinks section:** a `## Backlinks (N)` block is appended listing all notes that link to the shown note, each with its annotation. This replaces the need for a separate `nn backlinks <id>` call when exploring the local graph.

These additions appear only in plain-text output (`--json` is unaffected).

`--depth N` traverses outgoing links from the given note up to N hops, collecting all
reachable notes and printing them as a single concatenated Markdown document separated by
`---`. Useful for loading a coherent subgraph as context for an LLM.

```
nn show <id> --depth 2                 # root + 2 hops of outgoing links
nn show <id> --depth 1 --json          # JSON array with depth field per note
```

`--json` with `--depth` returns an array of note objects in BFS order, each with an added
`depth` field (0 = origin note, 1 = direct links, etc.).

## nn list

List and filter notes.

```
nn list [--tag TEXT] [--type TYPE] [--status STATUS] [--representation ontology|taxonomy|axiom]
        [--linked-from ID] [--linked-to ID] [--orphan] [--global] [--long]
        [--has-url] [--url-contains STRING]
        [--search TEXT] [--boost-recent] [--similar ID] [--sort FIELD] [--limit N] [--json]
        [--before DATE] [--rich] [--full] [--envelope]
```

`--boost-recent` boosts recently-modified notes in search result ranking. Requires `--search`. Use to surface recently-touched notes above older equally-relevant ones.

`--rich` includes `modified`, `link_count`, and `body_preview` fields in JSON output. Requires `--json`. Use when you need structured metadata without requesting specific `--fields`.

`--full` disables truncation of `excerpt` and `annotation` in JSON output. Requires `--json`. Use when full note body context is needed for downstream processing.

`--envelope` wraps search JSON output in a metadata envelope containing `query`, `result_count`, and `total_matching`. Requires `--json --search`. Use when the consumer needs result-set metadata alongside the notes.

`--before DATE` filters to notes modified before the given date (ISO 8601). Composes with `--since` to define a modification date range.

`--search TEXT` performs a ranked case-insensitive search across note title and body. Title matches rank above body matches. Notes with more inbound backlinks receive a log-scale centrality boost on top of BM25, so well-linked notes surface above equal-content orphans. The `score` field in `--json` output reflects both BM25 relevance and centrality.

`--similar ID` ranks all notes by BM25 similarity to the given note's title and body, excluding the note itself. Use for serendipitous discovery — find notes that share vocabulary with a given note but have no explicit link. Composes with `--status`, `--tag`, `--type`, `--limit`, `--json`. When `--similar` is active, `--sort` is ignored (similarity ranking takes precedence).

```
nn list --similar <id>                 # notes most similar to <id>
nn list --similar <id> --limit 5       # top 5 most similar
nn list --similar <id> --status permanent --json
```

`--sort FIELD` sorts results: `title` (alphabetical), `modified` (most-recently-modified first), `created` (default, most-recently-created first). Applied after filtering and ranking. Ignored when `--similar` is active.

`--global` returns protocol notes with no outgoing `governs` links — protocols that apply universally to the entire notebook rather than governing specific notes. Distinct from `--orphan`: a global protocol is intentionally universal, not forgotten.

`--long` filters to notes whose body exceeds the atomicity threshold (2000 chars). Use to find notes that have grown too large to split.

`--has-url` filters to notes containing at least one `http://` or `https://` URL. Use to find notes with external references.

`--url-contains STRING` filters to notes containing a URL that includes the given string. Only matches within actual URLs (requires `http://` or `https://` prefix) — bare text occurrences are ignored.

```
nn list --has-url                          # all notes with any URL
nn list --url-contains "github.com"        # notes linking to GitHub
nn list --has-url --search "auth"          # URL-containing notes matching "auth"
```

`--older-than N` filters to notes not modified in the last N days (age-based staleness). Uses the same thresholds as `nn show` freshness: 3 days = aging boundary, 14 days = stale boundary.

```
nn list --older-than 14            # notes not touched in 14+ days (stale tier)
nn list --older-than 3             # notes not touched in 3+ days (aging + stale)
nn list --older-than 14 --type concept --json
```

`--expired` filters to notes with an `expires` date set and in the past. Use to find notes marked for deletion.

`--has-expires` filters to notes with an `expires` date set (any date, past or future). Use to see all time-bounded notes.

**Expiration fields:** Notes support two complementary expiry mechanisms:

- `expires: YYYY-MM-DD` — date-based; automated by `--expired` filter and review
- `expires_when: "condition"` — semantic condition (e.g. "when the auth PR is merged"); surfaces in `nn review` under **Pending conditions** as a checklist for the reviewer to evaluate

`nn review` also surfaces **Expiry candidates**: observation notes older than 30 days with no expiry set and not permanent — use to identify notes that should have been time-bounded.

```
nn list --expired                  # notes past their expiration date
nn list --has-expires              # all notes with an expiration date
nn list --has-expires --json       # machine-readable expiring notes
```

`--has-open-items` filters to notes with at least one unchecked checkbox (`- [ ]`). Use to find notes with pending work items.

`--unblocked` filters to notes that have at least one `requires` link and whose required targets are all done (no unchecked checkboxes). A required target with no checkboxes is considered done vacuously — a warning is emitted to stderr in that case. Use to find notes that are now actionable because their prerequisites are complete.

`--no-inbound` filters to notes with zero inbound links. Stricter than `--orphan` (zero links in either direction) — use to find notes nothing references that still have outbound links.

`--unactioned` filters to notes that were accessed via `nn show` but have had no git commit touching their file since the last access. Advisory — requires `access.log`. Use to surface notes the LLM read but never updated. (Previously named `--stale`.)

Filters compose: `nn list --search "implicit" --type concept --sort modified` works as expected.

## nn check

Validate a note's representation subgraph structure.

```
nn check <id> [--as ontology|taxonomy|axiom] [--set-representation]
```

Reads `representation:` from the note's frontmatter and validates the graph structure of the subgraph rooted at that note. Exits non-zero with a descriptive error listing all violations. Use `--as` to override the frontmatter value for ad-hoc validation. `--set-representation` stamps the `representation` field on the note after passing validation.

**What it validates:**

- Traverses only outgoing links whose targets share the same `representation` value — links to notes with a different or absent representation are treated as leaf boundaries and not traversed
- Cycle detection — a cycle in the same-representation subgraph is always a violation
- Root must be `type: model`
- Non-root nodes must be `type: concept` or `type: argument`
- All violations are reported, not just the first

**Representation-specific checks:**

| Representation | Additional checks |
|---|---|
| `ontology` | connectivity only — no additional link type requirements |
| `taxonomy` | all outgoing links within the subgraph must use `refines` or `extends` |
| `axiom` | root must have at least one `grounded-by` link to a same-representation note |

**What it does not check:** section header presence within note bodies; cardinality, modality boundary, or distinguishing attributes (semantic — requires human review); links to notes outside the same-representation subgraph.

**Representation types:**

| Representation | Use when… |
|---|---|
| `ontology` | representing a domain's vocabulary and the relationships between its entities |
| `taxonomy` | partitioning a domain into exhaustive, mutually exclusive classes by a classification dimension |
| `axiom` | asserting a foundational constraint or invariant that other notes depend on — ruling out interpretations, scoping the claim |

```
nn check 20260411120045-3821                         # validate using frontmatter representation
nn check 20260411120045-3821 --as ontology           # validate as ontology regardless of frontmatter
nn check 20260411120045-3821 --as ontology --set-representation  # validate and stamp field
nn list --representation ontology                    # find all notes with representation: ontology
```

**Representation** is an optional orthogonal frontmatter field — independent of `type`. A note's `type` answers "what epistemic role does this play?" while `representation` answers "what structural contract must it satisfy?" Most notes omit it; set it when a note belongs to a formal structure like an ontology or taxonomy tree.

## nn rules

A small pure-Go **Datalog rules engine** that runs over facts derived from your notes. It is a *pure derivation layer*: facts and rules both come from the Markdown on disk (files are truth), and the engine only computes derived facts — it never writes.

```
nn rules check              # print every violation(ID, Reason); exits non-zero if any exist
nn rules query <predicate>  # print all derived facts for a predicate, e.g. nn rules query violation
nn rules list               # list rules loaded from notes (with note-ID provenance) + built-in count
```

**Auto-exposed facts.** Every note parse contributes these ground facts, forming a closed-world fact base ("no link ⇒ false"):

| Predicate | Meaning |
|---|---|
| `note(ID, Type, Status)` | one per note |
| `link(From, To, LinkType)` | one per outgoing link |
| `tag(ID, Tag)` | one per tag |
| `open_item(ID, Text)` | one per unchecked `- [ ]` checkbox |
| `expires(ID, YYYY-MM-DD)` | only if the note has an expiry |
| `representation(ID, Rep)` | only if the note has a representation |

**Built-in rules** re-express the `nn check` representation invariants (model root, concept/argument children, taxonomy `refines`/`extends`-only links, axiom `grounded-by`, no cycles) as `violation(ID, Reason)` clauses. `nn rules check` reports these across all notes.

**Built-in queryable predicates.** Beyond `violation`, the engine ships derivation rules you can query with `nn rules query <pred>` out of the box (run `nn rules list` to see the full set of predicate names):
- `reachable(A, B)` — B is reachable from A by following any links.
- `transitively_governs(P, N)` — protocol P governs N directly or down `refines`/`extends` chains.
- `done(X)` — X has no open `- [ ]` checkbox items (a satisfied task; vacuously true when it has no checkboxes, matching `nn list --unblocked`).
- `blocked(X)` — X has a `requires` link to a target that is not done, **or** to a target that is itself blocked (transitive dependency chain). Use `nn rules query blocked` to find every task waiting on an unfinished dependency, including indirect ones (which `nn list --unblocked` does not surface transitively).

**Writing your own rules.** Put a ` ```nn-rule ` fenced block in any note body — the rule versions in Git alongside the note. A `type:protocol` note can carry both its prose *and* a machine-checkable rule.

````markdown
```nn-rule
# transitive closure over governs → refines chains
transitively_governs(X, Y) :- link(X, Y, "governs").
transitively_governs(X, Z) :- transitively_governs(X, Y), link(Y, Z, "refines").

# flag any note that contradicts a permanent note
violation(N, "contradicts a permanent note") :-
    link(N, T, "contradicts"), note(T, _, "permanent").
```
````

**Syntax:** `head(args) :- body1(args), body2(args).` Uppercase args (or `_`) are variables (`_` is a wildcard that binds nothing); lowercase or `"quoted"` args are constants; a leading `!` negates a body literal (negation-as-failure). Comment lines start with `#`. Recursion (transitive closure) is supported and always terminates. Negation must be *stratified* — a ruleset where a predicate negatively depends on itself through a cycle is rejected with an error.

**Comparison** — a body literal `A != B` or `A = B` filters solutions to those where the two operands differ / match. Both operands must be bound by an earlier positive literal (an unbound comparison is a rule error). Example: `two_distinct(N) :- link(N, A, _), link(N, B, _), A != B.`

**Aggregation** — `count(V : source(...)) = K` binds `K` to the number of **distinct** values of `V` in the matching `source` facts, grouped by the head's other variables. The counted relation must be stratified (an aggregate that transitively depends on its own output is rejected). Example — "notes with 2+ distinct outbound links":

```nn-rule
outdeg(N, K) :- count(T : link(N, T, _)) = K.
well_connected(N) :- outdeg(N, K), K != "1", K != "0".
```

A **malformed rule** produces a warning naming the note ID and is skipped — it never prevents the note (or the rest of the ruleset) from loading.

## nn update-link / nn bulk-update-link

```
nn update-link <from-id> <to-id> [--annotation TEXT] [--type TYPE] [--status draft|reviewed]
nn bulk-update-link <from-id> --to <id> [--type TYPE] [--annotation TEXT] [--status draft|reviewed] [--to <id> ...]
```

Update annotation, type, and/or status of existing links in place — no unlink/relink needed. At least one change flag is required. Only specified fields are modified; unspecified fields are preserved.

`--status reviewed` signs off a draft link as human-endorsed. Use after verifying LLM-suggested links.

`nn bulk-update-link` applies all updates in a single git commit. `--type` and `--annotation` are paired with `--to` by position; if provided, their counts must match `--to`. `--status` applies to all `--to` targets.

## nn link / nn unlink / nn bulk-link / nn bulk-unlink

```
nn link <from-id-or-title> <to-id-or-title> --annotation "relationship description" --type TYPE [--status draft|reviewed]
nn unlink <from-id-or-title> <to-id-or-title> [--type TYPE]
nn bulk-link <from-id> --to <id> --annotation "..." --type TYPE [--status draft|reviewed] [--to <id> --annotation "..." --type TYPE]...
nn bulk-unlink <from-id> --to <id> [--to <id> ...] [--type TYPE ...]
```

`nn link` and `nn unlink` accept title substrings for both arguments. `nn bulk-link` and `nn bulk-unlink` require IDs; raw target IDs allow `bulk-unlink` to clean up links whose target note is missing.

`nn unlink --type TYPE` removes only edges of that type between the pair; without `--type`, all edges between the pair are removed. Multiple typed edges between the same pair are allowed (uniqueness is `(from, to, type)`). `nn bulk-unlink` preserves these semantics across repeated `--to` targets: omit `--type` to remove every edge type to each target, pass one `--type` to broadcast it, or pass one `--type` per target positionally.

> **`[[id]]` inline references are presentational only.** Writing `[[20260423-1234]]` in a note's prose body does not create a graph edge — it is not parsed by `nn graph`, `nn backlinks`, `nn path`, or `nn links`. Use `nn link` to create edges. This is intentional: the link graph is the authoritative record of relationships, not the prose.

Both `--annotation` and `--type` are required. A bare link is a schema violation.

Canonical types: `refines`, `contradicts`, `source-of`, `extends`, `supports`, `grounded-by`, `questions`, `governs`, `requires`.

Type definitions — choose the type whose definition matches the relationship you intend:

- `refines` — The source sharpens or narrows the target's claim without replacing it. Use when adding precision or a sub-case.
- `contradicts` — The source directly opposes the target's claim. Use when two notes cannot both be true.
- `source-of` — The target is derived from or authored by the source. Use for evidence, citations, or origin relationships.
- `extends` — The source adds structure or scope to the target without replacing it. Use when building on top of an existing model.
- `supports` — The source corroborates the target's claim. Use for independent evidence that strengthens but is not constitutive of the target.
- `grounded-by` — The source claim depends on the target observation as its evidential basis. Use when removing the target would make the source claim ungrounded (stronger than `supports`, which is corroborative only).
- `questions` — The source raises an unresolved challenge to the target. Use when the target's claim is uncertain or contested.
- `governs` ⚠ — The source is an operating protocol that constrains how the target domain is acted on. Only use when you intend the source note to act as an active protocol that governs LLM behavior.
- `requires` ⚠ — Note A requires note B means A cannot be acted on until B is complete. Use for task dependency only, not conceptual dependency. Completion is derived from B's checkbox state: done when all `- [x]` (or no checkboxes, vacuously). Used with `nn list --unblocked` to surface actionable notes.

`--status` defaults to `draft`. Pass `--status reviewed` when a human is explicitly creating and endorsing the link at creation time.

`nn bulk-link` creates all links in a single git commit. `--to`, `--annotation`, and `--type` are paired by position; counts must match. `--status` applies to all targets.

`nn bulk-unlink` validates the full flag shape before mutation, removes all requested links from one source-note snapshot, and writes at most one Git commit. Missing edges are idempotent no-ops.

## nn graph

```
nn graph [--json]
```

JSON output: `{ "nodes": [...], "edges": [...] }`

> **Walking the graph iteratively with a human?** Use the `nn-navigate` dispatch below. This section remains the command reference for graph syntax and output semantics.

### nn graph show (LLM-facing subgraph)

```
nn graph show [--focus <id>] [--depth N] [--direction outgoing|incoming|both] [--links TYPE,...] [--status STATUS,...] [--representation VALUE] [--zones] [--bodies] [--presentation-hints] [--format text|json|mermaid]
```

With `--focus`, renders a subgraph centered on `<id>`; BFS depth defaults to 2 and direction defaults to `outgoing`. `--links` accepts canonical link types, `--status` accepts `draft`, `reviewed`, or `permanent`, and `--representation` accepts one representation value. Filters constrain BFS expansion, so traversal does not pass through an ineligible intermediate note. Without --focus, graph show renders the full graph; explicitly supplied traversal flags, including `--depth`, require `--focus`. Mermaid output preserves stored edge orientation, includes link type and annotation labels, represents missing edge endpoints as deterministic placeholder nodes, and emits deterministic provenance comments for the normalized traversal options. Use `--format json` for structured output or `--format mermaid` for Markdown-compatible diagrams. Prefer focused output when exploring a note's neighborhood.

### nn graph routes (typed destination discovery)

```
nn graph routes --focus ID --links TYPES --search QUERY --limit N --json
```

Discovers query-relevant destinations that are reachable from `ID` by following stored links in their source→target direction, restricted to comma-separated canonical `TYPES`. Eligibility requires positive direct lexical BM25 evidence in the destination's title, body, or tags; graph-derived relevance such as a matching link annotation alone is insufficient. It scores every note against the full corpus, normalizes by the largest positive score, excludes the focus, unreachable notes, and destinations without direct evidence, then ranks eligible destinations by the normalized full-corpus `relevance_score` descending, shortest-hop count ascending, and destination ID ascending. Each JSON array entry contains `destination` (`id`, `title`, `relevance_score`) plus `witnesses`, an array of at most 3 aligned shortest node-and-edge paths. Ranking is unchanged by witness multiplicity. Witness selection prioritizes distinct first-hop nodes, then distinct complete edge type-sequence values, then the lexical full-path key. Within each witness, edge `i` connects `nodes[i]` to `nodes[i+1]`. `--json` is required.

### nn graph impact (explicit typed impact traversal)

```
nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json
```

Returns every note reached by a cycle-safe, depth-bounded BFS through the selected canonical link types. All flags are explicit and required: focus must be a nonblank existing ID, `TYPES` must be a nonblank comma-separated list without empty or unknown members, direction must be exactly `incoming` or `outgoing`, depth must be positive, and `--json` must be true. The JSON object reports `focus`, direction, normalized sorted unique `links`, requested depth, a `summary`, and `impacts` excluding focus sorted by depth then ID. Each impact has `node`, depth, and `witnesses`: at most 3 deterministic shortest focus→impact paths selected for first-hop diversity, then complete edge type-sequence diversity, then lexical full-path order. BFS retains every equal-shortest predecessor transition while reconstruction keeps a fixed bounded portfolio, so cycles and combinatorial convergence cannot cause unbounded path enumeration.

The `summary` contains `total_impacts`; `counts_by_depth` as a depth-ascending array of `{depth,count}`; `counts_by_first_hop` as `{id,title,count}` entries sorted by count descending then ID; and `witnesses_truncated`. Depth counts include each impact once. First-hop counts inspect only the selected witnesses and count an impact once per distinct `witness.nodes[1].id`; branches may overlap, so their counts can sum above `total_impacts`. `witnesses_truncated` is true if any impacted destination had more than 3 output candidates or if the bounded internal witness portfolio discarded candidates at that destination or along an inherited predecessor route. The aggregate does not alter any impact or witness entry.

`outgoing` follows each stored source→target edge. `incoming` traverses the reverse adjacency from stored target to stored source, but witness edges always retain their stored source/from → target/to orientation and annotation. Thus incoming `edges` point opposite consecutive focus→impact traversal nodes. Evidence examples: from an observation, `--links grounded-by --direction incoming` finds claims stored as claim→observation; from evidence stored as evidence→claim, `--links supports --direction outgoing` finds supported claims in stored direction.

`--zones` (requires `--focus`) annotates each node with the directional screen zone it occupies relative to the focus, derived from the node's direct link to the focus and that link's direction:

- **TOP** — what the focus answers to / depends on: `governs`/`supports` incoming, or `refines`/`extends`/`grounded-by` outgoing from the focus. Evidence rule: `grounded-by OUT → TOP`; `supports IN → TOP`.
- **BOTTOM** — what builds on the focus: `governs`/`supports` outgoing, or `refines`/`extends`/`grounded-by` incoming to the focus. Evidence rule: `grounded-by IN → BOTTOM`; `supports OUT → BOTTOM`.
- **LEFT** — tension: `contradicts`, `questions` (either direction).
- **RIGHT** — lateral provenance / task edges: `source-of`, `requires` (either direction).

In `--format json` each node gains a `zone` field (omitted when empty — the focus itself and nodes with no direct/unmapped link to the focus have no zone). In `--format text` nodes are grouped under `TOP`/`LEFT`/`RIGHT`/`BOTTOM` headers. This is the same zone mapping the interactive HTML graph viewer uses, so the CLI and the viewer stay in agreement.

`--bodies` includes each node's full body inline beneath its title (and, in `--format text`, its tags on a `tags:` line). In `--format json` each node gains a `body` field (omitted when empty). It applies to every text sub-view — zoned, focused-tree, and flat — and does not change which nodes are traversed; it only adds contents to nodes already in the result.

Every node also carries a **degree marker**: `--format text` appends `↑<out> ↓<in>` (outgoing/incoming link counts over the whole notebook) to each node line; `--format json` adds `out_degree`/`in_degree`. High inbound degree marks a **hub** (a load-bearing note many others depend on); near-zero marks a **leaf**. The zoned text view is preceded by a **key** mapping each zone to its link types. `--color auto|always|never` (default auto) prefixes each element with a **colored-circle emoji marker** — chosen over ANSI because emoji survive markdown and agent relay (an agent can retype 🟣 in prose; ANSI color dies there): **zone headers** by zone, **node titles** by note type (🟢 concept · 🟣 model · ⚪ observation · 🟡 hypothesis · 🔵 question · 🔴 argument · 🟦 protocol), and **edge labels** by link family (🔴 tension = contradicts/questions · 🔵 lateral = source-of/requires · 🟦 structural = refines/extends/grounded-by/governs/supports). With markers on, the zoned key also prints the type and link-family emoji legends. `auto` emits markers only to a TTY (so piped output and json stay clean and parseable); `always` forces them — **use `--color always` when relaying output to a human, since the emoji carry the type/family coloring through to chat where ANSI cannot**; `never` suppresses them.

### Human-driven iterative navigation: dispatch to `nn-navigate`

Use this dispatch when a human is driving an iterative graph walk, the conversation must retain a positioned focus across steps, or the request invokes teleport, orient, recenter, peek, scan, arrive, Back/Forward history, bookmarks, or navigation compaction. For a one-shot command or graph syntax/semantics lookup, remain in `nn-guide`.

Before navigating, run `nn skills list` if it has not yet run this session, then explicitly load the dedicated owner:

```bash
nn skills get nn-navigate
```

`nn-navigate` owns the detailed zoned navigation model, arrival scaling, route/impact/path overlays, teleport/scan/peek behavior, chooser and presentation discipline, and conversation-scoped history, bookmarks, and compaction. Do not conduct the iterative workflow from this command reference alone.

### nn graph apply (YAML changeset manifest)

```
nn graph apply <manifest.yaml> (--dry-run | --commit)
```

Creates notes and edges atomically from a single YAML manifest. Exactly one of `--dry-run` or `--commit` is required: `--dry-run` prints what would be created (`would create N note(s), add M edge(s)` plus per-item lines) without writing; `--commit` writes all notes and edges in one commit.

Manifest schema:

```yaml
notes:
  - key: a                    # optional local ref for edges (must be unique)
    title: My note            # required
    type: concept             # required; must be a valid note type
    content: |                # optional body
      Body text here
    tags: [foo, bar]          # optional
    applies_when: "..."       # optional
edges:
  - from: a                   # a manifest key, OR existing:<note-id>
    to: existing:20260817-1234
    type: refines             # link type
    annotation: "why this edge exists"   # required — errors if missing
```

Edge `from`/`to` references resolve two ways: a bare `key` declared in the `notes` block, or `existing:<note-id>` to link to a note already in the notebook. New notes are always created as `draft`. Validation errors: missing title, invalid type, duplicate `key`, missing edge annotation, or an unknown key reference (the error lists the known keys).

## nn status

```
nn status [--json] [--hubs N]
```

Reports: total notes, orphan count (with IDs/titles), draft count, broken links, draft link count, long notes, hub notes.

- **long notes**: notes whose body exceeds 2000 chars — candidates for splitting. Section omitted when none exist.
- **hub notes**: top N notes by link degree (inbound + outbound). Only shown when notebook has ≥10 notes. `--hubs N` overrides the default of 5.
- **draft links**: count of links with `status: draft` — links not yet human-endorsed.

`--json` output adds: `"draft_links": N`, `"long_notes": [{"id": "...", "title": "...", "body_len": N}]`, `"hub_notes": [{"id": "...", "title": "...", "degree": N}]`

## nn links

```
nn links <id> [--type TYPE] [--status draft|reviewed] [--json]
```

Lists outgoing links from a note with their annotations. `--type` filters by relationship type. `--status` filters by link status.

Link status: `draft` (default for new links — not yet human-endorsed), `reviewed` (human has verified the relationship). Legacy links without a status field are treated as `reviewed` for backward compatibility.

Text output: one entry per link — `targetID  title {status}\n  [type] annotation` (status and type shown when present)

`--json` output: `[{"id": "...", "title": "...", "annotation": "...", "type": "...", "status": "..."}]`

**Triage draft links:** `nn links <id> --status draft` shows only unreviewed links for a specific note.

## nn path

```
nn path <id-a> <id-b> [--json]
nn path <a> <b> --links <types> [--json]
```

Without `--links`, find the shortest undirected path between two notes via the link graph (BFS). Existing text and JSON behavior remain unchanged.

`--links TYPE,...` switches to directed traversal: follow only stored source→target links whose types are listed. Unknown or empty filters fail. The result contains only shortest-hop paths under that filter, capped at 3. Among equal-shortest alternatives, deterministic selection takes distinct first-hop nodes first, then distinct complete edge type-sequence values, then lexical full-path order.

Legacy untyped text remains each note on its own line with an `→` separator between hops. Typed text labels and renders every selected witness consistently, including each edge type and annotation.

Legacy untyped `--json` output remains exactly `[{"id": "...", "title": "..."}]`. Typed `--json` output is `{"witnesses":[{"nodes":[...],"edges":[{"from":"...","to":"...","type":"...","annotation":"..."}]}]}`. In any witness, edge `i` connects `nodes[i]` to `nodes[i+1]`; `nodes[1]` is the first hop after the origin.

## nn graph bridges

```
nn graph bridges [--search "<query>"] [--limit N] [--format text|json]
```

Rank notes that receive at least one incoming link and emit at least one outgoing link by `incoming-neighbor count × outgoing-neighbor count`. This is a load-bearing connector heuristic, not an articulation-point proof. `--format text|json` selects only the encoding; both formats carry the same evidence.

- Every result is one unified bridge record with `id`, `title`, structural `score`, `relevance_score`, and `witnesses`. The singular `witness` field is not emitted. Without search, JSON encodes `relevance_score` as `null` and text reports `relevance: n/a`.
- `--search QUERY` computes bridge scores over the complete graph, then keeps only bridge notes with a positive full-corpus search score. The query must be non-blank. Search results carry a numeric normalized `relevance_score` and rank by relevance descending, then structural `score` descending, then note ID ascending. Without search, results rank by structural score descending, then note ID ascending.
- `witnesses` contains at most 3 deterministic crossing examples. Construction gathers all incoming and outgoing edges, preserving endpoint ID/title, edge type, and annotation, and sorts each side by endpoint ID, type, then annotation before forming their Cartesian product. It keeps only the lexicographically earliest edge pair for each ephemeral full-graph region pair, so repeated edges within one pair cannot crowd out distinct region-pair examples; the cap is applied only after this deduplication. Selected pairs are ordered by incoming then outgoing region key: a clustered key is its representative ID, while an internal stable unclustered sentinel plus endpoint ID handles missing endpoints; edge-pair order is the final tie-breaker. These keys are ephemeral and are never emitted.
- Every crossing includes one `incoming` edge (`endpoint → bridge`), one `outgoing` edge (`bridge → endpoint`), and bounded `regions` context: `incoming` and `outgoing` each contain the endpoint's full-graph label-propagation region summary—`representative` (`id`, `title`) and full region `size`—or `null`/`unclustered` when unavailable. `same_region` reports whether both clustered endpoints received the same label. It can be `true`, and same-region pairs remain eligible: a crossing explains the connector heuristic, not proof that its endpoints occupy distinct territories. Representatives use the clusters command's deterministic highest-total-degree-then-ID rule; no durable region ID or ordinal is exposed. Text labels examples `crossing 1` through `crossing N` and presents both edges, both region summaries, and same-region status under each. Use this evidence to assess why the note is a plausible connector before acting on its returned ID.
- `--exclude ID` is repeatable, removes ranked candidates without changing full-graph bridge computation, and is applied before `--limit` (default: 10), so excluded results are replaced rather than leaving a short list.

Each returned `id` identifies the bridge note described by the record.

## nn clusters

```
nn clusters [--min N] [--singletons] [--json]
nn clusters --search "<query>" --json [--summary] [--min N] [--singletons]
```

Detect topological clusters of notes using label propagation. Each note starts with its own label and iteratively adopts the most common label among its linked neighbours.

- `--min N` omits clusters smaller than N notes (default: 2). Notes with no links are singletons and omitted by default.
- `--singletons` includes singleton clusters (notes with no links).
- `--search QUERY` preserves clustering over the complete graph, then returns only clusters containing positive-scoring search hits. It requires `--json`.
- `--summary` is search-only and omits full cluster `notes` while retaining `size`, `match_count`, `match_density`, `score`, `representative`, and ranked `matches`. Use `representative.id` as a subsequent `--focus`; rerun without `--summary` only when complete membership is needed.

Text output: one cluster per block — `cluster N (K notes):\n  ID  Title\n  ...`

Legacy `--json` output remains `[{"notes": [{"id": "...", "title": "..."}]}]`. Search-mode JSON ranks regions by the sum of their top three normalized match scores, preventing large regions from winning through an unbounded accumulation of weak matches. It includes `size`, total `match_count`, `match_density`, `score`, all ranked `matches`, `representative`, and—unless `--summary` is set—the full cluster `notes`. `match_density` is `match_count / size`, an explanatory signal, not a ranking input. The representative is the highest-total-degree member, with note ID as the stable tie-breaker.

## nn ast

```
nn ast <file> [--json] [--refs] [--root DIR]
```

Print a compact structural outline of a source file (imports, types, functions, constants). Uses gotreesitter (pure Go) to parse the file.

Supported languages: Go, Python, JavaScript, TypeScript, Rust, Java.

Text output always appends a `## Related notes` section with BM25-matched nn notes per symbol and the standard resolution instruction. Use `--json` for symbol-array-only output without the footer.

Text output:
```
file: src/backend/gitlocal.go  language: go
imports: fmt, os, path/filepath, ...
type Backend struct {
func (b *Backend) Write(n *note.Note) error {
...

## Related notes
- [[20260630…|gitlocal RMW lock pattern]] [likely relevant]
Resolve each related note before the next action — run `nn show <id>` to open...
```

`--json` output: `[{"kind": "...", "name": "...", "signature": "...", "line": N}]` (no footer)

`--refs` searches for name-match references to every symbol in the outline across the codebase rooted at `--root` (default: `.`). Emits one `references to "X"` section per symbol. Name-match only — not symbol-resolved, may include false positives. Prefer `nn trace` when you need call-graph traversal rather than text matches.

```
nn ast src/backend/gitlocal.go --refs --root ./
```

## nn tee

```
nn tee
```

Reads stdin, writes it unchanged to stdout, and prints BM25-matched related notes to stderr. Pipeline-transparent — downstream commands receive stdin byte-for-byte; note output never appears on stdout.

**Use `nn tee` when:**
- Fetching a web page or API response: `curl <url> | nn tee | jq .`
- Listing processes or system state: `ps aux | nn tee`
- Piping any content where you want related notes surfaced without breaking the pipeline

Large inputs are truncated to 4096 bytes for the BM25 search window; stdout receives the full content.

stderr output format:
```
## Related notes
- [[ID|title]] [likely relevant]
Resolve each related note before the next action...
```

**LLM usage note:** prefer `nn tee` over manually running `nn list --search` after fetching content — `nn tee` surfaces notes in one step and keeps the pipeline intact for downstream processing.

## nn grep

```
nn grep <pattern> [path...] [-i|--ignore-case] [--context N] [--notes-per-match K] [--max-matches N] [--trace] [--context-report]
```

Search files for a regular-expression pattern and annotate each retained match with related nn notes. `-i`/`--ignore-case` makes matching case-insensitive. `--context N` controls the source window around each match; `--max-matches N` limits the retained windows.

`--context-report` appends an exact source-overlap summary for the retained windows: context-block count, gross context-line occurrences, unique `(path, line)` identities, and overlap occurrences (`gross - unique`). Without the flag, normal grep output is unchanged. This first report measures source windows only; it does not deduplicate related notes or track content previously supplied in a session.

## nn trace

```
nn trace <root-dir> --symbol <name> [--symbol <name> ...] [--depth N] [--json] [--show-unresolved] [--root <dir>]
```

Syntax-aware call graph from one or more entry-point symbols. Uses gotreesitter to index all definitions in `<root-dir>` (or `--root <dir>` if given), then DFS-traces calls from each `--symbol` up to `--depth` hops (default 3).

**Prefer `nn trace` over `nn grep`** when you want to understand how a symbol is called or what it calls across files — trace follows actual call edges rather than text matches.

Each resolved node is annotated with related nn notes via BM25 (same mechanism as `nn grep`). After the tree, a `## Related notes` section lists all surfaced notes with the standard resolution instruction.

**Ambiguous receiver annotation:** when a method call `obj.method()` resolves to N > 1 definitions (because the receiver type is unknown), each expanded candidate is marked `[receiver: obj, N candidates — type-unqualified]`. This signals that those branches may be false positives — the call is real but the targets are uncertain.

Human-readable output (default):
```
AddLink (method) [internal/backend/gitlocal/gitlocal.go:288]
  note: [[20260630…|gitlocal RMW lock pattern]]
  → acquireLock (function) [internal/backend/gitlocal/gitlock.go:26]
  → Read (method) [internal/backend/gitlocal/gitlocal.go:109] [receiver: g, 3 candidates — type-unqualified]
```

`--json` output: `{"nodes": [...], "edges": [...]}` — each node has `nn_notes`, `ambiguous_receiver`, and `receiver` fields.

`--show-unresolved`: include stdlib/external leaves (default off; always in JSON with `resolved: false`). Go builtins (`append`, `make`, `len`, ...) are always filtered out.

`--root <dir>`: index this directory instead of `<root-dir>`, so calls to definitions in **sibling packages** resolve rather than showing as unresolved. Language-agnostic (indexes every grammar-detected file under the root, not just Go). Pass the repo/project root to follow logic across packages; stdlib/third-party calls stay unresolved because they are not in the project.

```
nn trace ./internal/backend/gitlocal --symbol AddLink --depth 3
nn trace ./cmd/nn/cmd --symbol newGrepCmd --json
nn trace ./cmd/nn/cmd --symbol newGrepCmd --root .   # resolve cross-package calls
```

## nn update

```
nn update <id-or-title> [--title TEXT] [--tags TEXT] [--tags-add TAG] [--tags-remove TAG]
         [--content TEXT] [--stdin] [--append TEXT] [--replace-section HEADING]
         [--type TYPE] [--status STATUS] [--no-edit]
         [--expires YYYY-MM-DD] [--expires-when TEXT] [--clear-expires] [--clear-expires-when]
         [--link-to ID --annotation TEXT]  # repeatable; --annotation must match --link-to count
         [--check]
```

Accepts a note ID **or a title substring** — if the substring matches exactly one note it is used; multiple matches returns an error. At least one change flag is required. `--content`/`--stdin`/`--append` are mutually exclusive.

| Flag | Effect |
|---|---|
| `--title TEXT` | Replace note title |
| `--tags TEXT` | Replace all tags (comma-separated) |
| `--tags-add TAG` | Add a tag without touching others (repeatable) |
| `--tags-remove TAG` | Remove a tag without touching others (repeatable) |
| `--content TEXT` | Replace note body entirely |
| `--stdin` | Read replacement body from stdin (heredoc-safe, no shell escaping) |
| `--replace-section HEADING` | Replace only the named `## Heading` section; requires `--content` or `--stdin`; errors if heading not found |
| `--append TEXT` | Append text to note body (double-newline separator) |
| `--status STATUS` | Set note status: `draft`, `reviewed`, or `permanent` |
| `--expires YYYY-MM-DD` | Set expiration date; note appears in `nn list --expired` after this date |
| `--expires-when TEXT` | Set conditional expiration (plain text condition, e.g. "when the PR is merged") |
| `--clear-expires` | Remove the expiration date from the note |
| `--clear-expires-when` | Remove the expires_when condition from the note |
| `--no-edit` | Skip `$EDITOR` (always use in non-TTY/LLM context) |
| `--since RFC3339` | **Required.** Reject update if note was modified after this timestamp; read `modified:` from `nn show` output. Omitting returns an error. |
| `--link-to ID` | Add a link to an existing note ID (repeatable) |
| `--annotation TEXT` | Link annotation paired with `--link-to` (repeatable, must match count) |
| `--check` | Opt-in; runs representation graph validation after update. No-op if note has no `representation` field. |

**Preferred LLM patterns:**

```bash
# Update by title substring (no ID lookup needed)
nn update "my note title" --content "new body" --no-edit

# Multiline body without shell escaping
nn update <id-or-title> --stdin --no-edit <<'EOF'
Body with `backticks`, "quotes", and $pecial chars — no escaping needed.
EOF

# Replace a single section, preserve the rest
nn update <id-or-title> --replace-section "Why" --content "New explanation." --no-edit

# Additive tag operations
nn update <id-or-title> --tags-add "zettelkasten" --tags-remove "inbox" --no-edit

# Demote or reset status
nn update <id-or-title> --status draft --no-edit
```

## nn remind

Create a temporary reminder note that surfaces in `nn show --global` until its expiration date.

```
nn remind "TEXT" [--for N] [--expires YYYY-MM-DD]
nn remind --find FRAGMENT
nn remind "new body" --update ID
```

- Creates an `observation` note tagged `reminder` with `permanent` status
- Title = first 60 characters of TEXT; body = full TEXT
- Default expiry: today + 1 day (use `--for N` for N days, or `--expires DATE` for a specific date)
- Appears in the `## Reminders` block of `nn show --global` until expired
- Expired reminders are silently omitted from `--global` output
- `--find FRAGMENT` — search reminder titles by substring; prints matching ID; errors if ambiguous or zero matches
- `--update ID` — replace body of existing reminder in place; preserves expiry; no new note created

```bash
nn remind "Don't merge the auth PR until legal signs off"        # expires tomorrow
nn remind "Check in with mobile team" --for 3                    # expires in 3 days
nn remind "Hold off on deploys" --expires 2026-06-01             # expires on date
nn remind --find "auth PR"                                        # find reminder ID by title fragment
nn remind "Updated context" --update <id>                         # update existing reminder body
```

## nn promote

```
nn promote <id-or-title> --to reviewed|permanent
```

Status progression: `draft` → `reviewed` → `permanent`. Accepts title substring. For direct status assignment (including demotion), prefer `nn update --status`.

## nn log

Show the git diff history for a single note — all commits that touched its file, with full patch output.

```
nn log <id-or-title> [--since DATE]
```

| Flag | Effect |
|---|---|
| `--since DATE` | Limit history to commits after this date (e.g. `2025-01-01`); passed directly to `git log --since=` |

Output is raw `git log -p --follow` for the note's filename. Use to audit what changed and when.

```bash
nn log <id>                         # full diff history
nn log <id> --since 2026-01-01      # only changes since Jan 2026
```

## nn delete

```
nn delete <id-or-title> --confirm
nn delete --from-stdin --confirm
```

`--confirm` is required. Warns if other notes link to the deleted note.

`--from-stdin` reads note IDs line-by-line from stdin and deletes each. Compose with `nn list` for batch deletion:

```
nn list --no-inbound --status draft --json | jq -r '.[].id' | nn delete --from-stdin --confirm
```

## nn todo

Manage open checkboxes (todo items) across notes.

```
nn todo list [--all] [--waiting] [--context <name>]
nn todo done <id> <pattern> [<pattern>...] [--resolution "why"]
nn todo reopen <id> <pattern> [<pattern>...]
```

**Todo item format** — write checkboxes as standard Markdown:
```
- [ ] open item
- [x] done item
```

**Inline metadata** (on the same `- [ ]` line):

| Tag | Syntax | Effect |
|---|---|---|
| Waiting | `[waiting: reason]` | Hidden from default list; shown with `--waiting` |
| Context | `@context` | Filterable with `--context <name>` |

Examples:
```
- [ ] [waiting: Josh to review] submit the PR
- [ ] @phone call the vendor
- [ ] @computer write the report
```

**nn todo list flags:**
- Default: shows unblocked items; excludes `[waiting: ...]` items and notes blocked by an incomplete `requires` target
- `--all`: show all open items including blocked notes
- `--waiting`: show only `[waiting: reason]` items, with the reason visible in output
- `--context <name>`: filter to items tagged `@<name>` (case-insensitive)

**nn todo done / reopen:**
- `nn todo done <id> <pattern>` — marks the first `- [ ]` line containing `<pattern>` as `- [x]`
- `nn todo reopen <id> <pattern>` — marks the first `- [x]` line containing `<pattern>` as `- [ ]`
- Pattern match is case-insensitive; errors if no match found
- **Multiple patterns** flip in a single write: `nn todo done <id> "pat a" "pat b"`. Prefer this over parallel `nn todo done` calls on the same note — concurrent single-pattern calls race on the note's version and the later write fails. Two matching modes: if a **single open line contains every pattern**, that one line is flipped (conjunctive AND — use several patterns to pinpoint one item, e.g. `"atomicity" "targeted test"`); otherwise **each pattern flips its own distinct line** (batch flip of several items). The conjunctive single-line match takes precedence. Matching is all-or-nothing: if any pattern matches no checkbox, the note is left unchanged.
- `--resolution "why"` on `done` appends commentary to each flipped line (e.g. `- [x] migrate schema — done in PR #42`), recording why it was completed.

**nn todo set** — add, replace, or remove inline metadata tags on a matching open checkbox:
```
nn todo set <id> <pattern> --waiting "reason"     # add/replace [waiting: reason]
nn todo set <id> <pattern> --clear-waiting         # remove [waiting: ...]
nn todo set <id> <pattern> --context phone         # add/replace @phone
nn todo set <id> <pattern> --clear-context         # remove @context
```
- Matches the first open `- [ ]` line containing `<pattern>` (case-insensitive)
- Flags can be combined in a single call (e.g. `--waiting "Josh" --context phone`)

## nn capture

Quickly capture raw material (articles, quotes, observations) as a draft note.

```
nn capture --title TEXT [--content TEXT] [--type TYPE] [--tags TEXT]
```

- Default type: `observation`. Override with `--type concept|argument|...`
- Status is always `draft` — the note is intended for LLM refinement
- Prints the created note ID to stdout

Typical flow:
```
nn capture --title "..." --content "..." # capture → get ID
nn update <id> --content "..." --no-edit  # LLM refines
nn suggest-links <id>                     # discover links
nn suggest-tags <id>                      # discover tags
nn tags                                   # enumerate tag vocabulary
```

## nn suggest-links

Format context for LLM-driven link suggestion. Does not call an LLM — emits a structured block for the LLM session to reason over.

```
nn suggest-links <id> [--limit N] [--format json]
```

Output contains:
- `## Focal note` — full body of the focal note
- `## Candidate notes (N total, M excluded — no term overlap)` — BM25-ranked candidates, each with tags and a 200-char summary
- Notes already linked to the focal note are marked `(already linked: <type>)`
- Zero-BM25-score notes are excluded; the count is reported in the header

Default limit: 20. The LLM reads the output and calls `nn link` or `nn bulk-link` for accepted suggestions.

## nn suggest-tags

Returns tag suggestions for a note based on BM25-similar notes that share tags the target lacks.

```
nn suggest-tags <id> [--json] [--min-notes N]
```

Only tags appearing in ≥ `--min-notes` similar notes (default: 2) are returned. The LLM applies accepted tags via `nn update <id> --tags "..."`.

## nn tags

Lists all tags in the notebook with note counts.

```
nn tags [--json]
```

JSON output: `[{"tag": "...", "count": N, "notes": ["id1", ...]}]`. Use before tagging a new note to orient against the existing vocabulary.

## nn review

Notebook health report formatted for LLM-driven analysis.

```
nn review [--format json]
```

Sections:
- **Growth**: total notes, by type, created in last 7/30 days
- **Connectivity**: total links, avg links per note, orphan count + IDs, dead-end count + IDs
- **Structural gaps**: draft note count + IDs, long note count + IDs (body > 2000 bytes)
- **Aging notes**: notes not modified recently, split into two buckets using the same thresholds as `nn show` freshness — `aging (3–14 days)` and `stale (>14 days)`, sorted oldest-first within each bucket
- **Friction candidates**: unreviewed observation notes tagged `friction-candidate`
- **Protocol telemetry**: protocol session-presence counts from `protocol-presence.log`
- **Note access**: note view counts from `access.log`

A "dead-end" note has outbound links but no inbound links — it contributes to others but nothing points back to it.

**Long notes** (body exceeds 2000 bytes) are listed under Structural gaps — candidates for splitting into atomic notes.

**Aging notes** surface content that may be stale. Notes in the `stale (>14 days)` bucket should be verified before relying on them; notes in `aging (3–14 days)` may need a recheck. This mirrors the `freshness:` line injected by `nn show`.

`--format json` keys: `growth`, `connectivity`, `drafts`, `long_notes`, `aging_notes`, `stale_notes`, `friction_candidates`, `protocol_telemetry`, `note_access`

## nn gap

Format topic neighborhood context for LLM gap analysis.

```
nn gap <topic> [--limit N] [--depth N] [--format json]
```

Loads notes matching `<topic>` via BM25 search, then expands to their direct neighbors (depth 1 by default) via forward links and backlinks. Output:

- `## Topic notes (N matching "topic")` — ranked by BM25 score
- `## Neighborhood (N notes, depth D)` — linked neighbors not in topic set

The LLM receiving this output identifies what is thoroughly covered, what is shallow, what is absent, and what questions the notes raise but do not answer.

Default limit: 20. Default depth: 1.

## nn index

Format topic cluster context for LLM-driven Map of Content creation.

```
nn index <topic> [--limit N] [--format json]
```

Loads notes matching `<topic>` via BM25, then groups them into clusters using link-based label propagation (scoped to the topic subset). Output:

- `## Topic: "topic" (N notes)` — all matching notes with summaries
- `## Clusters (N)` — grouped by link connectivity

The LLM names the clusters, identifies tensions and gaps, and creates an index note via `nn new`.

Default limit: 30. `--format json` keys: `topic_notes`, `clusters`

## nn random

Return a randomly selected note. Optionally filtered.

```
nn random [--tag TEXT] [--type TYPE] [--status STATUS] [--json] [--depth N]
          [--max-backlinks N]
```

Returns one note at random from the notebook. All filters from `nn list` are supported.
Use for deliberate serendipity — re-encounter a forgotten note and consider whether it
connects to current work.

`--depth N` traverses outgoing links from the selected note up to N hops, printing all reachable notes as a concatenated Markdown document. Use to load a random subgraph as context.

`--max-backlinks N` filters candidates to notes with at most N inbound links. Use to surface underlinked notes for integration review — notes that exist but haven't been woven into the graph yet.

```
nn random                         # any note
nn random --status permanent      # a random permanent note
nn random --tag philosophy --json
nn random --depth 2               # random note + 2 hops of outgoing links
nn random --max-backlinks 0       # a note nothing links to yet
nn random --max-backlinks 2       # a note with few inbound links
```

## nn shuf

Sample random units from files (or stdin) and show BM25-matched notes beneath each sample.

```
nn shuf [<path>...] [--count N] [--unit lines|paragraphs|symbols]
```

With no paths, reads from stdin. Piped stdin is not MIME-filtered, but its sampled units use the same size limit. Multiple paths are pooled and sampled uniformly. Files classified as non-text are skipped before sampling; `nn shuf` reports an aggregate count of skipped binary files on stderr. Sample units larger than 65536 bytes are also skipped and reported as an aggregate stderr count.

`--count N` (default 5) — number of units to sample.

`--unit` options:
- `lines` — single lines drawn uniformly at random
- `paragraphs` — blank-line-delimited blocks (default; good for prose and Markdown)
- `symbols` — treesitter-parsed functions/types/methods; falls back to paragraphs for unsupported file types

Each sampled unit is printed after a `---` separator, followed by a `## Related notes` block with up to 5 BM25-matched notes.

**When to use `nn shuf` vs similar commands:**

| Want | Use |
|------|-----|
| Statistical coverage of a corpus — serendipitous note matches with no specific query | `nn shuf` |
| Targeted search for a pattern or keyword | `nn grep` |
| Pass content through a pipeline and surface notes without consuming stdin | `nn tee` |
| Random note from the Zettelkasten (not from files) | `nn random` |
| Structural outline of a single source file | `nn ast` |

`nn shuf` is the **inverse of grep**: instead of "find content matching this query," it asks "given a random slice of this content, what notes are relevant?" Use it for statistical sampling across a codebase, a set of documents, or any corpus you want to cross-reference against your notes.

```
nn shuf notes/*.md --count 3             # 3 random paragraphs from notes
nn shuf src/cmd/*.go --unit symbols      # 5 random functions, treesitter-parsed
nn shuf --unit lines < corpus.txt        # random lines from stdin
cat *.go | nn shuf --unit paragraphs     # pipe multiple files
nn shuf . --count 10 --unit symbols      # recursively sample symbols from a directory
```

## nn fetch

```
nn fetch <url> [--capture]
```

Fetches a URL via HTTP GET, strips HTML to plaintext, and prints the result to stdout with a `## Related notes` section ranked by BM25 against your notebook.

**Flags:**
- `--capture` — create a draft observation note whose body is the fetched plaintext; prints the note ID to stderr

**Use `nn fetch` when:**
- You want to pull in a web page and immediately see which notebook notes relate to it
- You want to capture a URL's content as a note (`--capture`)
- You want to pipe fetched text into further processing: `nn fetch <url> | nn tee`

**LLM usage note:** prefer `nn fetch` over `curl <url> | nn tee` when you want the HTML stripped automatically — `nn fetch` removes tags, script/style blocks, and entities before running BM25.

## nn search-web

```
nn search-web <query> [--results N]
```

Best-effort web search via DuckDuckGo. Fetches the top N result pages, strips HTML, and prints a `## Result N:` header + 500-character preview + `## Related notes` from your notebook for each result.

**Flags:**
- `--results N` — number of results to fetch and display (default 3)

**Use `nn search-web` when:**
- You want to surface external context on a topic and compare it against your existing notes
- You want a quick literature scan without leaving the terminal

**LLM usage note:** DDG occasionally wraps results in JS-challenge pages; results quality varies. Use `nn fetch <url>` directly when you have a specific URL.

## nn ask

```
nn ask [--surface canvas|document|web] [--instructions "..."] [--mermaid "<diagram>"] [--document <file|folder>]
```

Ask a human for feedback via a chosen **surface**, block until they submit, then print the path to a thin result envelope. The primitive is the job — *"get a human to close a knowledge gap"* — and the surface is a routing decision keyed on the shape of the answer (ADR-0020). `nn ask` runs **without a configured notebook** (it is note-agnostic at the boundary).

**Result contract.** Every surface writes to a session directory (`~/.config/nn/feedback/<id>/`) and produces `result.json` — a thin envelope naming native artifacts by path:

```json
{ "id": "...", "surface": "canvas", "status": "submitted",
  "artifacts": [ { "format": "excalidraw", "path": "result.excalidraw" } ] }
```

Surface-specific shape lives *inside* the referenced files. **You read the artifact and decide** what, if anything, it becomes (a note, an update, a `graph apply`, or nothing) — `nn ask` never files the result automatically.

Session directories are **ephemeral scratch**: each `nn ask` run reclaims session directories older than 7 days. If you want to keep a result, persist it (into a note, or copy the artifact elsewhere) — do not rely on the session directory surviving.

**Surfaces:**
- `--surface canvas` (default, *hosted*) — an embedded Excalidraw diagram editor. `--mermaid "<diagram>"` seeds the canvas with an editable diagram (converted from Mermaid); the human edits it and clicks Done. Writes `result.excalidraw` (scene) + `result.png` (image).
- `--surface document` (*delegated*) — hands the document to the `plannotator` peer for text/markdown annotation. Annotates `--document <file|folder|url>` when given (a `https://` URL is passed straight through — plannotator fetches it), otherwise the `--instructions` text. Writes `result.plannotator.json`, a `{ "decision": "approved|annotated|dismissed", "feedback": "..." }` object you read to get the human's annotations.

**Flags:**
- `--surface` — which surface to route to (default `canvas`)
- `--instructions "..."` — the prompt shown to the human on the surface (canvas), or the content to annotate when no `--document` is given (document)
- `--mermaid "<diagram>"` — canvas only: seed the canvas from a Mermaid diagram
- `--document <file|folder>` — document only: the file or folder to annotate

**Use `nn ask` when:**
- You reach a point where only a human can supply the missing knowledge — a sketch, a judgment on a diagram, or annotations on prose
- You want that response back as structured data at a known path, on your terms

**LLM usage note:** `nn ask` blocks until the human submits — do not run it where no human is present. After it returns, read the named artifact(s) and decide what enters the notebook.

## nn install-skills

```
nn install-skills [--dest DIR] [--list]
```

Copies skill directories into `~/.claude/skills/` (or `--dest`).

## Configuration

`~/.config/nn/config.toml`:

```toml
[notebooks]
default = "personal"

[notebooks.personal]
path = "~/notes"
backend = "gitlocal"
```

Environment overrides:
- `NN_NOTEBOOK` — select a named notebook (overrides `default`)
- `NN_CONFIG_DIR` — override config directory (useful for testing)

## Note schema

```yaml
---
id: 20260411120045-3821
title: "The Atomicity Principle"
type: concept
status: draft
tags: [zettelkasten, methodology]
created: 2026-04-11T12:00:45Z
modified: 2026-04-11T12:05:00Z
---

Body text.

## Links

- [[20260411090000-1234]] [extends] {draft} — provides the foundational philosophy this principle implements
```

Link format in plain-text `show` output: `- [[target-id|Target Title]] [type] {status} — annotation` (titles resolved at render time). In the raw file on disk, the format remains `- [[target-id]] [type] {status} — annotation`.
- `[type]` optional: `refines`, `contradicts`, `source-of`, `extends`, `supports`, `questions`, `governs`, `requires`
- `{status}` optional: `draft` (default for new links), `reviewed` (human-endorsed). Absent = `reviewed` (legacy compat).

## LLM usage patterns

**Create and link in sequence:**
```
nn new --title "Concept A" --type concept --content "..." --no-edit
# note ID from output: 20260411120045-0001
nn list --json | jq '.[].id'   # find related note IDs
nn link 20260411120045-0001 <related-id> --annotation "extends this concept" --type extends
```

**Load all global protocols at session start:**
```
nn show --global
```

**Find orphans before a review session:**
```
nn list --orphan --json
```

**Export graph for visualisation:**
```
nn graph --json > graph.json
```

**Discover related notes (no known query):**
```
nn list --similar <id> --limit 10
```

**Load a topic cluster as LLM context:**
```
nn show <id> --depth 2
```

**Explore a note's immediate graph neighborhood (single round-trip):**
```
nn show <id>     # shows body + resolved outgoing links + backlinks in one call
```

**Serendipitous re-encounter:**
```
nn random --status permanent
```
