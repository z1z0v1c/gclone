package gnetcat

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// gnc is the root Cobra command for the Gnetcat utility
var gnc = &cobra.Command{
	Use:   "gnc command [flags]",
	Short: "Simple Netcat clone written in Go",
}

func main() {
	if err := gnc.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err); os.Exit(1)
	}
}
