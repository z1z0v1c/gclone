package gocker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:                "run [image] [command]",
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
	// Extract image name, subcommand and its arguments
	img, subcmd, argz := args[0], args[1], args[2:]

	imgRoot := filepath.Join(os.Getenv("HOME"), relativeImagesPath, img, "rootfs")
	cfgPath := filepath.Join(os.Getenv("HOME"), relativeImagesPath, img, ".config.json")

	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		fatalf("failed to open config file: %s", cfgPath)
	}
	defer cfgFile.Close()

	var cfg ImageConfig
	must(json.NewDecoder(cfgFile).Decode(&cfg), "failed to decode config file")

	// Prepare clean container environment
	env, workDir := prepareContainerEnv(&cfg)

	if os.Getenv("IS_CHILD") == "1" {
		// Unshare the mount namespace to isolate mounts from host
		must(syscall.Unshare(syscall.CLONE_NEWNS), "Unshare mount namespace")

		// Make all mounts private to prevent mount propagation to parent namespace
		must(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""), "Make mounts private")

		// Set the hostname
		must(syscall.Sethostname([]byte("container")), "Set hostname")

		// Change to Alpine root directory
		must(os.Chdir(imgRoot), "Change dir")

		// Change root filesystem
		must(syscall.Chroot("."), "Change root")

		// Change to root directory in the new filesystem
		must(os.Chdir("/"), "Change dir")

		must(os.Chdir(workDir), "Warning: failed to chdir to workingDir")

		must(os.MkdirAll("/proc", 0555), "Make proc dir")

		// Mount proc dir inside rootfs
		must(syscall.Mount("proc", "/proc", "proc", 0, ""), "Mount proc dir")
		defer syscall.Unmount("/proc", 0)

		// Create the command
		cmd := exec.Command(subcmd, argz...)

		// Forward all standard streams exactly as they are
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Env = env
		cmd.Dir = workDir

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
	cmd.Env = append(env, "IS_CHILD=1")

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
		cleanupCgroups() // os.Exit skips deferd calls

		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			fatalf("Error: %v", err)
		}
	}
}

func prepareContainerEnv(cfg *ImageConfig) ([]string, string) {
	var env []string
	workDir := "/"

	// Set minimal required environment variables
	env = append(env, "HOME=/root")
	env = append(env, "USER=root")
	env = append(env, "SHELL=/bin/sh")
	env = append(env, "TERM=xterm")

	env = append(env, cfg.Config.Env...)
	if cfg.Config.WorkingDir != "" {
		workDir = cfg.Config.WorkingDir
	}

	return env, workDir
}

func setupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	must(os.MkdirAll(cgroupPath, 0755), "Failed to create cgroup v2 path")

	// Set memory limit in bytes
	must(
		os.WriteFile(filepath.Join(cgroupPath, "memory.max"), []byte("50M"), 0644),
		"Failed to set memory limit",
	)

	// Set CPU limit (20%)
	must(
		// Format: "<max> <period>" where max and period are in microseconds
		os.WriteFile(filepath.Join(cgroupPath, "cpu.max"), []byte("20000 100000"), 0644),
		"Failed to set CPU limit",
	)

	// Add current process to the cgroup
	must(
		os.WriteFile(filepath.Join(cgroupPath, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0644),
		"Failed to add process to cgroup",
	)
}

func cleanupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	// Move the current process back to the root cgroup
	rootProcs := filepath.Join(cgroupRoot, "cgroup.procs")
	selfPid := []byte(strconv.Itoa(os.Getpid()))

	must(os.WriteFile(rootProcs, selfPid, 0644), "Warning: Failed to move process out of cgroup")

	// Now it's safe to remove the cgroup directory
	must(os.RemoveAll(cgroupPath), "Warning: Failed to remove cgroup directory")
}
