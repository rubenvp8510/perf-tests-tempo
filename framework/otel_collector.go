package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// SetupOTelCollector deploys OpenTelemetry Collector with RBAC
func (f *Framework) SetupOTelCollector() error {
	// Deploy RBAC first
	if err := f.setupOTelCollectorRBAC(); err != nil {
		return fmt.Errorf("failed to setup OTel Collector RBAC: %w", err)
	}

	// Deploy Collector CR
	if err := f.setupOTelCollectorCR(); err != nil {
		return fmt.Errorf("failed to setup OTel Collector CR: %w", err)
	}

	// Wait for collector to be ready
	return f.WaitForOTelCollectorReady(300 * time.Second)
}

// setupOTelCollectorRBAC sets up RBAC resources for OTel Collector
func (f *Framework) setupOTelCollectorRBAC() error {
	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-sa",
			Namespace: f.namespace,
		},
	}
	_, err := f.client.CoreV1().ServiceAccounts(f.namespace).Create(f.ctx, sa, metav1.CreateOptions{})
	if err != nil {
		// Ignore if already exists
	}

	// Create Role
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-role",
			Namespace: f.namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err = f.client.RbacV1().Roles(f.namespace).Create(f.ctx, role, metav1.CreateOptions{})
	if err != nil {
		// Ignore if already exists
	}

	// Create RoleBinding
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-rolebinding",
			Namespace: f.namespace,
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
				Namespace: f.namespace,
			},
		},
	}
	_, err = f.client.RbacV1().RoleBindings(f.namespace).Create(f.ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		// Ignore if already exists
	}

	// Create ClusterRole
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "allow-write-traces-tenant-1",
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
	_, err = f.client.RbacV1().ClusterRoles().Create(f.ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		// Ignore if already exists
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "allow-write-traces-tenant-1",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "allow-write-traces-tenant-1",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "otel-collector-sa",
				Namespace: f.namespace,
			},
		},
	}
	_, err = f.client.RbacV1().ClusterRoleBindings().Create(f.ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		// Ignore if already exists
	}

	return nil
}

// setupOTelCollectorCR sets up the OpenTelemetryCollector CR
func (f *Framework) setupOTelCollectorCR() error {
	// Build OpenTelemetryCollector CR programmatically
	collectorObj := f.buildOTelCollectorCR()

	// Apply using dynamic client
	dynamicClient, err := dynamic.NewForConfig(f.config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "opentelemetry.io",
		Version:  "v1beta1",
		Resource: "opentelemetrycollectors",
	}

	_, err = dynamicClient.Resource(gvr).Namespace(f.namespace).Create(f.ctx, collectorObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create OpenTelemetryCollector: %w", err)
	}

	return nil
}

// WaitForOTelCollectorReady waits for OpenTelemetry Collector to be ready
func (f *Framework) WaitForOTelCollectorReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check for deployment
		for _, deploymentName := range []string{"otel-collector-collector", "otel-collector"} {
			deployment, err := f.client.AppsV1().Deployments(f.namespace).Get(f.ctx, deploymentName, metav1.GetOptions{})
			if err == nil {
				if deployment.Status.ReadyReplicas == deployment.Status.Replicas &&
					deployment.Status.ReadyReplicas > 0 {
					return nil
				}
			}
		}

		// Check for pods directly
		pods, err := f.client.CoreV1().Pods(f.namespace).List(f.ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=opentelemetry-collector",
		})
		if err == nil {
			for _, pod := range pods.Items {
				if isPodReady(&pod) {
					return nil
				}
			}
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("otel collector not ready after %v", timeout)
}

// buildOTelCollectorCR builds an OpenTelemetryCollector CR programmatically
func (f *Framework) buildOTelCollectorCR() *unstructured.Unstructured {
	tempoGatewayHost := fmt.Sprintf("tempo-simplest-gateway.%s.svc.cluster.local", f.namespace)

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "opentelemetry.io/v1beta1",
			"kind":       "OpenTelemetryCollector",
			"metadata": map[string]interface{}{
				"name":      "otel-collector",
				"namespace": f.namespace,
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
