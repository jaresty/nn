#!/usr/bin/env bash
# Fires after any tool call fails (PostToolUseFailure).
# Behavior is defined in the virtual-nn-error-handling protocol loaded at session start.
printf '<system-reminder>\nA tool call just failed unexpectedly. Consult the virtual-nn-error-handling protocol loaded at session start for the required response procedure.\n</system-reminder>\n' >&2
exit 2
