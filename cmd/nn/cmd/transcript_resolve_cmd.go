package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type transcriptResolveResult struct {
	Session string `json:"session"`
	Path    string `json:"path"`
}

func newTranscriptResolveCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "resolve <session>",
		Short: "Resolve an exact session ID to its canonical transcript path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveTranscript(args[0])
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(transcriptResolveResult{Session: args[0], Path: path})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), path)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit session and canonical path as JSON")
	return cmd
}
