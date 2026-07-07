package cmd

import (
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		dest           string
		forLLM         string
		scope          string
		skillsDest     string
		extensionsDest string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install nn support for an LLM harness (skills + harness-specific integrations)",
		Long: `Install nn support for an LLM harness in one shot.

Presets (--for):
  claude   ~/.claude/skills/ + Claude Code hooks  (default)
  cursor   ~/.cursor/skills/
  zed      ~/.config/zed/skills/
  pi       ~/.pi/agent/skills/ + Pi extensions

For Claude, runs install-skills then install-hooks.
For Pi, runs install-skills --for pi then install-extensions.
For Cursor and Zed, runs install-skills only.

Hook scopes (--scope, Claude only):
  user     ~/.claude/settings.json (default, global)
  project  .claude/settings.json
  local    .claude/settings.local.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if forLLM == "" {
				forLLM = "claude"
			}

			if forLLM == "pi" {
				skillsCmd := newInstallSkillsCmd()
				skillsCmd.SetOut(cmd.OutOrStdout())
				skillsCmd.SetErr(cmd.ErrOrStderr())
				if err := skillsCmd.Flags().Set("for", "pi"); err != nil {
					return err
				}
				if skillsDest != "" {
					if err := skillsCmd.Flags().Set("dest", skillsDest); err != nil {
						return err
					}
				}
				if err := skillsCmd.RunE(skillsCmd, nil); err != nil {
					return err
				}

				extCmd := newInstallExtensionsCmd()
				extCmd.SetOut(cmd.OutOrStdout())
				extCmd.SetErr(cmd.ErrOrStderr())
				if extensionsDest != "" {
					if err := extCmd.Flags().Set("extensions-dest", extensionsDest); err != nil {
						return err
					}
				}
				return extCmd.RunE(extCmd, nil)
			}

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

			if forLLM != "claude" {
				return nil
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

	cmd.Flags().StringVar(&dest, "dest", "", "Custom destination directory for skills (overrides --for, Claude/Cursor/Zed only)")
	cmd.Flags().StringVar(&forLLM, "for", "", "Target LLM preset: claude (default), cursor, zed, pi")
	cmd.Flags().StringVar(&scope, "scope", "", "Hook installation scope: user (default), project, or local (Claude only)")
	cmd.Flags().StringVar(&skillsDest, "skills-dest", "", "Custom Pi skills directory (default: ~/.pi/agent/skills, Pi only)")
	cmd.Flags().StringVar(&extensionsDest, "extensions-dest", "", "Custom Pi extensions directory (default: ~/.pi/agent/extensions, Pi only)")
	return cmd
}
