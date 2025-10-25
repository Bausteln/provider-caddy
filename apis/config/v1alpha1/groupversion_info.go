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

// Package v1alpha1 contains the Caddy configuration API v1alpha1 resources.
// +kubebuilder:object:generate=true
// +groupName=config.caddy.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "config.caddy.crossplane.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// ProxyRoute type metadata.
var (
	ProxyRouteKind             = reflect.TypeOf(ProxyRoute{}).Name()
	ProxyRouteGroupKind        = schema.GroupKind{Group: Group, Kind: ProxyRouteKind}.String()
	ProxyRouteGroupVersionKind = SchemeGroupVersion.WithKind(ProxyRouteKind)

	ProxyRouteListKind             = reflect.TypeOf(ProxyRouteList{}).Name()
	ProxyRouteListGroupVersionKind = SchemeGroupVersion.WithKind(ProxyRouteListKind)
)

func init() {
	SchemeBuilder.Register(&ProxyRoute{}, &ProxyRouteList{})
}
