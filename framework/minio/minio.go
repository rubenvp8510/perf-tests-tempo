package minio

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// Clients provides access to Kubernetes clients needed for MinIO setup
type Clients interface {
	Client() kubernetes.Interface
	Context() context.Context
	Namespace() string
	Logger() *slog.Logger
	// GetTempoNodeSelector returns the node selector used for Tempo pods.
	// Used to create anti-affinity for MinIO.
	GetTempoNodeSelector() map[string]string
}

// buildNodeAntiAffinity creates a NodeAffinity that prevents scheduling on nodes
// matching the given selector. This ensures MinIO doesn't run on Tempo nodes.
func buildNodeAntiAffinity(nodeSelector map[string]string) *corev1.NodeAffinity {
	if len(nodeSelector) == 0 {
		return nil
	}

	var matchExpressions []corev1.NodeSelectorRequirement
	for key, value := range nodeSelector {
		var req corev1.NodeSelectorRequirement
		if value == "" {
			req = corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpDoesNotExist,
			}
		} else {
			req = corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpNotIn,
				Values:   []string{value},
			}
		}
		matchExpressions = append(matchExpressions, req)
	}

	return &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: matchExpressions,
				},
			},
		},
	}
}

// Config holds MinIO configuration options
type Config struct {
	// StorageSize is the PVC size for MinIO (e.g., "10Gi")
	// Default: "2Gi"
	StorageSize string
}

// DefaultStorageSize is the default PVC size for MinIO
const DefaultStorageSize = "2Gi"

// Setup deploys MinIO with PVC and waits for it to be ready
// Note: EnsureNamespace should be called before this function
func Setup(c Clients, config *Config) error {
	namespace := c.Namespace()
	client := c.Client()
	ctx := c.Context()

	// Determine storage size
	storageSize := DefaultStorageSize
	if config != nil && config.StorageSize != "" {
		storageSize = config.StorageSize
	}

	fmt.Printf("📦 Setting up MinIO with %s storage\n", storageSize)

	// Create PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "minio",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}

	_, err := client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create MinIO PVC: %w", err)
	}

	// Create Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
		},
		StringData: map[string]string{
			"endpoint":          fmt.Sprintf("http://minio.%s.svc.cluster.local:9000", namespace),
			"bucket":            "tempo",
			"access_key_id":     "tempo",
			"access_key_secret": "supersecret",
		},
		Type: corev1.SecretTypeOpaque,
	}

	_, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create MinIO secret: %w", err)
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "minio",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "minio",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "minio",
							Image: "quay.io/minio/minio:latest",
							Command: []string{
								"/bin/sh",
								"-c",
								"mkdir -p /storage/tempo && minio server /storage",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "MINIO_ACCESS_KEY",
									Value: "tempo",
								},
								{
									Name:  "MINIO_SECRET_KEY",
									Value: "supersecret",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9000,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "storage",
									MountPath: "/storage",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "minio",
								},
							},
						},
					},
				},
			},
		},
	}

	// Apply anti-affinity to avoid Tempo nodes if node selector is set
	if nodeSelector := c.GetTempoNodeSelector(); len(nodeSelector) > 0 {
		deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: buildNodeAntiAffinity(nodeSelector),
		}
	}

	_, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create MinIO deployment: %w", err)
	}

	// Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       9000,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(9000),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": "minio",
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	_, err = client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create MinIO service: %w", err)
	}

	// Wait for MinIO to be ready
	selector, err := labels.Parse("app.kubernetes.io/name=minio")
	if err != nil {
		return fmt.Errorf("failed to parse selector: %w", err)
	}

	return wait.ForPodsReady(c, selector, 120*time.Second, 1)
}
