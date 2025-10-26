/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package caddy provides a client for the Caddy admin API.
package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is a client for the Caddy admin API.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new Caddy API client.
func NewClient(endpoint string) *Client {
	return &Client{
		endpoint:   strings.TrimSuffix(endpoint, "/"),
		httpClient: &http.Client{},
	}
}

// ProxyRoute represents a Caddy reverse proxy route configuration.
type ProxyRoute struct {
	Match    []MatchSet `json:"match,omitempty"`
	Handle   []Handler  `json:"handle"`
	Terminal bool       `json:"terminal,omitempty"`
}

// MatchSet represents a set of matchers (Caddy uses array of matcher sets).
type MatchSet struct {
	Host   []string            `json:"host,omitempty"`
	Path   []string            `json:"path,omitempty"`
	Method []string            `json:"method,omitempty"`
	Header map[string][]string `json:"header,omitempty"`
}

// Handler represents a route handler.
type Handler struct {
	Handler       string         `json:"handler"`
	Routes        []Route        `json:"routes,omitempty"`
	Upstreams     []Upstream     `json:"upstreams,omitempty"`
	LoadBalancing *LoadBalancing `json:"load_balancing,omitempty"`
	Headers       *Headers       `json:"headers,omitempty"`
	HealthChecks  *HealthChecks  `json:"health_checks,omitempty"`
	Transport     *Transport     `json:"transport,omitempty"`
}

// Route represents a subroute.
type Route struct {
	Handle []Handler `json:"handle"`
}

// Upstream represents a backend server.
type Upstream struct {
	Dial        string `json:"dial"`
	MaxRequests int    `json:"max_requests,omitempty"`
}

// LoadBalancing represents load balancing configuration.
type LoadBalancing struct {
	SelectionPolicy *SelectionPolicy `json:"selection_policy,omitempty"`
	TryDuration     string           `json:"try_duration,omitempty"`
	TryInterval     string           `json:"try_interval,omitempty"`
}

// SelectionPolicy represents a load balancing selection policy.
type SelectionPolicy struct {
	Policy string `json:"policy,omitempty"`
}

// Headers represents header manipulation configuration.
type Headers struct {
	Request  *HeaderOps `json:"request,omitempty"`
	Response *HeaderOps `json:"response,omitempty"`
}

// HeaderOps represents header operations.
type HeaderOps struct {
	Set    map[string][]string `json:"set,omitempty"`
	Add    map[string][]string `json:"add,omitempty"`
	Delete []string            `json:"delete,omitempty"`
}

// HealthChecks represents health check configuration.
type HealthChecks struct {
	Active  *ActiveHealthCheck  `json:"active,omitempty"`
	Passive *PassiveHealthCheck `json:"passive,omitempty"`
}

// ActiveHealthCheck represents active health check configuration.
type ActiveHealthCheck struct {
	Path     string `json:"path,omitempty"`
	Interval string `json:"interval,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
}

// PassiveHealthCheck represents passive health check configuration.
type PassiveHealthCheck struct {
	MaxFails         int    `json:"max_fails,omitempty"`
	UnhealthyLatency string `json:"unhealthy_latency,omitempty"`
}

// Transport represents upstream transport configuration.
type Transport struct {
	Protocol string     `json:"protocol"`
	TLS      *TLSConfig `json:"tls,omitempty"`
}

// TLSConfig represents TLS configuration for upstream connections.
type TLSConfig struct {
	ServerName         string `json:"server_name,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
}

// UpstreamStatus represents the health status of an upstream.
type UpstreamStatus struct {
	Address     string `json:"address"`
	Healthy     bool   `json:"healthy"`
	NumRequests int    `json:"num_requests"`
}

// CreateProxyRoute creates a new proxy route in Caddy.
func (c *Client) CreateProxyRoute(ctx context.Context, serverName string, route *ProxyRoute) (string, error) {
	path := fmt.Sprintf("/config/apps/http/servers/%s/routes", serverName)

	body, err := json.Marshal(route)
	if err != nil {
		return "", fmt.Errorf("failed to marshal route: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create route: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Caddy doesn't return a route ID, so we'll use the host+path as identifier
	// In a production implementation, you might want to use @id directives
	routeID := generateRouteID(route)
	return routeID, nil
}

// UpdateProxyRoute updates an existing proxy route in Caddy.
func (c *Client) UpdateProxyRoute(ctx context.Context, serverName, routeID string, route *ProxyRoute) error {
	// For simplicity, we'll delete and recreate the route
	// In production, you'd want to use Caddy's PATCH endpoint or @id directives
	if err := c.DeleteProxyRoute(ctx, serverName, routeID); err != nil {
		return fmt.Errorf("failed to delete old route: %w", err)
	}

	_, err := c.CreateProxyRoute(ctx, serverName, route)
	if err != nil {
		return fmt.Errorf("failed to create updated route: %w", err)
	}

	return nil
}

// DeleteProxyRoute deletes a proxy route from Caddy.
//
//nolint:gocyclo // Complex due to multi-step deletion process
func (c *Client) DeleteProxyRoute(ctx context.Context, serverName, routeID string) error {
	// This is a simplified implementation
	// In production, you'd use Caddy's array removal or @id directives
	path := fmt.Sprintf("/config/apps/http/servers/%s/routes", serverName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get routes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// Route already doesn't exist
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var routes []ProxyRoute
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return fmt.Errorf("failed to decode routes: %w", err)
	}

	// Find and remove the route with matching ID
	index := -1
	for i, r := range routes {
		if generateRouteID(&r) == routeID {
			index = i
			break
		}
	}

	if index == -1 {
		// Route not found, consider it already deleted
		return nil
	}

	// Delete the route at the found index
	deletePath := fmt.Sprintf("/config/apps/http/servers/%s/routes/%d", serverName, index)
	req, err = http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint+deletePath, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetProxyRoute retrieves a proxy route from Caddy.
func (c *Client) GetProxyRoute(ctx context.Context, serverName, routeID string) (*ProxyRoute, error) {
	path := fmt.Sprintf("/config/apps/http/servers/%s/routes", serverName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("route not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var routes []ProxyRoute
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("failed to decode routes: %w", err)
	}

	// Find the route with matching ID
	for _, r := range routes {
		if generateRouteID(&r) == routeID {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("route not found")
}

// GetUpstreamStatus retrieves the health status of upstreams.
func (c *Client) GetUpstreamStatus(ctx context.Context) ([]UpstreamStatus, error) {
	path := "/reverse_proxy/upstreams"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstream status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var upstreams []UpstreamStatus
	if err := json.NewDecoder(resp.Body).Decode(&upstreams); err != nil {
		return nil, fmt.Errorf("failed to decode upstreams: %w", err)
	}

	return upstreams, nil
}

// generateRouteID generates a unique ID for a route based on its configuration.
// This is a simplified implementation - in production, you'd use Caddy's @id directive.
func generateRouteID(route *ProxyRoute) string {
	if len(route.Match) == 0 {
		return "default"
	}

	// Use first matcher set for ID generation
	matchSet := route.Match[0]

	var parts []string
	if len(matchSet.Host) > 0 {
		parts = append(parts, "host:"+strings.Join(matchSet.Host, ","))
	}
	if len(matchSet.Path) > 0 {
		parts = append(parts, "path:"+strings.Join(matchSet.Path, ","))
	}
	if len(matchSet.Method) > 0 {
		parts = append(parts, "method:"+strings.Join(matchSet.Method, ","))
	}

	if len(parts) == 0 {
		return "default"
	}

	return strings.Join(parts, "|")
}
