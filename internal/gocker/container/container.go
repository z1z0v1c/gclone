package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/z1z0v1c/gclone/internal/gocker/image"
	"github.com/z1z0v1c/gclone/internal/gocker/registry"
)

const (
	cgroupsRoot = "/sys/fs/cgroup"
)

// Container encapsulates container execution parameters.
type Container struct {
	registry.Config
	imgName    string
	imgRoot    string
	cgroupPath string
	cmd        string
	args       []string
}

// NewContainer creates a new Container from the given arguments.
func NewContainer(imgName, cmd string, args []string) (*Container, error) {
	imgRoot := filepath.Join(os.Getenv("HOME"), image.RelativeImagesPath, imgName, "rootfs")
	cfgPath := filepath.Join(os.Getenv("HOME"), image.RelativeImagesPath, imgName, ".config.json")

	cgroupName := fmt.Sprintf("gocker%d", os.Getpid())
	cgroupPath := filepath.Join(cgroupsRoot, cgroupName)

	c := &Container{
		imgName:    imgName,
		imgRoot:    imgRoot,
		cgroupPath: cgroupPath,
		cmd:        cmd,
		args:       args,
	}

	err := c.fromFile(cfgPath)
	if err != nil {
		return nil, err
	}

	// Append minimal required environment variables
	c.Env = append(c.Env, "HOME=/root", "USER=root", "SHELL=/bin/sh", "TERM=xterm")

	return c, nil
}

// Run starts the container execution.
func (c *Container) Run() error {
	var err error

	if os.Getenv("IS_CHILD") == "1" {
		err = c.runChildProcess()
	} else {
		err = c.runParentProcess()
	}

	return err
}

// runParentProcess sets up cgroups and forks a child process with namespace isolation.
func (c *Container) runParentProcess() error {
	c.setupCgroup()
	defer c.cleanupCgroup()

	// Recreate the command for the child process
	cmd := exec.Command("/proc/self/exe", os.Args[1:]...)

	cmd.Env = append(c.Env, "IS_CHILD=1")

	// Forward all standard streams exactly as they are
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// Use a new UTS. PID, Mount and User namespaces
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

	return cmd.Run()
}

// runChildProcess performs setup for the isolated container
// environment and executes the target command inside it.
func (c *Container) runChildProcess() error {
	if err := c.setupNamespaces(); err != nil {
		return err
	}

	if err := c.setupFilesystem(); err != nil {
		return err
	}

	if err := c.mountProc(); err != nil {
		return err
	}
	defer c.unmountProc()

	// Create the command
	cmd := exec.Command(c.cmd, c.args...)

	// Forward all standard streams exactly as they are
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Env, cmd.Dir = c.Env, c.WorkingDir

	return cmd.Run()
}

// setupNamespaces sets up namespaces isolation.
func (c *Container) setupNamespaces() error {
	// Unshare the mount namespace to isolate mounts from host
	if err := syscall.Unshare(syscall.CLONE_NEWNS); err != nil {
		return fmt.Errorf("failed to unshare mount namespace: %v", err)
	}

	// Make all mounts private to prevent mount propagation to parent namespace
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to make mounts private: %v", err)
	}

	if err := syscall.Sethostname([]byte(c.Hostname)); err != nil {
		return fmt.Errorf("failed to set hostname: %v", err)
	}

	return nil
}

// setupFilesystem changes the root filesystem to the container's rootfs.
func (c *Container) setupFilesystem() error {
	if err := os.Chdir(c.imgRoot); err != nil {
		return fmt.Errorf("failed to change dir: %v", err)
	}

	// Change root filesystem
	if err := syscall.Chroot("."); err != nil {
		return fmt.Errorf("failed to change root: %v", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to change dir: %v", err)
	}

	if err := os.Chdir(c.WorkingDir); err != nil {
		fmt.Printf("WARNING: failed to chdir to working dir: %v\n", err)
	}

	return nil
}

// mountProc mounts the /proc filesystem inside the container.
func (c *Container) mountProc() error {
	if err := os.MkdirAll("/proc", 0555); err != nil {
		return fmt.Errorf("failed to create proc dir: %v", err)
	}

	// Mount proc filesystem inside image's rootfs
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount proc dir: %v", err)
	}

	return nil
}

// unmountProc unmounts the /proc filesystem before exiting.
func (c *Container) unmountProc() {
	if err := syscall.Unmount("/proc", 0); err != nil {
		fmt.Printf("WARNING: failed to unmount proc dir: %v\n", err)
	}
}

// fromFile loads environment variables, hostname,
// and working directory from the image config file.
func (c *Container) fromFile(cfgPath string) error {
	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %s", cfgPath)
	}
	defer cfgFile.Close()

	var cfg registry.ImageConfig
	if err = json.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		return fmt.Errorf("failed to decode config file: %v", err)
	}

	c.Config = cfg.Config

	if c.WorkingDir == "" {
		c.WorkingDir = "/"
	}
	if c.Hostname == "" {
		c.Hostname = c.imgName + "-container"
	}

	return nil
}

// setupCgroup creates and configures a new v2 cgroup for the container process.
func (c *Container) setupCgroup() error {
	if err := os.MkdirAll(c.cgroupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup v2 path: %v", err)
	}

	// Set container's memory limit
	memoryMaxFile := filepath.Join(c.cgroupPath, "memory.max")
	if err := os.WriteFile(memoryMaxFile, []byte("50M"), 0644); err != nil {
		return fmt.Errorf("failed to set memory limit: %v", err)
	}

	// Set container's CPU limit (20%)
	// Format: "<max> <period>" where max and period are in microseconds
	cpuMaxFile := filepath.Join(c.cgroupPath, "cpu.max")
	if err := os.WriteFile(cpuMaxFile, []byte("20000 100000"), 0644); err != nil {
		return fmt.Errorf("failed to set CPU limit: %v", err)
	}

	// Add current process to the cgroup
	cgroupProcsFile := filepath.Join(c.cgroupPath, "cgroup.procs")
	if err := os.WriteFile(cgroupProcsFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %v", err)
	}

	return nil
}

// cleanupCgroup removes the custom cgroup created for the container process.
func (c *Container) cleanupCgroup() {
	rootProcs := filepath.Join(cgroupsRoot, "cgroup.procs")
	selfPid := []byte(strconv.Itoa(os.Getpid()))

	// Move the current process back to the root cgroup
	if err := os.WriteFile(rootProcs, selfPid, 0644); err != nil {
		fmt.Printf("Warning: Failed to move process out of cgroup: %v\n", err)
	}

	// Now it's safe to remove the cgroup directory
	if err := os.RemoveAll(c.cgroupPath); err != nil {
		fmt.Printf("Warning: Failed to remove cgroup directory: %v\n", err)
	}
}
