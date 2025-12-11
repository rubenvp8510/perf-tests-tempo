package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SetupMinIO deploys MinIO with PVC and waits for it to be ready
func (f *Framework) SetupMinIO() error {
	if err := f.EnsureNamespace(); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: f.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "minio",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
		},
	}

	_, err := f.client.CoreV1().PersistentVolumeClaims(f.namespace).Create(f.ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create MinIO PVC: %w", err)
	}

	// Create Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: f.namespace,
		},
		StringData: map[string]string{
			"endpoint":        fmt.Sprintf("http://minio.%s.svc.cluster.local:9000", f.namespace),
			"bucket":          "tempo",
			"access_key_id":   "tempo",
			"access_key_secret": "supersecret",
		},
		Type: corev1.SecretTypeOpaque,
	}

	_, err = f.client.CoreV1().Secrets(f.namespace).Create(f.ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create MinIO secret: %w", err)
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: f.namespace,
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

	_, err = f.client.AppsV1().Deployments(f.namespace).Create(f.ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create MinIO deployment: %w", err)
	}

	// Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: f.namespace,
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

	_, err = f.client.CoreV1().Services(f.namespace).Create(f.ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create MinIO service: %w", err)
	}

	// Wait for MinIO to be ready
	selector, err := labels.Parse("app.kubernetes.io/name=minio")
	if err != nil {
		return fmt.Errorf("failed to parse selector: %w", err)
	}

	return f.WaitForPodsReady(selector, 120*time.Second, 1)
}

