#!/usr/bin/env bash
# Fires after any tool call fails (PostToolUseFailure).
# Injects a system-reminder telling Claude to search nn before workarounding
# the failure. Exit 2 feeds stderr back to Claude as a system-reminder.
printf '<system-reminder>\nA tool call just failed unexpectedly. Before attempting a workaround or fix, run:\n\n  nn list --search "<topic>" --json\n\nPrior sessions may have captured the root cause, a known workaround, or a relevant constraint. Search nn first.\n</system-reminder>\n' >&2
exit 2
