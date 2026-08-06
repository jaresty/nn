package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// newSearchCmd searches notes using BM25 ranking.
func newSearchCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var sortBy string
	var limit int
	var filterStatus string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Alias for list --search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			listCmd := newListCmd(state)
			listCmd.SetOut(cmd.OutOrStdout())
			listCmd.SetErr(cmd.ErrOrStderr())
			listCmd.SetContext(cmd.Context())
			setFlag := func(name, value string) error {
				if err := listCmd.Flags().Set(name, value); err != nil {
					return fmt.Errorf("search: set --%s: %w", name, err)
				}
				return nil
			}
			if err := setFlag("search", args[0]); err != nil {
				return err
			}
			if jsonOut {
				if err := setFlag("json", "true"); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("sort") {
				if err := setFlag("sort", sortBy); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("limit") {
				if err := setFlag("limit", strconv.Itoa(limit)); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("status") {
				if err := setFlag("status", filterStatus); err != nil {
					return err
				}
			}
			return listCmd.RunE(listCmd, nil)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by: title, modified, created")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by note status")
	return cmd
}

const excerptLen = 120

// extractHeadingBreadcrumb returns the heading breadcrumb (e.g. "## Foo > ### Bar")
// for the position matchIdx in body, by walking backward to collect ancestor headings.
func extractHeadingBreadcrumb(body string, matchIdx int) string {
	lines := strings.Split(body[:matchIdx], "\n")
	// headings[level] = most recent heading text at that level (1=H1, 2=H2, 3=H3)
	headings := make(map[int]string)
	for _, line := range lines {
		for level := 1; level <= 6; level++ {
			prefix := strings.Repeat("#", level) + " "
			if strings.HasPrefix(line, prefix) {
				headings[level] = strings.TrimSpace(line)
				// clear all deeper levels when a shallower heading is found
				for deeper := level + 1; deeper <= 6; deeper++ {
					delete(headings, deeper)
				}
				break
			}
		}
	}
	if len(headings) == 0 {
		return ""
	}
	var parts []string
	for level := 1; level <= 6; level++ {
		if h, ok := headings[level]; ok {
			parts = append(parts, h)
		}
	}
	return strings.Join(parts, " > ")
}

// extractExcerpt returns a ≤120-char snippet from body containing the first
// query term found, with leading "..." when the snippet is not from the start.
func extractExcerpt(body, query string) string {
	if body == "" || query == "" {
		return ""
	}
	lower := strings.ToLower(body)
	for _, term := range strings.Fields(strings.ToLower(query)) {
		idx := strings.Index(lower, term)
		if idx < 0 {
			continue
		}
		start := idx - 40
		if start < 0 {
			start = 0
		}
		end := start + excerptLen
		if end > len(body) {
			end = len(body)
		}
		// trim to valid UTF-8 rune boundaries
		for start > 0 && !utf8.RuneStart(body[start]) {
			start--
		}
		for end < len(body) && !utf8.RuneStart(body[end]) {
			end++
		}
		snippet := body[start:end]
		if start > 0 {
			snippet = "..." + snippet
		}
		if crumb := extractHeadingBreadcrumb(body, idx); crumb != "" {
			snippet = crumb + " | " + snippet
		}
		return snippet
	}
	return ""
}
