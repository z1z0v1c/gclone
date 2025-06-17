package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var run = &cobra.Command{
	Use:   "run",
	Short: "Equivalent to the docker run subcommand",
	Run:   Run,
}

func Run(cmd *cobra.Command, args []string) {
	c := exec.Command("ls", "-l")

	// Capture output
	output, err := c.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return
	}

	fmt.Printf("%s\n", output)
}
