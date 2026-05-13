#!/usr/bin/env bash
# Fires after any tool call fails (PostToolUseFailure).
# Injects a system-reminder telling Claude to search nn before workarounding
# the failure. Exit 2 feeds stderr back to Claude as a system-reminder.
#
# Skip condition (ADR-0014): if the most recent assistant turn contains a line
# whose first two words (case-insensitive) are "Expected FAIL:", the lookup
# is not required. The assistant must cite it verbatim when skipping:
#   Skipping lookup: cited "Expected FAIL: <reason>"
printf '<system-reminder>\nA tool call just failed unexpectedly. Before attempting a workaround or fix, run:\n\n  nn list --search "<topic>" --json\n\nPrior sessions may have captured the root cause, a known workaround, or a relevant constraint. Search nn first.\n\nSkip condition: if the immediately preceding assistant turn contains a line\nbeginning with "Expected FAIL: <reason>" (case-insensitive), the lookup is\nnot required. To skip, write: Skipping lookup: cited "Expected FAIL: <reason>"\n— quoting the exact declaration verbatim. Absence of that verbatim quote means\nthe skip condition is not met and the lookup is required.\n</system-reminder>\n' >&2
exit 2
