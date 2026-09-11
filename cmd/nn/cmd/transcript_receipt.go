package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaresty/nn/internal/note"
	"github.com/spf13/cobra"
)

var transcriptReceiptNow = func() time.Time { return time.Now().UTC() }

func newTranscriptReceiptCmd(state *rootState) *cobra.Command {
	var assignment, disposition, parentSession string
	var adopted, rejected, results, verification []string
	var expiresIn time.Duration
	var linkTos, linkTypes, annotations []string

	cmd := &cobra.Command{
		Use:   "receipt <session>",
		Short: "Create an expiring parent-adjudicated transcript integration receipt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			assignment = strings.TrimSpace(assignment)
			if assignment == "" {
				return fmt.Errorf("--assignment is required")
			}
			switch disposition {
			case "accepted", "partially-adopted", "rejected":
			default:
				return fmt.Errorf("--disposition must be accepted, partially-adopted, or rejected")
			}
			if expiresIn <= 0 {
				return fmt.Errorf("--expires-in must be positive")
			}
			if len(linkTos) != len(linkTypes) || len(linkTos) != len(annotations) {
				return fmt.Errorf("--link-to, --link-type, and --annotation must be paired")
			}
			resolvedTargets := make([]string, len(linkTos))
			for i, linkType := range linkTypes {
				if !note.IsKnownLinkType(linkType) {
					return fmt.Errorf("invalid --link-type %q: must be one of %s", linkType, strings.Join(note.LinkTypeOrder, ", "))
				}
				target, err := resolveNote(state, linkTos[i])
				if err != nil {
					return fmt.Errorf("receipt link: %w", err)
				}
				resolvedTargets[i] = target.ID
			}

			resolved, err := resolveTranscript(args[0])
			if err != nil {
				return err
			}
			info, err := os.Stat(resolved)
			if err != nil {
				return fmt.Errorf("transcript receipt: %w", err)
			}
			if info.IsDir() {
				return fmt.Errorf("transcript receipt: %s is a directory", resolved)
			}
			sessionID := transcriptMetadataID(resolved)
			if sessionID == "" {
				base := filepath.Base(resolved)
				sessionID = strings.TrimSuffix(strings.TrimSuffix(base, ".jsonl"), ".output")
			}
			provider := classifyTranscript(resolved)
			now := transcriptReceiptNow()
			expires := now.Add(expiresIn)

			var body strings.Builder
			fmt.Fprintf(&body, "provider: %s\nsession: %s\n", provider, sessionID)
			if strings.TrimSpace(parentSession) != "" {
				fmt.Fprintf(&body, "parent_session: %s\n", strings.TrimSpace(parentSession))
			}
			fmt.Fprintf(&body, "disposition: %s\n\n## Assignment\n\n%s\n", disposition, assignment)
			writeReceiptSection(&body, "Adopted", adopted)
			writeReceiptSection(&body, "Rejected or revised", rejected)
			writeReceiptSection(&body, "Result", results)
			writeReceiptSection(&body, "Verification", verification)
			fmt.Fprintf(&body, "\n## Reconstruction\n\n```bash\nnn transcript show %s\n```\n", sessionID)

			n := &note.Note{
				ID: note.GenerateID(), Title: "Integration receipt: " + assignment,
				Type: note.TypeObservation, Status: note.StatusDraft,
				Tags:    []string{"subagent-handoff", "transcript-receipt"},
				Created: now, Modified: now, Expires: &expires, Body: body.String(),
			}
			for i, target := range resolvedTargets {
				n.Links = append(n.Links, note.Link{TargetID: target, Type: linkTypes[i], Annotation: annotations[i]})
			}
			if err := state.backend.Write(n); err != nil {
				return fmt.Errorf("create transcript receipt: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "created %s\nsession: %s\nexpires: %s\n", n.ID, sessionID, expires.Format("2006-01-02"))
			return nil
		},
	}
	cmd.Flags().StringVar(&assignment, "assignment", "", "Delegated assignment (required)")
	cmd.Flags().StringVar(&disposition, "disposition", "", "Parent disposition: accepted|partially-adopted|rejected (required)")
	cmd.Flags().StringArrayVar(&adopted, "adopted", nil, "Adopted claim or contribution (repeatable)")
	cmd.Flags().StringArrayVar(&rejected, "rejected", nil, "Rejected or revised claim (repeatable)")
	cmd.Flags().StringArrayVar(&results, "result", nil, "Resulting artifact, decision, or commit (repeatable)")
	cmd.Flags().StringArrayVar(&verification, "verification", nil, "Parent verification result (repeatable)")
	cmd.Flags().StringVar(&parentSession, "parent-session", "", "Parent session ID when known")
	cmd.Flags().DurationVar(&expiresIn, "expires-in", 14*24*time.Hour, "Receipt lifetime (default 336h)")
	cmd.Flags().StringArrayVar(&linkTos, "link-to", nil, "Link receipt to an existing note ID (repeatable)")
	cmd.Flags().StringArrayVar(&linkTypes, "link-type", nil, "Known link type paired with --link-to")
	cmd.Flags().StringArrayVar(&annotations, "annotation", nil, "Link annotation paired with --link-to")
	return cmd
}

func writeReceiptSection(body *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(body, "\n## %s\n", title)
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			fmt.Fprintf(body, "\n- %s", value)
		}
	}
	body.WriteByte('\n')
}
