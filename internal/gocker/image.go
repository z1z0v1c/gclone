package gocker

import "path/filepath"

const (
	registry    = "registry-1.docker.io"
	tag         = "latest"
	authBaseURL = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
)

type Image struct {
	Name       string
	Repository string
	Token      string

	Manifest *Manifest
	Config   *ImageConfig
}

func NewImage(name string) *Image {
	return &Image{
		Name: name,
		Repository: filepath.Join("library", name),
	}
}