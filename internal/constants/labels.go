// Package constants provides centralized constant definitions for the autoscaler.
// This file contains Kubernetes label keys used for filtering and identification.
package constants

// Kubernetes Label Keys
// Label keys used on Kubernetes resources for filtering and identification.
const (
	// ControllerInstanceLabelKey is the label key used to associate VAs with specific controller instances.
	// Used for multi-controller isolation where each controller only manages VAs with matching labels.
	ControllerInstanceLabelKey = "wva.llmd.ai/controller-instance"

	// NamespaceConfigEnabledLabelKey is the label key used to opt-in namespaces for namespace-local ConfigMap overrides.
	// When a namespace has this label set to "true", the controller will watch for namespace-local ConfigMaps
	// even if no VariantAutoscaling resources exist in that namespace yet.
	// This enables creating namespace-local ConfigMaps before VAs are created, avoiding race conditions.
	NamespaceConfigEnabledLabelKey = "wva.llmd.ai/config-enabled"

	// ConfigLabelKey is the label key used to identify WVA-managed ConfigMaps.
	// Only ConfigMaps with this label set to ConfigLabelValue are cached by the controller,
	// preventing the controller-runtime cache from storing every ConfigMap in the cluster.
	// This label must be present on all WVA well-known ConfigMaps and any user-created
	// namespace-local ConfigMap overrides.
	ConfigLabelKey = "wva.llmd.ai/config"

	// ConfigLabelValue is the expected value for ConfigLabelKey.
	ConfigLabelValue = "true"
)

// Kubernetes Annotation Keys
// Annotation keys used on Kubernetes resources for metadata and exclusion.
const (
	// NamespaceExcludeAnnotationKey is the annotation key used to exclude namespaces from WVA management.
	// When a namespace has this annotation set to "true", the controller will not watch it
	// for namespace-local ConfigMaps or reconcile VariantAutoscaling resources in it,
	// even if the namespace has VAs or opt-in labels.
	// This provides explicit control to exclude namespaces from WVA management.
	NamespaceExcludeAnnotationKey = "wva.llmd.ai/exclude"
)
