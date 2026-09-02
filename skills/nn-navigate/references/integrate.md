---
name: integrate
applies_when: "When Navigation proposes, reviews, applies, declines, or returns from a bounded mutation to notebook truth."
---

# Reference: Integrate

## Purpose and ownership

**✚ Integrate…** is the agent-owned Navigation gateway for a bounded edit to notebook truth discovered while walking visited evidence. It is not an `nn integrate` command and does not replace the existing mutation commands or their safeguards. Integrate owns proposal composition, human review, truthful dispatch, and Navigation return; `nn-guide` owns command syntax, and any specialized workflow retains its own authority.

Integrate may create, update, retitle, retag, retype, change status, or delete notes; add, remove, retype, or revise typed relationships; apply a supported mixed change; or conclude that no mutation is justified. “Retype” for notes means changing note type through the existing update workflow; relationship retyping uses the existing edge workflow. Configuration, source-code edits, and conversation-scoped Navigation state are outside Integrate.

## Media-run integration

A natural request such as “integrate this media run RUN_ID” invokes Integrate directly and is not an `nn integrate` command. Before proposing any notebook change:

1. Run `nn media context --run RUN_ID --page 1` and retrieve every disclosed next page; context retrieval must not reprocess media.
2. Validate run, source, bundle, and manifest provenance plus qualified coverage and truncation disclosures.
3. Use the available image-reading capability to load each actual path from every typed `image_attachments` record; a path string alone is not visual evidence. Treat image timestamps as sampled instants, never intervening coverage.
4. Treat transcript chunks and images as evidence with their distinct limits. Synthesized document transcript boundaries are not native sentence timings.
5. Search and read relevant existing notebook truth before deciding whether any creation, update, or relationship is justified.
6. Produce the existing non-mutating proposal with sourced claims separated from interpretation, uncertainty, affected notes/edges, complete before/after intent, and expected operation/commit count. Write only after explicit human approval through existing mutation workflows.

## Contextual promotion

Promote `✚ Integrate — <concrete change>` only when the current complete visited evidence supports a **concrete bounded notebook mutation supported by visited evidence**. The proposal must name the exact receiving note, new note, relationship, correction, status change, or deletion under consideration and explain why it matters to the retained goal. Generic cleanup, lexical similarity, graph proximity, or “the notebook could be better” is insufficient.

Integrate occupies the existing **contextual shortcut slot**. It is **never a permanent menu row** and never expands `All navigation actions…`; the stable `Lenses…`, `All navigation actions…`, and final `■ Arrive` rows remain unchanged. When no concrete mutation is justified, do not promote Integrate merely to advertise availability. Natural requests such as “integrate this,” “capture what we learned,” “link these,” “update that note,” or “delete the obsolete note” invoke it directly when a retained frame exists.

Use the effect marker and adjacent legend exactly as follows:

> **✚ notebook truth changes; focus and history retained; Orient refreshes**

## Non-mutating proposal

Before any write, compose a proposal that is itself non-mutating. It may contain one or more candidate edit records, each classified as one of:

- create note;
- update note body or section;
- retitle, retag, retype, or change status;
- delete note;
- add a canonical typed relationship;
- remove a relationship;
- retype a relationship or revise its annotation;
- supported mixed change; or
- no-op / keep conversation-only.

The review surface must show:

1. purpose and retained-goal consequence;
2. evidence IDs and exact source boundaries;
3. sourced claims versus generated interpretation;
4. **affected IDs and exact before/after intent** for every existing artifact;
5. complete proposed title, type, body boundary, metadata, status, or section change for each note edit;
6. stored source, canonical type, stored target, and non-empty annotation for every relationship edit;
7. authority limits, uncertainty, and claims the change will not make;
8. execution workflow, expected semantic operation/commit count, and any unsupported atomic combination; and
9. for deletion, affected **backlinks and loss consequences**, including relationships that would disappear or notes that rely on the target.

For a new synthesis, explain why it is durable, non-duplicative, and not better represented by an existing receiving note. For an update, quote or summarize the current relevant body and show the replacement or appended meaning. For a deletion, distinguish obsolete, superseded, erroneous, and merely inconvenient material; deletion is not a cleanup default.

Offer bounded human choices appropriate to the proposal: apply all, apply selected edits, revise note edits, revise relationship edits, choose an existing receiver, keep conversation-only, or cancel. Every edit requires **explicit human approval** before application. Approval of creation or content does not imply status promotion, edge review, formal-representation classification, or acceptance of generated claims unless those are separately visible and approved.

## Evidence and edge discipline

Read exact current bodies and stored relationships for every affected existing note before proposing a mutation. Search, similarity, clustering, bridge scores, route ranking, generated geometry, proximity, and analogy are retrieval or explanatory signals; none establishes a notebook relationship by itself.

Every added or retyped relationship must use a canonical non-empty type and annotation and be **supported by both endpoint bodies** under that type’s definition. State stored direction explicitly. If one endpoint does not support the proposed claim, offer a note update for separate approval or decline the edge; never use the planned update as if it were already notebook truth.

Find gaps and Find an analog may surface candidates, but comparison-only analogies remain non-links. Human Graph/Canvas/Document annotations are evidence to interpret, not automatic note or edge writes. Generated groupings, containment, arrows, and layouts remain non-stored until Integrate independently establishes and receives approval for the corresponding mutation.

## Truthful execution boundary

After approval, dispatch only through valid existing `nn` workflows using their current command contracts:

- atomic new-note plus edge creation: `nn graph apply` when its manifest supports the complete approved change;
- note creation: `nn new`;
- body, metadata, type, tags, status, expiry, or supported link additions: `nn update` with a fresh `--since` value;
- note deletion: `nn delete --confirm` after showing backlink/loss consequences;
- edge addition: `nn link` with canonical type and annotation;
- edge removal: `nn unlink` with the intended endpoints/type;
- legacy edge retyping: `nn link set-type` with ambiguity protection;
- todos and other specialized note state: their owning `nn` workflow.

Prefer a workflow that performs the entire approved semantic change as one operation and one Git commit. Integrate must **never represent multiple semantic operations or commits as one atomic application**. If the approved heterogeneous change is not supported atomically by an existing workflow, do not silently sequence commands. Report the boundary and require **scope reduction or a separately implemented atomic capability**. The human may approve a narrowed single operation; approval of the larger proposal is not permission to split it into hidden commits.

Integrate does not bypass specialized safeguards. Dispatch and satisfy the owners for **deletion, status, and formal-representation workflows**, optimistic concurrency, typed-edge validation, confirmation, representation checks, or any other applicable mutation guard. It never edits notebook files directly to evade a CLI contract and never automatically promotes a new note or relationship.

## Navigation lifecycle

Proposal, revision, decline, cancellation, and failed validation retain the exact focus, complete frame, Back/Forward stacks, bookmarks, interaction mode, current menu, and menu stack. They perform no implicit Recenter and no notebook mutation. A promoted Integrate returns semantically to Quick actions; a direct conversational invocation retains its prior semantic menu position until application determines the refresh path.

After a successful write, retain focus and graph history, but **invalidate the cached frame** because bodies, links, degrees, zones, or status may have changed. Fetch `movement`, `state`, and `presentation`, then retrieve topology first and complete snapshot-bound bodies for the retained focus. Then **rerun Orient before presentation** and reset the semantic menu to Quick actions while preserving interaction mode. Guided renders the refreshed Focus + Map + Moves and Quick actions; Advanced keeps help closed. If the retained focus was deleted by explicit approval, its frame is no longer reproducible: report that exact consequence and require a new landing rather than inventing a focus.

Report the applied semantic operation, affected IDs, actual commit/result, and any approved items not applied. Never claim success from a partial or failed workflow. A newly created or newly linked note appears only where the refreshed stored graph actually places it.
