# nn-eval-capture

Runs the capture-discipline behavioral eval suite.

## TRIGGER

Use when asked to eval, test, or check LLM compliance with the capture-discipline protocol.

## Instructions

**Step 1** — Load scenario data as JSON:

```bash
bash .claude/skills/nn-eval-capture/eval-capture.sh --json > /tmp/eval_scenarios.json
```

To inspect a single scenario in human-readable form: `bash .claude/skills/nn-eval-capture/eval-capture.sh 3`

**Step 2** — Parse the JSON and spawn one subagent per scenario (all in parallel, `run_in_background: true`, `subagent_type: general-purpose`). Use each item's `prompt` field as the subagent prompt. Do not modify the prompts. **Do not ask the script to spawn subagents** — spawning is the orchestrator's job; the script only generates prompts.

```bash
# Read scenario count and fields:
python3 -c "import json; d=json.load(open('/tmp/eval_scenarios.json')); [print(x['num'], x['gate'][:60]) for x in d]"
```

**Step 3** — When all subagents return, check each response against its `gate` field:
- **PASS**: the gate string appears in the response at the required position
- **FAIL**: gate string absent, wrong position, or wrong form

**Step 4** — Report results as a table: scenario number, PASS/FAIL, verbatim excerpt from the response that determined the verdict.

If 3 or more FAILs: run `/nn-session-debrief` to capture findings and consider updating the protocol body in `cmd/nn/cmd/show.go`.
