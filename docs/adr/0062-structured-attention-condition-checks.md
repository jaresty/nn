# ADR-0062: Structured checks for the bundled attention condition

Status: Accepted; native and mutation guards verified (see ../transcript-condition-checks-verification.md)

Expose why the bundled low-edit-ratio condition matched or did not match. Add
optional checks to the condition receipt and render each actual, operator,
threshold and pass/fail result. The hypothetical for unknown applicability may
cite the first failed check.

Do not infer diagnostics from arbitrary Datalog. Only `Builtin()` marks the exact
embedded policy as supporting these checks. Policy variants parsed in tests or
future policies without an explicit diagnostic adapter retain status/reason/ratio
without checks. Thus diagnostics cannot contradict a changed rule operator.

For determinate bundled evaluation, report minimum commands and maximum edit
ratio checks. For indeterminate input, preserve its explicit reason and omit
unevaluated comparisons. Old receipts without checks remain readable. Checks do
not alter rule evaluation, applicability, outcome precedence, or task authority.
