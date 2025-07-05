package gocker

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var gocker = &cobra.Command{
	Use:   "gocker [command]",
	Short: "My own version of Docker",
}

func init() {
	gocker.AddCommand(runCmd, pullCmd)
}

func Execute() {
	if err := gocker.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
