package cmd

import (
	"log"
	"os"
	"os/exec"
	"syscall"

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

	// Set SysProcAttr to use a new UTS namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}

	// Hardcode hostname
	if err := syscall.Sethostname([]byte("container")); err != nil {
		log.Fatalf("Set hostname: %v", err)
	}

	// Execute command
	err := cmd.Run()
	if err != nil {
		// Inspect the exit code if it's an ExitError
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			// Other kind of error
			log.Fatalln(err)
		}
	}
}
