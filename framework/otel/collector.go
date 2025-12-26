package otel

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"
	"github.com/redhat/perf-tests-tempo/test/framework/wait"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// CollectorGVR is an alias for backward compatibility - use gvr.OpenTelemetryCollector directly instead
var CollectorGVR = gvr.OpenTelemetryCollector

// FrameworkOperations provides access to framework capabilities needed by otel
type FrameworkOperations interface {
	Client() kubernetes.Interface
	DynamicClient() dynamic.Interface
	Context() context.Context
	Namespace() string
	Logger() *slog.Logger
	TrackCR(gvr schema.GroupVersionResource, namespace, name string)
	TrackClusterResource(gvr schema.GroupVersionResource, name string)
	GetManagedLabels() map[string]string
}

// SetupCollector deploys OpenTelemetry Collector with RBAC
func SetupCollector(fw FrameworkOperations) error {
	// Deploy RBAC first
	if err := setupRBAC(fw); err != nil {
		return fmt.Errorf("failed to setup OTel Collector RBAC: %w", err)
	}

	// Deploy Collector CR
	if err := setupCollectorCR(fw); err != nil {
		return fmt.Errorf("failed to setup OTel Collector CR: %w", err)
	}

	// Wait for collector to be ready
	return waitForCollectorReady(fw, 300*time.Second)
}

// setupRBAC sets up RBAC resources for OTel Collector
func setupRBAC(fw FrameworkOperations) error {
	namespace := fw.Namespace()
	client := fw.Client()
	ctx := fw.Context()
	managedLabels := fw.GetManagedLabels()

	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-sa",
			Namespace: namespace,
			Labels:    managedLabels,
		},
	}
	_, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// Create Role
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-role",
			Namespace: namespace,
			Labels:    managedLabels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err = client.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Role: %w", err)
	}

	// Create RoleBinding
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-rolebinding",
			Namespace: namespace,
			Labels:    managedLabels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "otel-collector-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "otel-collector-sa",
				Namespace: namespace,
			},
		},
	}
	_, err = client.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create RoleBinding: %w", err)
	}

	// Generate unique names for cluster-scoped resources to avoid conflicts
	clusterRoleName := fmt.Sprintf("allow-write-traces-%s", namespace)
	clusterRoleBindingName := fmt.Sprintf("allow-write-traces-%s", namespace)

	// Create ClusterRole
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterRoleName,
			Labels: managedLabels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"tempo.grafana.com"},
				Resources:     []string{"tenant-1"},
				ResourceNames: []string{"traces"},
				Verbs:         []string{"create"},
			},
		},
	}
	_, err = client.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ClusterRole: %w", err)
	}
	// Track ClusterRole
	fw.TrackClusterResource(gvr.ClusterRole, clusterRoleName)

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterRoleBindingName,
			Labels: managedLabels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "otel-collector-sa",
				Namespace: namespace,
			},
		},
	}
	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}
	// Track ClusterRoleBinding
	fw.TrackClusterResource(gvr.ClusterRoleBinding, clusterRoleBindingName)

	return nil
}

// setupCollectorCR sets up the OpenTelemetryCollector CR
func setupCollectorCR(fw FrameworkOperations) error {
	namespace := fw.Namespace()

	// Build OpenTelemetryCollector CR programmatically
	collectorObj := buildCollectorCR(namespace)

	// Add managed labels
	labels := collectorObj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range fw.GetManagedLabels() {
		labels[k] = v
	}
	collectorObj.SetLabels(labels)

	_, err := fw.DynamicClient().Resource(CollectorGVR).Namespace(namespace).Create(fw.Context(), collectorObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create OpenTelemetryCollector: %w", err)
	}

	// Track the created resource
	fw.TrackCR(CollectorGVR, namespace, "otel-collector")

	return nil
}

// waitForCollectorReady waits for OpenTelemetry Collector to be ready
func waitForCollectorReady(fw FrameworkOperations, timeout time.Duration) error {
	namespace := fw.Namespace()
	client := fw.Client()
	ctx := fw.Context()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check for deployment
		for _, deploymentName := range []string{"otel-collector-collector", "otel-collector"} {
			deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err == nil {
				if deployment.Status.ReadyReplicas == deployment.Status.Replicas &&
					deployment.Status.ReadyReplicas > 0 {
					return nil
				}
			}
		}

		// Check for pods directly
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=opentelemetry-collector",
		})
		if err == nil {
			for _, pod := range pods.Items {
				if wait.IsPodReady(&pod) {
					return nil
				}
			}
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("otel collector not ready after %v", timeout)
}

// buildCollectorCR builds an OpenTelemetryCollector CR programmatically
func buildCollectorCR(namespace string) *unstructured.Unstructured {
	tempoGatewayHost := fmt.Sprintf("tempo-simplest-gateway.%s.svc.cluster.local", namespace)

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "opentelemetry.io/v1beta1",
			"kind":       "OpenTelemetryCollector",
			"metadata": map[string]interface{}{
				"name":      "otel-collector",
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"mode":           "deployment",
				"serviceAccount": "otel-collector-sa",
				"config": map[string]interface{}{
					"extensions": map[string]interface{}{
						"bearertokenauth": map[string]interface{}{
							"filename": "/var/run/secrets/kubernetes.io/serviceaccount/token",
						},
					},
					"receivers": map[string]interface{}{
						"otlp": map[string]interface{}{
							"protocols": map[string]interface{}{
								"grpc": map[string]interface{}{},
								"http": map[string]interface{}{},
							},
						},
					},
					"exporters": map[string]interface{}{
						"otlp": map[string]interface{}{
							"endpoint": fmt.Sprintf("%s:4317", tempoGatewayHost),
							"tls": map[string]interface{}{
								"ca_file": "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
							},
							"auth": map[string]interface{}{
								"authenticator": "bearertokenauth",
							},
							"headers": map[string]interface{}{
								"X-Scope-OrgID": "tenant-1",
							},
						},
						"otlphttp": map[string]interface{}{
							"endpoint": fmt.Sprintf("https://%s:8080/api/traces/v1/tenant-1", tempoGatewayHost),
							"tls": map[string]interface{}{
								"ca_file": "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
							},
							"auth": map[string]interface{}{
								"authenticator": "bearertokenauth",
							},
							"headers": map[string]interface{}{
								"X-Scope-OrgID": "tenant-1",
							},
						},
					},
					"service": map[string]interface{}{
						"extensions": []interface{}{"bearertokenauth"},
						"pipelines": map[string]interface{}{
							"traces": map[string]interface{}{
								"receivers": []interface{}{"otlp"},
								"exporters": []interface{}{"otlp"},
							},
						},
					},
				},
			},
		},
	}
}
