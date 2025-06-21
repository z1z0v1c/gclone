package cmd

import (
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

const rootfs = "./alpine"

var run = &cobra.Command{
	Use:                "run [command]",
	Short:              "Execute any Linux command",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                Run,
}

func Run(c *cobra.Command, args []string) {
	if os.Getenv("IS_CHILD") == "1" {
		// Set the hostname
		if err := syscall.Sethostname([]byte("container")); err != nil {
			log.Fatalf("Set hostname: %v", err)
		}

		// Change to Alpine root directory
		if err := os.Chdir(rootfs); err != nil {
			log.Fatalf("Change dir: %v", err)
		}

		// Change root filesystem
		if err := syscall.Chroot("."); err != nil {
			log.Fatalf("Change root: %v", err)
		}

		// Change to root directory in the new filesystem
		if err := os.Chdir("/"); err != nil {
			log.Fatalf("Change dir: %v", err)
		}

		if err := os.MkdirAll("/proc", 0555); err != nil {
			log.Fatalf("Make proc dir: %v", err)
		}

		// Mount proc dir inside rootfs
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			log.Fatalf("Mount proc dir: %v", err)
		}

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
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			} else {
				log.Fatalf("Error: %v", err)
			}
		}

		// Clean up
		if err := syscall.Unmount("/proc", 0); err != nil {
			log.Fatalf("Unmount proc: %v", err)
		}

		return
	}

	// Recreate the command for the child process
	cmd := exec.Command("/proc/self/exe", os.Args[1:]...)

	// Set IS_CHILD environment variable
	cmd.Env = append(os.Environ(), "IS_CHILD=1")

	// Forward all standard streams exactly as they are
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Use a new UTS namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}

	// Re-execute command
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			log.Fatalf("Error: %v", err)
		}
	}
}
