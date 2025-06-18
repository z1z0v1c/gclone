package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var run = &cobra.Command{
	Use:                "run [command]",
	Short:              "Execute any Linux command",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                Run,
}

func Run(c *cobra.Command, args []string) {
	// Extract the subcommand and its flags
	subcmd := args[0]
	flags := args[1:]

	// Create the command
	cmd := exec.Command(subcmd, flags...)

	// Forward all standard streams exactly as they are
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute command
	err := cmd.Run()
	if err != nil {
		// Inspect the exit code if it's an ExitError 
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			// Other kind of error
			os.Exit(1)
		}
	}
}
