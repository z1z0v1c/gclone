package gocker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const relativeImagesPath = ".local/share/gocker/images/"

// Container encapsulates container execution parameters
type Container struct {
	ImgName string
	ImgRoot string
	Cmd     string
	Args    []string
	Cfg     ContainerConfig
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

	container := &Container{
		ImgName: img,
		ImgRoot: imgRoot,
		Cmd:     cmd,
		Args:    args,
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
