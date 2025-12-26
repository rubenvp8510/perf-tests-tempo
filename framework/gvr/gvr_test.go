package gvr

import (
	"testing"
)

func TestTempoGVRs(t *testing.T) {
	if TempoMonolithic.Group != "tempo.grafana.com" {
		t.Errorf("expected Group 'tempo.grafana.com', got %q", TempoMonolithic.Group)
	}
	if TempoMonolithic.Version != "v1alpha1" {
		t.Errorf("expected Version 'v1alpha1', got %q", TempoMonolithic.Version)
	}
	if TempoMonolithic.Resource != "tempomonolithics" {
		t.Errorf("expected Resource 'tempomonolithics', got %q", TempoMonolithic.Resource)
	}

	if TempoStack.Resource != "tempostacks" {
		t.Errorf("expected Resource 'tempostacks', got %q", TempoStack.Resource)
	}
}

func TestOpenTelemetryGVR(t *testing.T) {
	if OpenTelemetryCollector.Group != "opentelemetry.io" {
		t.Errorf("expected Group 'opentelemetry.io', got %q", OpenTelemetryCollector.Group)
	}
	if OpenTelemetryCollector.Version != "v1beta1" {
		t.Errorf("expected Version 'v1beta1', got %q", OpenTelemetryCollector.Version)
	}
	if OpenTelemetryCollector.Resource != "opentelemetrycollectors" {
		t.Errorf("expected Resource 'opentelemetrycollectors', got %q", OpenTelemetryCollector.Resource)
	}
}

func TestRBACGVRs(t *testing.T) {
	if ClusterRole.Group != "rbac.authorization.k8s.io" {
		t.Errorf("expected Group 'rbac.authorization.k8s.io', got %q", ClusterRole.Group)
	}
	if ClusterRole.Version != "v1" {
		t.Errorf("expected Version 'v1', got %q", ClusterRole.Version)
	}
	if ClusterRole.Resource != "clusterroles" {
		t.Errorf("expected Resource 'clusterroles', got %q", ClusterRole.Resource)
	}

	if ClusterRoleBinding.Resource != "clusterrolebindings" {
		t.Errorf("expected Resource 'clusterrolebindings', got %q", ClusterRoleBinding.Resource)
	}
}

func TestCoreGVRs(t *testing.T) {
	// Core resources have empty Group
	if Namespace.Group != "" {
		t.Errorf("expected empty Group for Namespace, got %q", Namespace.Group)
	}
	if Namespace.Version != "v1" {
		t.Errorf("expected Version 'v1', got %q", Namespace.Version)
	}
	if Namespace.Resource != "namespaces" {
		t.Errorf("expected Resource 'namespaces', got %q", Namespace.Resource)
	}

	if Pod.Resource != "pods" {
		t.Errorf("expected Resource 'pods', got %q", Pod.Resource)
	}
	if Service.Resource != "services" {
		t.Errorf("expected Resource 'services', got %q", Service.Resource)
	}
}

func TestAppsGVRs(t *testing.T) {
	if Deployment.Group != "apps" {
		t.Errorf("expected Group 'apps', got %q", Deployment.Group)
	}
	if Deployment.Resource != "deployments" {
		t.Errorf("expected Resource 'deployments', got %q", Deployment.Resource)
	}

	if StatefulSet.Resource != "statefulsets" {
		t.Errorf("expected Resource 'statefulsets', got %q", StatefulSet.Resource)
	}
}

func TestRouteGVR(t *testing.T) {
	if Route.Group != "route.openshift.io" {
		t.Errorf("expected Group 'route.openshift.io', got %q", Route.Group)
	}
	if Route.Resource != "routes" {
		t.Errorf("expected Resource 'routes', got %q", Route.Resource)
	}
}

func TestCRDConstants(t *testing.T) {
	if TempoMonolithicCRD != "tempomonolithics.tempo.grafana.com" {
		t.Errorf("expected TempoMonolithicCRD 'tempomonolithics.tempo.grafana.com', got %q", TempoMonolithicCRD)
	}
	if TempoStackCRD != "tempostacks.tempo.grafana.com" {
		t.Errorf("expected TempoStackCRD 'tempostacks.tempo.grafana.com', got %q", TempoStackCRD)
	}
	if OpenTelemetryCollectorCRD != "opentelemetrycollectors.opentelemetry.io" {
		t.Errorf("expected OpenTelemetryCollectorCRD 'opentelemetrycollectors.opentelemetry.io', got %q", OpenTelemetryCollectorCRD)
	}
}

func TestAllTempoCRs(t *testing.T) {
	crs := AllTempoCRs()
	if len(crs) != 2 {
		t.Errorf("expected 2 Tempo CRs, got %d", len(crs))
	}

	found := make(map[string]bool)
	for _, cr := range crs {
		found[cr.Resource] = true
	}

	if !found["tempomonolithics"] {
		t.Error("expected tempomonolithics in AllTempoCRs")
	}
	if !found["tempostacks"] {
		t.Error("expected tempostacks in AllTempoCRs")
	}
}

func TestAllManagedCRs(t *testing.T) {
	crs := AllManagedCRs()
	if len(crs) != 3 {
		t.Errorf("expected 3 managed CRs, got %d", len(crs))
	}

	found := make(map[string]bool)
	for _, cr := range crs {
		found[cr.Resource] = true
	}

	if !found["tempomonolithics"] {
		t.Error("expected tempomonolithics in AllManagedCRs")
	}
	if !found["tempostacks"] {
		t.Error("expected tempostacks in AllManagedCRs")
	}
	if !found["opentelemetrycollectors"] {
		t.Error("expected opentelemetrycollectors in AllManagedCRs")
	}
}

func TestAllClusterScopedResources(t *testing.T) {
	resources := AllClusterScopedResources()
	if len(resources) != 3 {
		t.Errorf("expected 3 cluster-scoped resources, got %d", len(resources))
	}

	found := make(map[string]bool)
	for _, r := range resources {
		found[r.Resource] = true
	}

	if !found["clusterroles"] {
		t.Error("expected clusterroles in AllClusterScopedResources")
	}
	if !found["clusterrolebindings"] {
		t.Error("expected clusterrolebindings in AllClusterScopedResources")
	}
	if !found["persistentvolumes"] {
		t.Error("expected persistentvolumes in AllClusterScopedResources")
	}
}
