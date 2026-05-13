package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/project-ai-services/ai-services/internal/pkg/proxy"
	"github.com/project-ai-services/ai-services/internal/pkg/runtime"
)

const (
	maxRetries  = 10
	retryDelay  = 2 * time.Second
	httpTimeout = 5 * time.Second
)

// Manager implements the ProxyManager interface for Caddy server.
type Manager struct {
	adminURL string
	server   string
	client   *http.Client
}

// NewManagerWithConfig creates a new Caddy proxy manager with custom configuration.
func NewManagerWithConfig(adminURL, serverName string) *Manager {
	return &Manager{
		adminURL: adminURL,
		server:   serverName,
		client:   &http.Client{Timeout: httpTimeout},
	}
}

// readResponseBody reads and returns the response body as a string.
func readResponseBody(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("(failed to read response body: %v)", err)
	}
	return string(body)
}

// RegisterRoute registers a new route with Caddy by appending to the routes array.
func (m *Manager) RegisterRoute(route proxy.Route) error {
	// Build route object
	caddyRoute := map[string]interface{}{
		"@id": route.ID,
		"match": []map[string]interface{}{
			{
				"host": []string{route.Domain},
			},
		},
		"handle": []map[string]interface{}{
			{
				"handler": "reverse_proxy",
				"upstreams": []map[string]string{
					{
						"dial": route.Upstream,
					},
				},
			},
		},
		"terminal": route.Terminal,
	}

	data, err := json.Marshal(caddyRoute)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	// Check if routes array exists, initialize if needed
	checkURL := fmt.Sprintf("%s/config/apps/http/servers/%s/routes", m.adminURL, m.server)
	checkResp, err := m.client.Get(checkURL)
	if err != nil {
		return fmt.Errorf("failed to check routes: %w", err)
	}
	defer checkResp.Body.Close()

	body, _ := io.ReadAll(checkResp.Body)
	if string(body) == "null\n" || string(body) == "null" {
		// Initialize empty routes array
		initReq, _ := http.NewRequest(http.MethodPost, checkURL, bytes.NewBufferString("[]"))
		initReq.Header.Set("Content-Type", "application/json")
		initResp, err := m.client.Do(initReq)
		if err != nil {
			return fmt.Errorf("failed to initialize routes: %w", err)
		}
		initResp.Body.Close()
	}

	// Append route to array using /-
	appendURL := fmt.Sprintf("%s/config/apps/http/servers/%s/routes/-", m.adminURL, m.server)
	req, err := http.NewRequest(http.MethodPost, appendURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody := readResponseBody(resp)
		return fmt.Errorf("caddy returned error (status %d): %s", resp.StatusCode, respBody)
	}

	return nil
}

// HealthCheck verifies Caddy Admin API is available and responding.
func (m *Manager) HealthCheck() error {
	for i := 0; i < maxRetries; i++ {
		resp, err := m.client.Get(fmt.Sprintf("%s/config/", m.adminURL))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("caddy admin API not available after %d retries", maxRetries)
}

// GetAdminPort retrieves the host port mapped to Caddy's admin API (container port 2019).
// This is a helper function that can be used by any deployment (catalog, application, service).
func GetAdminPort(rt runtime.Runtime, appName string) (string, error) {
	caddyPodName := fmt.Sprintf("%s--caddy", appName)
	pod, err := rt.InspectPod(caddyPodName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect Caddy pod: %w", err)
	}

	// Get port mappings from the Ports field
	// Ports is a map[string][]string where key is "containerPort/protocol" and value is list of host ports
	// Example: {"2019/tcp": ["37249"], "443/tcp": ["39341"]}
	for containerPort, hostPorts := range pod.Ports {
		// Check if this is the admin API port (2019)
		if strings.HasPrefix(containerPort, "2019/") && len(hostPorts) > 0 {
			return hostPorts[0], nil
		}
	}

	return "", fmt.Errorf("admin port (2019) mapping not found in pod ports")
}

// Ensure Manager implements ProxyManager interface
var _ proxy.ProxyManager = (*Manager)(nil)

// Made with Bob
