package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newRemindCmd(state *rootState) *cobra.Command {
	var (
		forDays      int
		expiresStr   string
		findFragment string
		updateID     string
	)

	cmd := &cobra.Command{
		Use:   "remind <content>",
		Short: "Create a temporary reminder note surfaced at session start",
		Args: func(cmd *cobra.Command, args []string) error {
			find, _ := cmd.Flags().GetString("find")
			update, _ := cmd.Flags().GetString("update")
			if find != "" || update != "" {
				return cobra.MaximumNArgs(1)(cmd, args)
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var content string
			if len(args) > 0 {
				content = args[0]
			}

			if findFragment != "" {
				notes, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("remind --find: %w", err)
				}
				var matches []*note.Note
				for _, n := range notes {
					if !hasTag(n, "reminder") {
						continue
					}
					if strings.Contains(strings.ToLower(n.Title), strings.ToLower(findFragment)) {
						matches = append(matches, n)
					}
				}
				if len(matches) == 0 {
					return fmt.Errorf("remind --find: no reminder matching %q", findFragment)
				}
				if len(matches) > 1 {
					var ids []string
					for _, m := range matches {
						ids = append(ids, m.ID+" "+m.Title)
					}
					return fmt.Errorf("remind --find: ambiguous — %d matches:\n%s", len(matches), strings.Join(ids, "\n"))
				}
				fmt.Fprintln(outWriter(cmd), matches[0].ID)
				return nil
			}

			if updateID != "" {
				existing, err := state.backend.Read(updateID)
				if err != nil {
					return fmt.Errorf("remind --update: %w", err)
				}
				existing.Body = content
				existing.Modified = time.Now().UTC()
				if err := state.backend.Update(existing); err != nil {
					return fmt.Errorf("remind --update: %w", err)
				}
				fmt.Fprintf(outWriter(cmd), "updated %s\n", existing.ID)
				return nil
			}

			title := content
			if len(title) > 60 {
				title = title[:60]
			}

			now := time.Now().UTC()
			var expires time.Time
			switch {
			case expiresStr != "":
				t, err := time.Parse("2006-01-02", expiresStr)
				if err != nil {
					return fmt.Errorf("--expires: invalid date %q, want YYYY-MM-DD", expiresStr)
				}
				expires = t
			case forDays > 0:
				expires = now.Add(time.Duration(forDays) * 24 * time.Hour)
			default:
				expires = now.Add(24 * time.Hour)
			}

			n := &note.Note{
				ID:       note.GenerateID(),
				Title:    title,
				Type:     note.TypeObservation,
				Status:   note.StatusPermanent,
				Tags:     []string{"reminder"},
				Expires:  &expires,
				Created:  now,
				Modified: now,
				Body:     content,
			}

			if err := state.backend.Write(n); err != nil {
				return fmt.Errorf("remind: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "created %s (expires %s)\n", n.ID, expires.Format("2006-01-02"))
			return nil
		},
	}

	cmd.Flags().IntVar(&forDays, "for", 0, "Expire after N days (e.g. --for 2); default 1")
	cmd.Flags().StringVar(&expiresStr, "expires", "", "Explicit expiry date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&findFragment, "find", "", "Search reminder titles by substring; print matching IDs")
	cmd.Flags().StringVar(&updateID, "update", "", "Update body of existing reminder in place (preserves expiry)")
	return cmd
}

// parseForFlag parses strings like "2d", "7d" into a day count.
// Returns 0 if the string is empty or unparseable.
func parseForFlag(s string) int {
	s = strings.TrimSuffix(strings.TrimSpace(s), "d")
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
