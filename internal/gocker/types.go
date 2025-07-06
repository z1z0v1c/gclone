package gocker

type AuthResponse struct {
	Token string `json:"token"`
}

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
