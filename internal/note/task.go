package note

import "strings"

// checkboxLine reports whether a line is a markdown checkbox item (optional leading spaces).
func checkboxLine(line string) (isCheckbox bool, checked bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- [ ] ") || trimmed == "- [ ]" {
		return true, false
	}
	if strings.HasPrefix(trimmed, "- [x] ") || trimmed == "- [x]" ||
		strings.HasPrefix(trimmed, "- [X] ") || trimmed == "- [X]" {
		return true, true
	}
	return false, false
}

// HasCheckbox reports whether body contains any real checkbox line (checked or unchecked).
func HasCheckbox(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		if isCheckbox, _ := checkboxLine(line); isCheckbox {
			return true
		}
	}
	return false
}

// IsDone reports whether a note body is considered complete.
// A note is done when it has no checkboxes (vacuously) or all checkboxes are checked.
func IsDone(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		isCheckbox, checked := checkboxLine(line)
		if !isCheckbox {
			continue
		}
		if !checked {
			return false
		}
	}
	return true // vacuously true when no checkboxes, or all checked
}
