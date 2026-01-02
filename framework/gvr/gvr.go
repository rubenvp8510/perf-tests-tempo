package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Tempo custom resources
var (
	// TempoMonolithic is the GVR for TempoMonolithic custom resources
	TempoMonolithic = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempomonolithics",
	}

	// TempoStack is the GVR for TempoStack custom resources
	TempoStack = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempostacks",
	}
)

// OpenTelemetry custom resources
var (
	// OpenTelemetryCollector is the GVR for OpenTelemetryCollector custom resources
	OpenTelemetryCollector = schema.GroupVersionResource{
		Group:    "opentelemetry.io",
		Version:  "v1beta1",
		Resource: "opentelemetrycollectors",
	}
)

// RBAC resources
var (
	// ClusterRole is the GVR for ClusterRole resources
	ClusterRole = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterroles",
	}

	// ClusterRoleBinding is the GVR for ClusterRoleBinding resources
	ClusterRoleBinding = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterrolebindings",
	}

	// Role is the GVR for Role resources
	Role = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "roles",
	}

	// RoleBinding is the GVR for RoleBinding resources
	RoleBinding = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "rolebindings",
	}
)

// Core resources
var (
	// Namespace is the GVR for Namespace resources
	Namespace = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}

	// PersistentVolume is the GVR for PersistentVolume resources
	PersistentVolume = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumes",
	}

	// PersistentVolumeClaim is the GVR for PersistentVolumeClaim resources
	PersistentVolumeClaim = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumeclaims",
	}

	// Secret is the GVR for Secret resources
	Secret = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}

	// ConfigMap is the GVR for ConfigMap resources
	ConfigMap = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	// Pod is the GVR for Pod resources
	Pod = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}

	// Service is the GVR for Service resources
	Service = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	}
)

// Apps resources
var (
	// Deployment is the GVR for Deployment resources
	Deployment = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	// StatefulSet is the GVR for StatefulSet resources
	StatefulSet = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	}
)

// Batch resources
var (
	// Job is the GVR for Job resources
	Job = schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}
)

// OpenShift Route resources
var (
	// Route is the GVR for OpenShift Route resources
	Route = schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}
)

// API Extensions
var (
	// CustomResourceDefinition is the GVR for CRD resources
	CustomResourceDefinition = schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
)

// Monitoring resources
var (
	// ServiceMonitor is the GVR for Prometheus ServiceMonitor resources
	ServiceMonitor = schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	}

	// PodMonitor is the GVR for Prometheus PodMonitor resources
	PodMonitor = schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "podmonitors",
	}
)

// CRD names for prerequisite checks
const (
	// TempoMonolithicCRD is the full name of the TempoMonolithic CRD
	TempoMonolithicCRD = "tempomonolithics.tempo.grafana.com"

	// TempoStackCRD is the full name of the TempoStack CRD
	TempoStackCRD = "tempostacks.tempo.grafana.com"

	// OpenTelemetryCollectorCRD is the full name of the OpenTelemetryCollector CRD
	OpenTelemetryCollectorCRD = "opentelemetrycollectors.opentelemetry.io"
)

// AllTempoCRs returns all Tempo-related custom resource GVRs
func AllTempoCRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		TempoMonolithic,
		TempoStack,
	}
}

// AllManagedCRs returns all custom resource GVRs managed by the framework
func AllManagedCRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		TempoMonolithic,
		TempoStack,
		OpenTelemetryCollector,
	}
}

// AllClusterScopedResources returns all cluster-scoped resource GVRs
func AllClusterScopedResources() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		ClusterRole,
		ClusterRoleBinding,
		PersistentVolume,
	}
}
