package podman

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/bindings/kube"
	"github.com/containers/podman/v5/pkg/bindings/pods"
)

type PodmanClient struct {
	Context context.Context
}

// NewPodmanClient creates and returns a new PodmanClient instance
func NewPodmanClient() (*PodmanClient, error) {
	ctx, err := bindings.NewConnectionWithIdentity(context.Background(), "ssh://root@127.0.0.1:51065/run/podman/podman.sock", "/Users/mayukac/.local/share/containers/podman/machine/machine", false)
	if err != nil {
		return nil, err
	}
	return &PodmanClient{Context: ctx}, nil
}

// Example function to list images (you can expand with more Podman functionalities)
func (pc *PodmanClient) ListImages() ([]string, error) {
	imagesList, err := images.List(pc.Context, nil)
	if err != nil {
		return nil, err
	}

	var imageNames []string
	for _, img := range imagesList {
		imageNames = append(imageNames, img.ID)
	}
	return imageNames, nil
}

func (pc *PodmanClient) ListPods() (any, error) {
	podList, err := pods.List(pc.Context, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	return podList, nil
}

func (pc *PodmanClient) CreatePodFromTemplate(filePath string, params map[string]any) error {
	tmplBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	var rendered bytes.Buffer

	tmpl, err := template.New("pod").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(&rendered, params); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// fmt.Println("Rendered YAML:\n", rendered.String())

	// Wrap the bytes in a bytes.Reader
	reader := bytes.NewReader(rendered.Bytes())

	_, err = kube.PlayWithBody(pc.Context, reader, nil)
	if err != nil {
		return fmt.Errorf("failed to execute podman kube play: %w", err)
	}

	return nil
}
