package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/k3s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func K3SContainer(t *testing.T) kubernetes.Interface {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	container, err := k3s.Run(ctx, "rancher/k3s:v1.27.1-k3s1")
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		require.NoError(t, container.Terminate(ctx))
	})

	rawConfig, err := container.GetKubeConfig(ctx)
	require.NoError(t, err)

	config, err := clientcmd.RESTConfigFromKubeConfig(rawConfig)
	require.NoError(t, err)

	client, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	return client
}

func KubernetesTarget(t *testing.T, client kubernetes.Interface) *corev1.Pod {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				"autopgo.scrape.path":   "/test/path",
				"autopgo.scrape.port":   "8080",
				"autopgo.scrape.scheme": "https",
			},
			Labels: map[string]string{
				"autopgo.scrape":     "true",
				"autopgo.scrape.app": "test",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "busybox:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8080,
						},
					},
				},
			},
		},
	}

	_, err := client.CoreV1().
		Pods(corev1.NamespaceDefault).
		Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		pod, err = client.CoreV1().
			Pods(corev1.NamespaceDefault).
			Get(ctx, pod.Name, metav1.GetOptions{})
		require.NoError(t, err)

		return pod.Status.Phase == corev1.PodRunning
	}, time.Minute, time.Second)

	return pod
}
