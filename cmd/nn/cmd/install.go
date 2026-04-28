package cmd

import (
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		dest   string
		forLLM string
		scope  string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install nn skills and hooks (runs install-skills then install-hooks)",
		Long: `Install nn skills and hooks in one shot.

Runs install-skills followed by install-hooks with the given options.

Presets (--for):
  claude   ~/.claude/skills/  (default)
  cursor   ~/.cursor/skills/
  zed      ~/.config/zed/skills/

Hook scopes (--scope):
  user     ~/.claude/settings.json (default, global)
  project  .claude/settings.json
  local    .claude/settings.local.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			skillsCmd := newInstallSkillsCmd()
			skillsCmd.SetOut(cmd.OutOrStdout())
			skillsCmd.SetErr(cmd.ErrOrStderr())
			if forLLM != "" {
				if err := skillsCmd.Flags().Set("for", forLLM); err != nil {
					return err
				}
			}
			if dest != "" {
				if err := skillsCmd.Flags().Set("dest", dest); err != nil {
					return err
				}
			}
			if err := skillsCmd.RunE(skillsCmd, nil); err != nil {
				return err
			}

			hooksCmd := newInstallHooksCmd()
			hooksCmd.SetOut(cmd.OutOrStdout())
			hooksCmd.SetErr(cmd.ErrOrStderr())
			if scope != "" {
				if err := hooksCmd.Flags().Set("scope", scope); err != nil {
					return err
				}
			}
			return hooksCmd.RunE(hooksCmd, nil)
		},
	}

	cmd.Flags().StringVar(&dest, "dest", "", "Custom destination directory for skills (overrides --for)")
	cmd.Flags().StringVar(&forLLM, "for", "", "Target LLM preset: claude (default), cursor, zed")
	cmd.Flags().StringVar(&scope, "scope", "", "Hook installation scope: user (default), project, or local")
	return cmd
}
