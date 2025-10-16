package runtime

type Runtime interface {
	ListImages() ([]string, error)
	ListPods() (any, error)
	CreatePodFromTemplate(filePath string, params map[string]any) error
}
