package target

import (
	"context"
	"log/slog"
	"net"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The KubernetesSource type is used to list scrapable targets from a Kubernetes cluster.
	KubernetesSource struct {
		client kubernetes.Interface
		labels labels.Selector
		fields fields.Selector
	}
)

// NewKubernetesSource returns a new instance of the KubernetesSource type that can list scrapable targets contained
// within a Kubernetes cluster. The app parameter determines which pods are scraped based on their autopgo.app label.
func NewKubernetesSource(client kubernetes.Interface, app string) (*KubernetesSource, error) {
	return &KubernetesSource{
		client: client,
		labels: labels.SelectorFromSet(labels.Set{
			appLabel:    app,
			scrapeLabel: "true",
		}),
		fields: fields.SelectorFromSet(fields.Set{
			// We only want pods that have a running status, so they'll have a pod IP and in theory
			// be addressable.
			"status.phase": string(corev1.PodRunning),
		}),
	}, nil
}

// List all scrapable targets within the Kubernetes cluster. This functions by listing all pods that have the label
// autopgo.scrape set to true and the autopgo.app label matching that of the scraper. The pod IP will be used as the
// Target.Address field and an optional pprof path can be provided by setting the autopgo.path label on the pod.
func (ks *KubernetesSource) List(ctx context.Context) ([]Target, error) {
	log := logger.FromContext(ctx)

	options := metav1.ListOptions{
		LabelSelector: ks.labels.String(),
		FieldSelector: ks.fields.String(),
	}

	log.DebugContext(ctx, "listing kubernetes pods")
	pods, err := ks.client.CoreV1().Pods(corev1.NamespaceAll).List(ctx, options)
	if err != nil {
		return nil, err
	}

	log.
		With(slog.Int("count", len(pods.Items))).
		DebugContext(ctx, "found labelled pods")

	var targets []Target
	for _, pod := range pods.Items {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		log = log.With(
			slog.String("pod.name", pod.Name),
			slog.String("pod.namespace", pod.Namespace),
			slog.String("pod.uid", string(pod.UID)),
		)

		if pod.Status.PodIP == "" {
			log.WarnContext(ctx, "ignoring pod with no pod ip")
			continue
		}

		if pod.Status.Phase != corev1.PodRunning {
			log.WarnContext(ctx, "ignoring pod that is not running")
			continue
		}

		annotations := pod.GetObjectMeta().GetAnnotations()

		port := annotations[portLabel]
		if port == "" {
			log.WarnContext(ctx, "ignoring pod with empty port annotation")
			continue
		}

		scheme := annotations[schemeLabel]
		if scheme == "" {
			scheme = "http"
		}

		u := url.URL{
			Scheme: scheme,
			Host:   net.JoinHostPort(pod.Status.PodIP, port),
		}

		targets = append(targets, Target{
			Address: u.String(),
			Path:    annotations[pathLabel],
		})
	}

	return targets, ctx.Err()
}

// Name returns "kubernetes". This method is used to implement the operation.Check interface for use in health checks.
func (ks *KubernetesSource) Name() string {
	return "kubernetes"
}

// Check attempts to list pods within kubernetes. This method is used to implement the operation.Checker interface for
// use in health checks.
func (ks *KubernetesSource) Check(ctx context.Context) error {
	options := metav1.ListOptions{
		LabelSelector: ks.labels.String(),
		FieldSelector: ks.fields.String(),
	}

	_, err := ks.client.CoreV1().Pods(corev1.NamespaceAll).List(ctx, options)
	return err
}
