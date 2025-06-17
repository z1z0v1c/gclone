package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var gocker = &cobra.Command{
	Use:   "gocker",
	Short: "My own version of Docker",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Greetings from Gocker!")
	},
}

func init() {
  gocker.AddCommand(run)
}

func Execute() {
	if err := gocker.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
