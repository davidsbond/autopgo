package target_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/davidsbond/autopgo/internal/target"
	"github.com/davidsbond/autopgo/internal/testutil"
)

func TestKubernetesSource_List(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		App          string
		ExpectsError bool
		Objects      []runtime.Object
		Expected     []target.Target
	}{
		{
			Name: "success",
			App:  "test",
			Expected: []target.Target{
				{
					Address: "https://127.0.0.1:8080",
					Path:    "/test/path",
				},
			},
			Objects: []runtime.Object{
				&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								Labels: map[string]string{
									"autopgo.scrape":     "true",
									"autopgo.scrape.app": "test",
								},
								Annotations: map[string]string{
									"autopgo.scrape.path":   "/test/path",
									"autopgo.scrape.port":   "8080",
									"autopgo.scrape.scheme": "https",
								},
								Namespace: corev1.NamespaceDefault,
							},
							Status: corev1.PodStatus{
								PodIP: "127.0.0.1",
								Phase: corev1.PodRunning,
							},
						},
					},
				},
			},
		},
		{
			Name: "ignores missing port",
			App:  "test",
			Objects: []runtime.Object{
				&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								Labels: map[string]string{
									"autopgo.scrape":     "true",
									"autopgo.scrape.app": "test",
								},
								Annotations: map[string]string{
									"autopgo.scrape.path":   "/test/path",
									"autopgo.scrape.scheme": "https",
								},
								Namespace: corev1.NamespaceDefault,
							},
							Status: corev1.PodStatus{
								PodIP: "127.0.0.1",
								Phase: corev1.PodRunning,
							},
						},
					},
				},
			},
		},
		{
			Name: "ignores no pod ip",
			App:  "test",
			Objects: []runtime.Object{
				&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								Labels: map[string]string{
									"autopgo.scrape":     "true",
									"autopgo.scrape.app": "test",
								},
								Annotations: map[string]string{
									"autopgo.scrape.path":   "/test/path",
									"autopgo.scrape.port":   "8080",
									"autopgo.scrape.scheme": "https",
								},
								Namespace: corev1.NamespaceDefault,
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
							},
						},
					},
				},
			},
		},
		{
			Name: "ignores not running",
			App:  "test",
			Objects: []runtime.Object{
				&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								Labels: map[string]string{
									"autopgo.scrape":     "true",
									"autopgo.scrape.app": "test",
								},
								Annotations: map[string]string{
									"autopgo.scrape.path":   "/test/path",
									"autopgo.scrape.port":   "8080",
									"autopgo.scrape.scheme": "https",
								},
								Namespace: corev1.NamespaceDefault,
							},
							Status: corev1.PodStatus{
								PodIP: "127.0.0.1",
								Phase: corev1.PodFailed,
							},
						},
					},
				},
			},
		},
		{
			Name: "defaults scheme to http",
			App:  "test",
			Expected: []target.Target{
				{
					Address: "http://127.0.0.1:8080",
					Path:    "/test/path",
				},
			},
			Objects: []runtime.Object{
				&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								Labels: map[string]string{
									"autopgo.scrape":     "true",
									"autopgo.scrape.app": "test",
								},
								Annotations: map[string]string{
									"autopgo.scrape.path": "/test/path",
									"autopgo.scrape.port": "8080",
								},
								Namespace: corev1.NamespaceDefault,
							},
							Status: corev1.PodStatus{
								PodIP: "127.0.0.1",
								Phase: corev1.PodRunning,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			kube := fake.NewClientset(tc.Objects...)

			source, err := target.NewKubernetesSource(kube, tc.App)
			require.NoError(t, err)

			actual, err := source.List(ctx)
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestKubernetesSource_List_Integration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	client := testutil.K3SContainer(t)
	expected := testutil.KubernetesTarget(t, client)

	source, err := target.NewKubernetesSource(client, "test")
	require.NoError(t, err)

	result, err := source.List(ctx)
	require.NoError(t, err)

	if assert.Len(t, result, 1) {
		actual := result[0]
		assert.EqualValues(t, expected, actual)
	}
}
