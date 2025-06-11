package discover

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opdev/discover-workload/discovery"
)

func TestManifestInsert(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		ctx      context.Context
		input    []corev1.Pod
		expected discovery.Manifest
	}{
		"unique containers": {
			ctx: context.TODO(),
			input: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-1-1",
								Image: "example.com/namespace/image:0.1.1",
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "container-1-2",
								Image: "example.com/namespace/image:0.1.2",
							},
						},
						EphemeralContainers: []corev1.EphemeralContainer{
							{
								EphemeralContainerCommon: corev1.EphemeralContainerCommon{
									Name:  "container-1-3",
									Image: "example.com/namespace/image:0.1.3",
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-2"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-2-1",
								Image: "example.com/namespace/image:0.2.1",
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "container-2-2",
								Image: "example.com/namespace/image:0.2.2",
							},
						},
						EphemeralContainers: []corev1.EphemeralContainer{
							{
								EphemeralContainerCommon: corev1.EphemeralContainerCommon{
									Name:  "container-2-3",
									Image: "example.com/namespace/image:0.2.3",
								},
							},
						},
					},
				},
			},
			expected: discovery.Manifest{
				DiscoveredImages: []discovery.DiscoveredImage{
					{
						Image: "example.com/namespace/image:0.1.1",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.1.2",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-2",
								Type: discovery.ContainerTypeInit,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.1.3",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-3",
								Type: discovery.ContainerTypeEphemeral,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.2.1",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-2-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.2.2",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-2-2",
								Type: discovery.ContainerTypeInit,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.2.3",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-2-3",
								Type: discovery.ContainerTypeEphemeral,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
						},
					},
				},
			},
		},
		"multiple pods with same image": {
			ctx: context.TODO(),
			input: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-1-1",
								Image: "example.com/namespace/image:0.1.0",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-2"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-2-1",
								Image: "example.com/namespace/image:0.1.0",
							},
						},
					},
				},
			},
			expected: discovery.Manifest{
				DiscoveredImages: []discovery.DiscoveredImage{
					{
						Image: "example.com/namespace/image:0.1.0",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
							{
								Name: "container-2-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
						},
					},
				},
			},
		},
		"multiple containers with same image": {
			ctx: context.TODO(),
			input: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-1-1",
								Image: "example.com/namespace/image:0.1.0",
							},
							{
								Name:  "container-1-2",
								Image: "example.com/namespace/image:0.1.0",
							},
						},
					},
				},
			},
			expected: discovery.Manifest{
				DiscoveredImages: []discovery.DiscoveredImage{
					{
						Image: "example.com/namespace/image:0.1.0",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
							{
								Name: "container-1-2",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
						},
					},
				},
			},
		},
		"multiple pods and containers with same image": {
			ctx: context.TODO(),
			input: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-1-1",
								Image: "example.com/namespace/image:0.1.1",
							},
							{
								Name:  "container-1-2",
								Image: "example.com/namespace/image:0.1.2",
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "container-1-3",
								Image: "example.com/namespace/image:0.1.1",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-2"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-2-1",
								Image: "example.com/namespace/image:0.1.2",
							},
							{
								Name:  "container-2-2",
								Image: "example.com/namespace/image:0.1.3",
							},
							{
								Name:  "container-2-3",
								Image: "example.com/namespace/image:0.1.3",
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "container-2-4",
								Image: "example.com/namespace/image:0.1.2",
							},
						},
					},
				},
			},
			expected: discovery.Manifest{
				DiscoveredImages: []discovery.DiscoveredImage{
					{
						Image: "example.com/namespace/image:0.1.1",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
							{
								Name: "container-1-3",
								Type: discovery.ContainerTypeInit,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.1.2",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-1-2",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-1",
								},
							},
							{
								Name: "container-2-1",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
							{
								Name: "container-2-4",
								Type: discovery.ContainerTypeInit,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
						},
					},
					{
						Image: "example.com/namespace/image:0.1.3",
						Containers: []discovery.DiscoveredContainer{
							{
								Name: "container-2-2",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
							{
								Name: "container-2-3",
								Type: discovery.ContainerTypeStandard,
								Pod: discovery.DiscoveredPod{
									Name: "pod-2",
								},
							},
						},
					},
				},
			},
		},
	}

	for description, tc := range testcases {
		testLogger := NewSlogDiscardLogger()
		t.Run(description, func(t *testing.T) {
			t.Parallel()
			actual := discovery.Manifest{}
			for _, pod := range tc.input {
				actual = appendToManifest(actual, processContainers(&pod, testLogger)...)
			}

			if len(actual.DiscoveredImages) != len(tc.expected.DiscoveredImages) {
				t.Fatalf("Processing returned %v; expected %v", actual, tc.expected)
			}
			for idx := range actual.DiscoveredImages {
				if !imagesEqual(actual.DiscoveredImages[idx], tc.expected.DiscoveredImages[idx]) {
					t.Fatalf("Processing returned %v; expected %v", actual, tc.expected)
				}
			}
		})
	}
}

func TestManifestJSONProcessor(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		ctx      context.Context
		input    []corev1.Pod
		expected []byte
	}{
		"initContainer only": {
			ctx: context.TODO(),
			input: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "init-podname"},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name:  "init-cname",
								Image: "example.com/namespace/image:0.0.1",
							},
						},
					},
				},
			},
			expected: []byte("{\"DiscoveredImages\":[{\"Image\":\"example.com/namespace/image:0.0.1\",\"Containers\":[{\"Name\":\"init-cname\",\"Type\":\"InitContainer\",\"Pod\":{\"Name\":\"init-podname\",\"Namespace\":\"\"}}]}]}\n"),
		},
	}

	for description, tc := range testcases {
		testLogger := NewSlogDiscardLogger()
		t.Run(description, func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer([]byte{})
			fn := NewManifestJSONProcessorFn(buffer)

			ch := make(chan *corev1.Pod)

			var wg sync.WaitGroup
			wg.Add(1)
			var testFnErr error
			go func() {
				defer wg.Done()
				testFnErr = fn(tc.ctx, ch, testLogger)
			}()

			for i := range tc.input {
				ch <- &(tc.input[i])
			}

			close(ch)
			wg.Wait()
			actual, err := io.ReadAll(buffer)

			if testFnErr != nil {
				t.Fatalf("processor function threw an error unexpectedly: %q", err)
			}

			if err != nil {
				t.Fatalf("unable to read output written to buffer: %q", err)
			}

			if !bytes.Equal(actual, tc.expected) {
				t.Fatalf("ManifestJSONPrinter processing function returned the wrong output. actual: %q expected %q", actual, tc.expected)
			}
		})
	}
}

func TestContainerProcessing(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		input    corev1.Pod
		expected []discovery.DiscoveredImage
	}{
		"initContainers only": {
			input: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "init-podname"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "init-cname",
							Image: "example.com/namespace/image:0.0.1",
						},
					},
				},
			},
			expected: []discovery.DiscoveredImage{
				{
					Image: "example.com/namespace/image:0.0.1",
					Containers: []discovery.DiscoveredContainer{
						{
							Name: "init-cname",
							Type: discovery.ContainerTypeInit,
							Pod: discovery.DiscoveredPod{
								Name: "init-podname",
							},
						},
					},
				},
			},
		},
		"containers only": {
			input: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "c-podname"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "c-cname",
							Image: "example.com/namespace/image:0.0.1",
						},
					},
				},
			},
			expected: []discovery.DiscoveredImage{
				{
					Image: "example.com/namespace/image:0.0.1",
					Containers: []discovery.DiscoveredContainer{
						{
							Name: "c-cname",
							Type: discovery.ContainerTypeStandard,
							Pod: discovery.DiscoveredPod{
								Name: "c-podname",
							},
						},
					},
				},
			},
		},
		"ephemeralContainers only": {
			input: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "eph-podname"},
				Spec: corev1.PodSpec{
					EphemeralContainers: []corev1.EphemeralContainer{
						{
							EphemeralContainerCommon: corev1.EphemeralContainerCommon{
								Name:  "eph-cname",
								Image: "example.com/namespace/image:0.0.1",
							},
						},
					},
				},
			},
			expected: []discovery.DiscoveredImage{
				{
					Image: "example.com/namespace/image:0.0.1",
					Containers: []discovery.DiscoveredContainer{
						{
							Name: "eph-cname",
							Type: discovery.ContainerTypeEphemeral,
							Pod: discovery.DiscoveredPod{
								Name: "eph-podname",
							},
						},
					},
				},
			},
		},
		"all container types": {
			input: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "all-podname"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "c-cname",
							Image: "example.com/namespace/image:0.0.1",
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "init-cname",
							Image: "example.com/namespace/image:0.0.1",
						},
					},
					EphemeralContainers: []corev1.EphemeralContainer{
						{
							EphemeralContainerCommon: corev1.EphemeralContainerCommon{
								Name:  "eph-cname",
								Image: "example.com/namespace/image:0.0.1",
							},
						},
					},
				},
			},
			expected: []discovery.DiscoveredImage{
				{
					Image: "example.com/namespace/image:0.0.1",
					Containers: []discovery.DiscoveredContainer{
						{
							Name: "c-cname",
							Type: discovery.ContainerTypeStandard,
							Pod: discovery.DiscoveredPod{
								Name: "all-podname",
							},
						},
					},
				}, {
					Image: "example.com/namespace/image:0.0.1",
					Containers: []discovery.DiscoveredContainer{
						{
							Name: "init-cname",
							Type: discovery.ContainerTypeInit,
							Pod: discovery.DiscoveredPod{
								Name: "all-podname",
							},
						},
					},
				}, {
					Image: "example.com/namespace/image:0.0.1",
					Containers: []discovery.DiscoveredContainer{
						{
							Name: "eph-cname",
							Type: discovery.ContainerTypeEphemeral,
							Pod: discovery.DiscoveredPod{
								Name: "all-podname",
							},
						},
					},
				},
			},
		},
	}

	for description, tc := range testcases {
		testLogger := NewSlogDiscardLogger()
		t.Run(description, func(t *testing.T) {
			t.Parallel()
			actual := processContainers(&tc.input, testLogger)
			// Note: slices.Equal checks values at increasing indexes. For test
			// purposes, make sure the actual and expected values are sorted. If
			// not possible in the definition of the table, then we'll need to
			// add a sort function here.
			if len(actual) != len(tc.expected) {
				t.Fatalf("Processing returned %v; expected %v", actual, tc.expected)
			}

			for idx := range actual {
				if !imagesEqual(actual[idx], tc.expected[idx]) {
					t.Fatalf("Processing returned %v; expected %v", actual, tc.expected)
				}
			}
		})
	}
}
