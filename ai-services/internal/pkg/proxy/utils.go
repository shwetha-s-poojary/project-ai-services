package proxy

import (
	"fmt"
	"strings"

	"github.com/project-ai-services/ai-services/internal/pkg/cli/templates"
)

// BuildRoutesFromConfig builds routes by reading routes_file.yaml configuration.
// This approach uses static configuration to define routing without needing pod inspection.
func BuildRoutesFromConfig(tp templates.Template, appName, hostIP string) ([]Route, error) {
	// Load routes configuration
	routesConfig, err := tp.LoadRoutesFile(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to load routes config: %w", err)
	}

	var routes []Route

	// Process each pod in the configuration
	for _, podConfig := range routesConfig.Routes {
		podName := podConfig.Pod

		// Process each service in the pod
		for _, svc := range podConfig.Services {
			// Extract container port number (remove /tcp suffix)
			containerPort := strings.Split(svc.ContainerPort, "/")[0]

			// Build route
			route := Route{
				ID:       fmt.Sprintf("%s--%s", podName, svc.Subdomain),
				Domain:   fmt.Sprintf("%s.%s.nip.io", svc.Subdomain, hostIP),
				Upstream: fmt.Sprintf("%s:%s", podName, containerPort),
				Terminal: true,
			}
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// Made with Bob
