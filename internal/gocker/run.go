package gocker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:                "run image command [flags]",
	Short:              "Run a container from a downloaded image",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                run,
}

func must(err error, errMsg string) {
	if err != nil {
		fatalf(errMsg+": %v\n", err)
		os.Exit(1)
	}
}

func run(c *cobra.Command, args []string) {
	cnt, err := NewContainer(args)
	if err != nil {
		fatalf("Error during container creation: %v", err)
	}

	if os.Getenv("IS_CHILD") == "1" {
		// Unshare the mount namespace to isolate mounts from host
		must(syscall.Unshare(syscall.CLONE_NEWNS), "Unshare mount namespace")

		// Make all mounts private to prevent mount propagation to parent namespace
		must(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""), "Make mounts private")

		// Set the hostname
		must(syscall.Sethostname([]byte("container")), "Set hostname")

		// Change to Alpine root directory
		must(os.Chdir(cnt.ImgRoot), "Change dir")

		// Change root filesystem
		must(syscall.Chroot("."), "Change root")

		// Change to root directory in the new filesystem
		must(os.Chdir("/"), "Change dir")

		must(os.Chdir(cnt.Cfg.WorkDir), "Warning: failed to chdir to workingDir")

		must(os.MkdirAll("/proc", 0555), "Make proc dir")

		// Mount proc dir inside rootfs
		must(syscall.Mount("proc", "/proc", "proc", 0, ""), "Mount proc dir")
		defer syscall.Unmount("/proc", 0)

		// Create the command
		cmd := exec.Command(cnt.Cmd, cnt.Args...)

		// Forward all standard streams exactly as they are
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Env = cnt.Cfg.Env
		cmd.Dir = cnt.Cfg.WorkDir

		// Execute command
		if err := cmd.Run(); err != nil {
			// Clean up before exit
			syscall.Unmount("/proc", 0)

			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			} else {
				fatalf("Error: %v", err)
			}
		}

		return
	}

	setupCgroups()
	defer cleanupCgroups()

	// Recreate the command for the child process
	cmd := exec.Command("/proc/self/exe", os.Args[1:]...)

	// Set IS_CHILD environment variable
	cmd.Env = append(cnt.Cfg.Env, "IS_CHILD=1")

	// Forward all standard streams exactly as they are
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Use a new UTS. PID and Mount namespaces
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		GidMappingsEnableSetgroups: false, // disable setgroups to avoid EPERM
	}

	// Re-execute command
	if err := cmd.Run(); err != nil {
		cleanupCgroups() // os.Exit skips defered calls

		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			fatalf("Error: %v", err)
		}
	}
}

// setupCgroups creates and configures a new v2 cgroup for the container process
func setupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	must(os.MkdirAll(cgroupPath, 0755), "Failed to create cgroup v2 path")

	// Set container's memory limit
	memoryMaxFile := filepath.Join(cgroupPath, "memory.max")
	must(os.WriteFile(memoryMaxFile, []byte("50M"), 0644), "Failed to set memory limit")

	// Set container's CPU limit (20%)
	// Format: "<max> <period>" where max and period are in microseconds
	cpuMaxFile := filepath.Join(cgroupPath, "cpu.max")
	must(os.WriteFile(cpuMaxFile, []byte("20000 100000"), 0644), "Failed to set CPU limit")

	// Add current process to the cgroup
	cgroupProcsFile := filepath.Join(cgroupPath, "cgroup.procs")
	must(os.WriteFile(cgroupProcsFile, []byte(strconv.Itoa(os.Getpid())), 0644),
		"Failed to add process to cgroup")
}

// cleanupCgroups removes the custom cgroup created for the container process
func cleanupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	rootProcs := filepath.Join(cgroupRoot, "cgroup.procs")
	selfPid := []byte(strconv.Itoa(os.Getpid()))

	// Move the current process back to the root cgroup
	if err := os.WriteFile(rootProcs, selfPid, 0644); err != nil {
		fmt.Printf("Warning: Failed to move process out of cgroup: %v\n", err)
	}

	// Now it's safe to remove the cgroup directory
	if err := os.RemoveAll(cgroupPath); err != nil {
		fmt.Printf("Warning: Failed to remove cgroup directory: %v\n", err)
	}
}
