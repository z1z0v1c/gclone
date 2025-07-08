package image

// AuthResponse represents the token response from the Docker Registry auth API.
type AuthResponse struct {
	Token string `json:"token"`
}

// Manifest represents a platform-specific image manifest (schema v2).
// It includes metadata about the config blob and image layers.
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

// ManifestIndex represents a manifest list (multi-platform index).
// It maps platforms (e.g., linux/amd64) to specific manifest digests
type ManifestIndex struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// ImageConfig represents the full image configuration
type ImageConfig struct {
	Architecture string `json:"architecture"`
	Config       struct {
		Hostname     string              `json:"Hostname"`
		Domainname   string              `json:"Domainname"`
		User         string              `json:"User"`
		AttachStdin  bool                `json:"AttachStdin"`
		AttachStdout bool                `json:"AttachStdout"`
		AttachStderr bool                `json:"AttachStderr"`
		Tty          bool                `json:"Tty"`
		OpenStdin    bool                `json:"OpenStdin"`
		StdinOnce    bool                `json:"StdinOnce"`
		Env          []string            `json:"Env"`
		Cmd          []string            `json:"Cmd"`
		Image        string              `json:"Image"`
		Volumes      map[string]struct{} `json:"Volumes"`
		WorkingDir   string              `json:"WorkingDir"`
		Entrypoint   []string            `json:"Entrypoint"`
		OnBuild      []string            `json:"OnBuild"`
		Labels       map[string]string   `json:"Labels"`
	} `json:"config"`
	Container       string `json:"container"`
	ContainerConfig struct {
		Hostname     string              `json:"Hostname"`
		Domainname   string              `json:"Domainname"`
		User         string              `json:"User"`
		AttachStdin  bool                `json:"AttachStdin"`
		AttachStdout bool                `json:"AttachStdout"`
		AttachStderr bool                `json:"AttachStderr"`
		Tty          bool                `json:"Tty"`
		OpenStdin    bool                `json:"OpenStdin"`
		StdinOnce    bool                `json:"StdinOnce"`
		Env          []string            `json:"Env"`
		Cmd          []string            `json:"Cmd"`
		Image        string              `json:"Image"`
		Volumes      map[string]struct{} `json:"Volumes"`
		WorkingDir   string              `json:"WorkingDir"`
		Entrypoint   []string            `json:"Entrypoint"`
		OnBuild      []string            `json:"OnBuild"`
		Labels       map[string]string   `json:"Labels"`
	} `json:"container_config"`
	Created       string `json:"created"`
	DockerVersion string `json:"docker_version"`
	History       []struct {
		Created    string `json:"created"`
		CreatedBy  string `json:"created_by"`
		EmptyLayer bool   `json:"empty_layer,omitempty"`
	} `json:"history"`
	Os     string `json:"os"`
	Rootfs struct {
		Type    string   `json:"type"`
		DiffIds []string `json:"diff_ids"`
	} `json:"rootfs"`
}
