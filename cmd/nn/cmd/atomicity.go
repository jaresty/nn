package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// atomicityThreshold is the body length (in bytes) above which a warning is emitted.
const atomicityThreshold = 2000

// warnIfLarge writes a warning to stderr if body exceeds atomicityThreshold.
func warnIfLarge(cmd *cobra.Command, body string) {
	if len(body) > atomicityThreshold {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: note body is %d chars — consider splitting into atomic notes\n", len(body))
	}
}
