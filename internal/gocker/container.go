package gocker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	cgroupsRoot        = "/sys/fs/cgroup"
	relativeImagesPath = ".local/share/gocker/images/"
)

// Container encapsulates container execution parameters
type Container struct {
	ImgName    string
	ImgRoot    string
	CgroupPath string
	Cmd        string
	Args       []string
	Cfg        ContainerConfig
}

type ContainerConfig struct {
	Env          []string
	WorkDir      string
	Hostname     string
	EnableCgroup bool
}

func NewContainer(args []string) (*Container, error) {
	// Extract image name, subcommand and its arguments
	img, cmd, args := args[0], args[1], args[2:]

	imgRoot := filepath.Join(os.Getenv("HOME"), relativeImagesPath, img, "rootfs")
	cfgPath := filepath.Join(os.Getenv("HOME"), relativeImagesPath, img, ".config.json")

	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupsRoot, cgroupName)

	container := &Container{
		ImgName:    img,
		ImgRoot:    imgRoot,
		CgroupPath: cgroupPath,
		Cmd:        cmd,
		Args:       args,
	}

	err := container.loadConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (c *Container) loadConfig(path string) error {
	cfgFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config file: %s", path)
	}
	defer cfgFile.Close()

	var cfg ImageConfig
	must(json.NewDecoder(cfgFile).Decode(&cfg), "failed to decode config file")

	// Prepare clean container environment
	env, workDir := prepareContainerEnv(&cfg)

	c.Cfg.Env = env
	c.Cfg.WorkDir = workDir
	c.Cfg.Hostname = "new-container"

	return nil
}

// prepareContainerEnv sets the environment variables and working directory
// for a container process based on the provided image configuration.
func prepareContainerEnv(cfg *ImageConfig) ([]string, string) {
	var env []string
	workDir := "/"

	// Set minimal required environment variables
	env = append(env, "HOME=/root")
	env = append(env, "USER=root")
	env = append(env, "SHELL=/bin/sh")
	env = append(env, "TERM=xterm")

	// Append variables from the config file
	env = append(env, cfg.Config.Env...)
	if cfg.Config.WorkingDir != "" {
		workDir = cfg.Config.WorkingDir
	}

	return env, workDir
}

func (c *Container) run() {
	if os.Getenv("IS_CHILD") == "1" {
		c.runChildProcess()
	} else {
		c.runParentProcess()
	}
}

func (c *Container) runParentProcess() {
	c.setupCgroup()
	defer c.cleanupCgroups()

	// Recreate the command for the child process
	cmd := exec.Command("/proc/self/exe", os.Args[1:]...)

	// Set IS_CHILD environment variable
	cmd.Env = append(c.Cfg.Env, "IS_CHILD=1")

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
		c.cleanupCgroups() // os.Exit skips defered calls

		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			fatalf("Error: %v", err)
		}
	}
}

func (c *Container) runChildProcess() {
	// Unshare the mount namespace to isolate mounts from host
	must(syscall.Unshare(syscall.CLONE_NEWNS), "Unshare mount namespace")

	// Make all mounts private to prevent mount propagation to parent namespace
	must(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""), "Make mounts private")

	// Set the hostname
	must(syscall.Sethostname([]byte("container")), "Set hostname")

	// Change to Alpine root directory
	must(os.Chdir(c.ImgRoot), "Change dir")

	// Change root filesystem
	must(syscall.Chroot("."), "Change root")

	// Change to root directory in the new filesystem
	must(os.Chdir("/"), "Change dir")

	must(os.Chdir(c.Cfg.WorkDir), "Warning: failed to chdir to workingDir")

	must(os.MkdirAll("/proc", 0555), "Make proc dir")

	// Mount proc dir inside rootfs
	must(syscall.Mount("proc", "/proc", "proc", 0, ""), "Mount proc dir")
	defer syscall.Unmount("/proc", 0)

	// Create the command
	cmd := exec.Command(c.Cmd, c.Args...)

	// Forward all standard streams exactly as they are
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = c.Cfg.Env
	cmd.Dir = c.Cfg.WorkDir

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
}

// setupCgroup creates and configures a new v2 cgroup for the container process
func (c *Container) setupCgroup() {
	must(os.MkdirAll(c.CgroupPath, 0755), "Failed to create cgroup v2 path")

	// Set container's memory limit
	memoryMaxFile := filepath.Join(c.CgroupPath, "memory.max")
	must(os.WriteFile(memoryMaxFile, []byte("50M"), 0644), "Failed to set memory limit")

	// Set container's CPU limit (20%)
	// Format: "<max> <period>" where max and period are in microseconds
	cpuMaxFile := filepath.Join(c.CgroupPath, "cpu.max")
	must(os.WriteFile(cpuMaxFile, []byte("20000 100000"), 0644), "Failed to set CPU limit")

	// Add current process to the cgroup
	cgroupProcsFile := filepath.Join(c.CgroupPath, "cgroup.procs")
	must(os.WriteFile(cgroupProcsFile, []byte(strconv.Itoa(os.Getpid())), 0644),
		"Failed to add process to cgroup")
}

// cleanupCgroups removes the custom cgroup created for the container process
func (c *Container) cleanupCgroups() {
	rootProcs := filepath.Join(cgroupsRoot, "cgroup.procs")
	selfPid := []byte(strconv.Itoa(os.Getpid()))

	// Move the current process back to the root cgroup
	if err := os.WriteFile(rootProcs, selfPid, 0644); err != nil {
		fmt.Printf("Warning: Failed to move process out of cgroup: %v\n", err)
	}

	// Now it's safe to remove the cgroup directory
	if err := os.RemoveAll(c.CgroupPath); err != nil {
		fmt.Printf("Warning: Failed to remove cgroup directory: %v\n", err)
	}
}
