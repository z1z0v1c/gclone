package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var run = &cobra.Command{
	Use:                "run [image] [command]",
	Short:              "Run a container from a downloaded image",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                Run,
}

func Run(c *cobra.Command, args []string) {
	// Extract the image name, subcommand and its flags
	image, subcmd, argz := args[0], args[1], args[2:]

	rootfs := filepath.Join("./", image)
	configPath := filepath.Join("./", image, ".config.json")

	configFile, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Can't open config file: %s", configPath)
	}
	defer configFile.Close()

	var cfg ImageConfig
	if err := json.NewDecoder(configFile).Decode(&cfg); err != nil {
		log.Fatalf("Can't decode config file: %s", configPath)
	}

	// Prepare clean container environment
	var env []string
	workingDir := "/"

	// Set minimal required environment variables
	env = append(env, "HOME=/root")
	env = append(env, "USER=root")
	env = append(env, "SHELL=/bin/sh")
	env = append(env, "TERM=xterm")

	env = append(env, cfg.Config.Env...)
	if cfg.Config.WorkingDir != "" {
		workingDir = cfg.Config.WorkingDir
	}

	if os.Getenv("IS_CHILD") == "1" {
		// Unshare the mount namespace to isolate mounts from host
		if err := syscall.Unshare(syscall.CLONE_NEWNS); err != nil {
			log.Fatalf("Unshare mount namespace: %v", err)
		}

		// Make all mounts private to prevent mount propagation to parent namespace
		if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
			log.Fatalf("Make mounts private: %v", err)
		}

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

		if err := os.Chdir(workingDir); err != nil {
			log.Printf("Warning: failed to chdir to workingDir: %v, staying in /", err)
		}

		if err := os.MkdirAll("/proc", 0555); err != nil {
			log.Fatalf("Make proc dir: %v", err)
		}

		// Mount proc dir inside rootfs
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			log.Fatalf("Mount proc dir: %v", err)
		}
		defer syscall.Unmount("/proc", 0)

		// Create the command
		cmd := exec.Command(subcmd, argz...)

		// Forward all standard streams exactly as they are
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Env = env
		cmd.Dir = workingDir

		// Execute command
		if err := cmd.Run(); err != nil {
			// Clean up before exit
			syscall.Unmount("/proc", 0)

			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			} else {
				log.Fatalf("Error: %v", err)
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
			log.Fatalf("Error: %v", err)
		}
	}
}

func setupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		log.Fatalf("Failed to create cgroup v2 path: %v", err)
	}

	// Set memory limit in bytes
	if err := os.WriteFile(filepath.Join(cgroupPath, "memory.max"), []byte("50M"), 0644); err != nil {
		log.Fatalf("Failed to set memory limit: %v", err)
	}

	// Set CPU limit (20%)
	// Format: "<max> <period>" where max and period are in microseconds
	if err := os.WriteFile(filepath.Join(cgroupPath, "cpu.max"), []byte("20000 100000"), 0644); err != nil {
		log.Fatalf("Failed to set CPU limit: %v", err)
	}

	// Add current process to the cgroup
	if err := os.WriteFile(filepath.Join(cgroupPath, "cgroup.procs"),
		[]byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		log.Fatalf("Failed to add process to cgroup: %v", err)
	}
}

func cleanupCgroups() {
	cgroupRoot := "/sys/fs/cgroup"
	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupRoot, cgroupName)

	// Move the current process back to the root cgroup
	rootProcs := filepath.Join(cgroupRoot, "cgroup.procs")
	selfPid := []byte(strconv.Itoa(os.Getpid()))

	if err := os.WriteFile(rootProcs, selfPid, 0644); err != nil {
		log.Printf("Warning: Failed to move process out of cgroup: %v", err)
		return
	}

	// Now it's safe to remove the cgroup directory
	if err := os.RemoveAll(cgroupPath); err != nil {
		log.Printf("Warning: Failed to remove cgroup directory: %v", err)
	}
}
