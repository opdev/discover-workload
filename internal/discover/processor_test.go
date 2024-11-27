package discover

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestManifestJSONProcessor(t *testing.T) {
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
			expected: []byte("{\"DiscoveredImages\":[{\"PodName\":\"init-podname\",\"ContainerName\":\"init-cname\",\"Image\":\"example.com/namespace/image:0.0.1\"}]}\n"),
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
	testcases := map[string]struct {
		input    corev1.Pod
		expected []DiscoveredImage
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
			expected: []DiscoveredImage{
				{
					PodName:       "init-podname",
					ContainerName: "init-cname",
					Image:         "example.com/namespace/image:0.0.1",
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
			expected: []DiscoveredImage{
				{
					PodName:       "c-podname",
					ContainerName: "c-cname",
					Image:         "example.com/namespace/image:0.0.1",
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
			expected: []DiscoveredImage{
				{
					PodName:       "eph-podname",
					ContainerName: "eph-cname",
					Image:         "example.com/namespace/image:0.0.1",
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
			expected: []DiscoveredImage{
				{
					PodName:       "all-podname",
					ContainerName: "c-cname",
					Image:         "example.com/namespace/image:0.0.1",
				},
				{
					PodName:       "all-podname",
					ContainerName: "init-cname",
					Image:         "example.com/namespace/image:0.0.1",
				},
				{
					PodName:       "all-podname",
					ContainerName: "eph-cname",
					Image:         "example.com/namespace/image:0.0.1",
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
			if !slices.Equal(actual, tc.expected) {
				t.Fatalf("Processing returned %q; expected %q", actual, tc.expected)
			}
		})
	}
}
