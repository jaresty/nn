# ADR-0016: Commit Queue Drain Pattern for Cross-Process Git Serialization

## Status

Proposed

## Context

Each `nn` CLI invocation is a separate OS process. When an LLM agent issues rapid sequential operations (e.g. bulk capture, link, update), multiple processes race to `git add + git commit` against the same repository. The existing mitigation is index.lock retry with backoff, but this serializes at the git layer and adds latency proportional to lock contention depth.

The root problem: git's object store and index are not designed for concurrent writers. We want to serialize commits without making every `nn` operation wait for the previous one's full git round-trip.

## Decision

Introduce a **commit queue drain pattern** implemented entirely in the existing `nn` binary — no separate daemon binary required.

### Queue directory

Each write operation (Write, Delete, Link, Update) enqueues a **commit item** before attempting git operations:

- Queue dir: `~/.config/nn/commit-queue/` (created on first use)
- Each item: a JSON file written via `os.CreateTemp` then `os.Rename` into the queue dir (atomic appearance)
- Item filename: `<timestamp-nanos>-<pid>.json` (sortable, unique)
- Item content: `{ "op": "...", "message": "...", "files": [...] }` — everything needed to run `git add <files> && git commit -m <message>`

### Drain lock

- Lock file: `~/.config/nn/commit-queue.lock`
- Format: `<pid>\n` written atomically via `O_CREAT|O_EXCL`
- The process that wins `O_EXCL` becomes the **drainer** for this cycle

### Protocol per `nn` operation

```
1. Write files to notebook dir (unchanged)
2. Write commit item to queue dir (atomic rename)
3. Try O_CREAT|O_EXCL on commit-queue.lock
   a. EEXIST → check pid in lock file is alive (kill -0)
              → alive: exit (drainer running, item queued)
              → dead:  delete stale lock, retry step 3
   b. Success → become drainer (proceed to step 4)
4. Drain loop:
   a. Read queue dir, sort by filename (arrival order)
   b. For each item: git add <files> && git commit -m <message>, delete item
   c. If queue empty: delete lock file, exit drain loop
   d. Else: loop back to (a) — new items may have arrived
```

### `nn drain` command

A new subcommand for manual intervention:

```
nn drain              # force-drain the queue, stealing a stale lock if needed
nn drain --status     # print queue depth and lock holder pid (no drain)
```

`nn drain` bypasses the O_EXCL election — it always attempts to drain, stealing a dead lock. Intended for recovery after a drainer crash and for scripted pre-push sync.

### Stale lock recovery

If `O_EXCL` returns `EEXIST` but `kill -0 <pid>` returns `ESRCH` (no such process), the lock is stale. The detecting process deletes the lock file and retries `O_EXCL`. This is safe because: if two processes simultaneously detect a stale lock and both delete+retry, only one wins `O_EXCL` — the other gets `EEXIST` again and re-checks, now seeing a live pid.

### Ordering guarantee

Items are committed in filename sort order (timestamp-nanos prefix). Because `os.Rename` is atomic and the drainer reads the queue dir after acquiring the lock, any item enqueued before the drainer's `ReadDir` call is guaranteed to be committed in this drain cycle. Items enqueued after are either picked up in the next drain loop iteration (if the drainer is still running) or by the next drainer.

### Coalescing

The drainer MAY coalesce consecutive items into a single commit when their `op` fields are the same and their `files` sets are disjoint — this reduces git overhead under burst load. Coalescing is optional and off by default; it would break the "one operation = one semantic commit" invariant if enabled, so it requires an explicit config flag (`commit_queue_coalesce = true`).

## Consequences

- **Preserves**: one operation = one semantic commit (coalescing off by default)
- **Eliminates**: index.lock contention under normal burst load
- **Adds**: ~1 `os.Rename` + `O_EXCL` syscall per operation (negligible)
- **Risk**: queue dir accumulates items on repeated drainer crashes — `nn drain` is the recovery path
- **Scope**: change is entirely within `internal/backend/gitlocal/` and a new `cmd/nn/cmd/drain.go`
