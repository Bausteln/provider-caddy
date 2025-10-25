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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ProxyRouteParameters define the desired state of a Caddy reverse proxy route.
type ProxyRouteParameters struct {
	// CaddyEndpoint is the Caddy admin API endpoint (e.g., "http://localhost:2019")
	// +kubebuilder:validation:Required
	CaddyEndpoint string `json:"caddyEndpoint"`

	// ServerName is the name of the Caddy server to add this route to.
	// If not specified, defaults to "srv0".
	// +optional
	ServerName *string `json:"serverName,omitempty"`

	// Match defines the conditions to match for this route.
	// +optional
	Match *RouteMatch `json:"match,omitempty"`

	// Upstreams defines the backend servers to proxy to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Upstreams []Upstream `json:"upstreams"`

	// LoadBalancing defines the load balancing policy.
	// +optional
	LoadBalancing *LoadBalancing `json:"loadBalancing,omitempty"`

	// Headers allows manipulation of request and response headers.
	// +optional
	Headers *HeaderOps `json:"headers,omitempty"`

	// HealthChecks defines active and passive health checks for upstreams.
	// +optional
	HealthChecks *HealthChecks `json:"healthChecks,omitempty"`

	// TLS defines TLS settings for upstream connections.
	// +optional
	TLS *UpstreamTLS `json:"tls,omitempty"`
}

// RouteMatch defines the matching conditions for a route.
type RouteMatch struct {
	// Host matches the request host (domain names).
	// +optional
	Host []string `json:"host,omitempty"`

	// Path matches the request path.
	// Supports wildcards like "/api/*"
	// +optional
	Path []string `json:"path,omitempty"`

	// Method matches the HTTP method.
	// +optional
	Method []string `json:"method,omitempty"`

	// Headers matches request headers.
	// +optional
	Headers map[string][]string `json:"headers,omitempty"`
}

// Upstream represents a backend server.
type Upstream struct {
	// Dial is the address to dial to connect to the upstream.
	// Format: "host:port" or just "host" (defaults to port 80/443)
	// +kubebuilder:validation:Required
	Dial string `json:"dial"`

	// MaxRequests is the maximum number of concurrent requests to this upstream.
	// +optional
	MaxRequests *int `json:"maxRequests,omitempty"`
}

// LoadBalancing defines load balancing configuration.
type LoadBalancing struct {
	// Policy is the load balancing policy to use.
	// Options: "random", "round_robin", "least_conn", "ip_hash", "header", "cookie"
	// +kubebuilder:validation:Enum=random;round_robin;least_conn;ip_hash;header;cookie
	// +optional
	Policy *string `json:"policy,omitempty"`

	// TryDuration is how long to try selecting available backends.
	// +optional
	TryDuration *string `json:"tryDuration,omitempty"`

	// TryInterval is how long to wait between retries.
	// +optional
	TryInterval *string `json:"tryInterval,omitempty"`
}

// HeaderOps defines header manipulation operations.
type HeaderOps struct {
	// Request defines operations on request headers.
	// +optional
	Request *HeaderManipulation `json:"request,omitempty"`

	// Response defines operations on response headers.
	// +optional
	Response *HeaderManipulation `json:"response,omitempty"`
}

// HeaderManipulation defines header operations.
type HeaderManipulation struct {
	// Set sets header values, replacing existing ones.
	// +optional
	Set map[string][]string `json:"set,omitempty"`

	// Add adds header values.
	// +optional
	Add map[string][]string `json:"add,omitempty"`

	// Delete removes headers.
	// +optional
	Delete []string `json:"delete,omitempty"`
}

// HealthChecks defines health check configuration.
type HealthChecks struct {
	// Active defines active health checks.
	// +optional
	Active *ActiveHealthCheck `json:"active,omitempty"`

	// Passive defines passive health checks.
	// +optional
	Passive *PassiveHealthCheck `json:"passive,omitempty"`
}

// ActiveHealthCheck defines active health check configuration.
type ActiveHealthCheck struct {
	// Path is the URI path to use for health checks.
	// +optional
	Path *string `json:"path,omitempty"`

	// Interval is how often to perform active health checks.
	// +optional
	Interval *string `json:"interval,omitempty"`

	// Timeout is how long to wait for a response.
	// +optional
	Timeout *string `json:"timeout,omitempty"`
}

// PassiveHealthCheck defines passive health check configuration.
type PassiveHealthCheck struct {
	// MaxFails is the maximum number of failed requests before marking unhealthy.
	// +optional
	MaxFails *int `json:"maxFails,omitempty"`

	// UnhealthyLatency is the latency threshold to consider unhealthy.
	// +optional
	UnhealthyLatency *string `json:"unhealthyLatency,omitempty"`
}

// UpstreamTLS defines TLS settings for upstream connections.
type UpstreamTLS struct {
	// Enabled enables TLS for upstream connections.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// ServerName is the server name for TLS verification.
	// +optional
	ServerName *string `json:"serverName,omitempty"`

	// InsecureSkipVerify disables TLS certificate verification.
	// +optional
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`
}

// ProxyRouteObservation represents the observed state of a ProxyRoute.
type ProxyRouteObservation struct {
	// RouteID is the ID assigned by Caddy to this route.
	// +optional
	RouteID string `json:"routeId,omitempty"`

	// UpstreamStatuses contains the health status of upstreams.
	// +optional
	UpstreamStatuses []UpstreamStatus `json:"upstreamStatuses,omitempty"`
}

// UpstreamStatus represents the health status of an upstream.
type UpstreamStatus struct {
	// Address is the upstream address.
	Address string `json:"address"`

	// Healthy indicates if the upstream is healthy.
	Healthy bool `json:"healthy"`

	// NumRequests is the number of active requests to this upstream.
	NumRequests int `json:"numRequests"`
}

// A ProxyRouteSpec defines the desired state of a ProxyRoute.
type ProxyRouteSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ProxyRouteParameters `json:"forProvider"`
}

// A ProxyRouteStatus represents the observed state of a ProxyRoute.
type ProxyRouteStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProxyRouteObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ProxyRoute configures a Caddy reverse proxy route.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,caddy}
type ProxyRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProxyRouteSpec   `json:"spec"`
	Status ProxyRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProxyRouteList contains a list of ProxyRoute
type ProxyRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyRoute `json:"items"`
}

// ProxyRoute type metadata.
var (
	ProxyRouteKindAPIVersion = ProxyRouteKind + "." + SchemeGroupVersion.String()
)

// GetCondition of this ProxyRoute.
func (mg *ProxyRoute) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this ProxyRoute.
func (mg *ProxyRoute) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// GetManagementPolicies of this ProxyRoute.
func (mg *ProxyRoute) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this ProxyRoute.
func (mg *ProxyRoute) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this ProxyRoute.
func (mg *ProxyRoute) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this ProxyRoute.
func (mg *ProxyRoute) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this ProxyRoute.
func (mg *ProxyRoute) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// SetManagementPolicies of this ProxyRoute.
func (mg *ProxyRoute) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this ProxyRoute.
func (mg *ProxyRoute) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this ProxyRoute.
func (mg *ProxyRoute) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
