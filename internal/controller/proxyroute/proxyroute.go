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

package proxyroute

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/crossplane/provider-caddy/apis/config/v1alpha1"
	caddyclient "github.com/crossplane/provider-caddy/internal/clients/caddy"
)

const (
	errNotProxyRoute = "managed resource is not a ProxyRoute custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Caddy client"
	errCreateRoute   = "cannot create proxy route"
	errUpdateRoute   = "cannot update proxy route"
	errDeleteRoute   = "cannot delete proxy route"
	errGetRoute      = "cannot get proxy route"
)

// SetupGated adds a controller that reconciles ProxyRoute managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	return Setup(mgr, o)
}

// Setup adds a controller that reconciles ProxyRoute managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ProxyRouteGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ProxyRouteGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:   mgr.GetClient(),
			logger: o.Logger,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ProxyRoute{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method is called.
type connector struct {
	kube   client.Client
	logger logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Getting the managed resource's ProviderConfig.
// 2. Getting the credentials specified by the ProviderConfig.
// 3. Using the credentials to form a client.
// Note: For Caddy, we use the endpoint specified directly in the ProxyRoute spec.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.ProxyRoute)
	if !ok {
		return nil, errors.New(errNotProxyRoute)
	}

	return &external{
		client: caddyclient.NewClient(cr.Spec.ForProvider.CaddyEndpoint),
		logger: c.logger,
	}, nil
}

// An external observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client *caddyclient.Client
	logger logging.Logger
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ProxyRoute)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProxyRoute)
	}

	// Get the external name (route ID) from the annotation
	routeID := meta.GetExternalName(cr)
	if routeID == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	serverName := "srv0"
	if cr.Spec.ForProvider.ServerName != nil {
		serverName = *cr.Spec.ForProvider.ServerName
	}

	route, err := e.client.GetProxyRoute(ctx, serverName, routeID)
	if err != nil {
		// If the route is not found, treat it as non-existent
		if strings.Contains(err.Error(), "not found") {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		// For other errors, return them
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRoute)
	}

	// Update the status with observed values
	cr.Status.AtProvider.RouteID = routeID

	// Get upstream status
	upstreams, err := e.client.GetUpstreamStatus(ctx)
	if err != nil {
		// Don't fail if we can't get upstream status
		e.logger.Info("Failed to get upstream status", "error", err)
	} else {
		cr.Status.AtProvider.UpstreamStatuses = convertUpstreamStatuses(upstreams)
	}

	// Determine if the resource is up to date
	upToDate := isUpToDate(cr, route)

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ProxyRoute)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProxyRoute)
	}

	cr.Status.SetConditions(xpv1.Creating())

	serverName := "srv0"
	if cr.Spec.ForProvider.ServerName != nil {
		serverName = *cr.Spec.ForProvider.ServerName
	}

	route := convertToProxyRoute(cr)

	routeID, err := e.client.CreateProxyRoute(ctx, serverName, route)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRoute)
	}

	// Set the external name annotation
	meta.SetExternalName(cr, routeID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ProxyRoute)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProxyRoute)
	}

	serverName := "srv0"
	if cr.Spec.ForProvider.ServerName != nil {
		serverName = *cr.Spec.ForProvider.ServerName
	}

	routeID := meta.GetExternalName(cr)
	route := convertToProxyRoute(cr)

	if err := e.client.UpdateProxyRoute(ctx, serverName, routeID, route); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRoute)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ProxyRoute)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProxyRoute)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	serverName := "srv0"
	if cr.Spec.ForProvider.ServerName != nil {
		serverName = *cr.Spec.ForProvider.ServerName
	}

	routeID := meta.GetExternalName(cr)
	if routeID == "" {
		// Nothing to delete
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteProxyRoute(ctx, serverName, routeID); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRoute)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is called when the controller is shutting down.
func (e *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect for HTTP client
	return nil
}

// convertToProxyRoute converts the CRD spec to the Caddy client format.
//
//nolint:gocyclo // Conversion function with linear complexity
func convertToProxyRoute(cr *v1alpha1.ProxyRoute) *caddyclient.ProxyRoute {
	route := &caddyclient.ProxyRoute{
		Terminal: true,
	}

	// Convert match conditions (Caddy expects array of matcher sets)
	if cr.Spec.ForProvider.Match != nil {
		route.Match = []caddyclient.MatchSet{
			{
				Host:   cr.Spec.ForProvider.Match.Host,
				Path:   cr.Spec.ForProvider.Match.Path,
				Method: cr.Spec.ForProvider.Match.Method,
				Header: cr.Spec.ForProvider.Match.Headers,
			},
		}
	}

	// Create the reverse_proxy handler
	handler := caddyclient.Handler{
		Handler: "reverse_proxy",
	}

	// Convert upstreams
	for _, upstream := range cr.Spec.ForProvider.Upstreams {
		u := caddyclient.Upstream{
			Dial: upstream.Dial,
		}
		if upstream.MaxRequests != nil {
			u.MaxRequests = *upstream.MaxRequests
		}
		handler.Upstreams = append(handler.Upstreams, u)
	}

	// Convert load balancing
	if cr.Spec.ForProvider.LoadBalancing != nil {
		handler.LoadBalancing = &caddyclient.LoadBalancing{}
		if cr.Spec.ForProvider.LoadBalancing.Policy != nil {
			handler.LoadBalancing.SelectionPolicy = &caddyclient.SelectionPolicy{
				Policy: *cr.Spec.ForProvider.LoadBalancing.Policy,
			}
		}
		if cr.Spec.ForProvider.LoadBalancing.TryDuration != nil {
			handler.LoadBalancing.TryDuration = *cr.Spec.ForProvider.LoadBalancing.TryDuration
		}
		if cr.Spec.ForProvider.LoadBalancing.TryInterval != nil {
			handler.LoadBalancing.TryInterval = *cr.Spec.ForProvider.LoadBalancing.TryInterval
		}
	}

	// Convert headers
	if cr.Spec.ForProvider.Headers != nil {
		handler.Headers = &caddyclient.Headers{}
		if cr.Spec.ForProvider.Headers.Request != nil {
			handler.Headers.Request = &caddyclient.HeaderOps{
				Set:    cr.Spec.ForProvider.Headers.Request.Set,
				Add:    cr.Spec.ForProvider.Headers.Request.Add,
				Delete: cr.Spec.ForProvider.Headers.Request.Delete,
			}
		}
		if cr.Spec.ForProvider.Headers.Response != nil {
			handler.Headers.Response = &caddyclient.HeaderOps{
				Set:    cr.Spec.ForProvider.Headers.Response.Set,
				Add:    cr.Spec.ForProvider.Headers.Response.Add,
				Delete: cr.Spec.ForProvider.Headers.Response.Delete,
			}
		}
	}

	// Convert health checks
	if cr.Spec.ForProvider.HealthChecks != nil {
		handler.HealthChecks = &caddyclient.HealthChecks{}
		if cr.Spec.ForProvider.HealthChecks.Active != nil {
			handler.HealthChecks.Active = &caddyclient.ActiveHealthCheck{}
			if cr.Spec.ForProvider.HealthChecks.Active.Path != nil {
				handler.HealthChecks.Active.Path = *cr.Spec.ForProvider.HealthChecks.Active.Path
			}
			if cr.Spec.ForProvider.HealthChecks.Active.Interval != nil {
				handler.HealthChecks.Active.Interval = *cr.Spec.ForProvider.HealthChecks.Active.Interval
			}
			if cr.Spec.ForProvider.HealthChecks.Active.Timeout != nil {
				handler.HealthChecks.Active.Timeout = *cr.Spec.ForProvider.HealthChecks.Active.Timeout
			}
		}
		if cr.Spec.ForProvider.HealthChecks.Passive != nil {
			handler.HealthChecks.Passive = &caddyclient.PassiveHealthCheck{}
			if cr.Spec.ForProvider.HealthChecks.Passive.MaxFails != nil {
				handler.HealthChecks.Passive.MaxFails = *cr.Spec.ForProvider.HealthChecks.Passive.MaxFails
			}
			if cr.Spec.ForProvider.HealthChecks.Passive.UnhealthyLatency != nil {
				handler.HealthChecks.Passive.UnhealthyLatency = *cr.Spec.ForProvider.HealthChecks.Passive.UnhealthyLatency
			}
		}
	}

	// Convert TLS
	if cr.Spec.ForProvider.TLS != nil && cr.Spec.ForProvider.TLS.Enabled != nil && *cr.Spec.ForProvider.TLS.Enabled {
		handler.Transport = &caddyclient.Transport{
			Protocol: "http",
			TLS:      &caddyclient.TLSConfig{},
		}
		if cr.Spec.ForProvider.TLS.ServerName != nil {
			handler.Transport.TLS.ServerName = *cr.Spec.ForProvider.TLS.ServerName
		}
		if cr.Spec.ForProvider.TLS.InsecureSkipVerify != nil {
			handler.Transport.TLS.InsecureSkipVerify = *cr.Spec.ForProvider.TLS.InsecureSkipVerify
		}
	}

	route.Handle = []caddyclient.Handler{handler}

	return route
}

// isUpToDate checks if the Caddy route matches the desired state.
func isUpToDate(cr *v1alpha1.ProxyRoute, route *caddyclient.ProxyRoute) bool {
	// For simplicity, we'll consider the resource always needs an update
	// In production, you'd implement a detailed comparison
	// This encourages periodic reconciliation which is safer for external systems
	return false
}

// convertUpstreamStatuses converts Caddy client upstream statuses to CRD format.
func convertUpstreamStatuses(upstreams []caddyclient.UpstreamStatus) []v1alpha1.UpstreamStatus {
	statuses := make([]v1alpha1.UpstreamStatus, len(upstreams))
	for i, u := range upstreams {
		statuses[i] = v1alpha1.UpstreamStatus{
			Address:     u.Address,
			Healthy:     u.Healthy,
			NumRequests: u.NumRequests,
		}
	}
	return statuses
}
