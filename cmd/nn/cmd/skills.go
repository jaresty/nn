package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	nnSkills "github.com/jaresty/nn/skills"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "List or retrieve embedded nn skills",
	}
	cmd.AddCommand(newSkillsListCmd(), newSkillsGetCmd())
	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available embedded skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := nnSkills.FS.ReadDir(".")
			if err != nil {
				return fmt.Errorf("read embedded skills: %w", err)
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := e.Name()
				desc := skillDescription(name)
				fmt.Fprintf(outWriter(cmd), "  %-28s %s\n", name, desc)
			}
			return nil
		},
	}
}

func newSkillsGetCmd() *cobra.Command {
	var (
		listReferences bool
		reference      string
	)
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Print a named skill core or one of its references",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if listReferences {
				refs, err := nnSkills.ListReferences(nnSkills.FS, name)
				if err != nil {
					return err
				}
				for _, ref := range refs {
					fmt.Fprintf(outWriter(cmd), "%s\t%s\n", ref.Name, ref.AppliesWhen)
				}
				return nil
			}

			var (
				data []byte
				err  error
			)
			if cmd.Flags().Changed("reference") {
				data, err = nnSkills.ReadReference(nnSkills.FS, name, reference)
			} else {
				data, err = nnSkills.ReadSkill(nnSkills.FS, name)
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(outWriter(cmd), string(data))
			return err
		},
	}
	cmd.Flags().BoolVar(&listReferences, "list-references", false, "List available reference names and applicability")
	cmd.Flags().StringVar(&reference, "reference", "", "Print one reference by logical name")
	cmd.MarkFlagsMutuallyExclusive("list-references", "reference")
	return cmd
}

// skillDescription extracts the description field from a skill's SKILL.md frontmatter.
func skillDescription(name string) string {
	data, err := nnSkills.FS.ReadFile(name + "/SKILL.md")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}
