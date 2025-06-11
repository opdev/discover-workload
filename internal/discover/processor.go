package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"

	corev1 "k8s.io/api/core/v1"

	"github.com/opdev/discover-workload/discovery"
)

// NewManifestJSONProcessorFn produces a ProcessingFunction that will write a
// Manifest in JSON to out. This Processor finds all images from containers,
// initContainers, and ephemeralContainers.
func NewManifestJSONProcessorFn(out io.Writer) ProcessingFunction {
	return func(ctx context.Context, source <-chan *corev1.Pod, logger *slog.Logger) error {
		m := discovery.Manifest{}

		continueRunning := true
		for continueRunning {
			select {
			case p, stillOpen := <-source:
				if !stillOpen {
					logger.Debug("processorFn completing because the channel is closed")
					continueRunning = false
					break
				}
				m = appendToManifest(m, processContainers(p, logger)...)
			case <-ctx.Done():
				logger.Debug("processorFn completing because the context completed")
				continueRunning = false
			}
		}

		if len(m.DiscoveredImages) == 0 {
			logger.Info("will not write manifest because no workloads were discovered")
			return nil
		}

		manifestJSON, err := json.Marshal(m)
		if err != nil {
			logger.Error("unable to convert output manifest to JSON", "errMsg", err)
			return err
		}

		fmt.Fprintln(out, string(manifestJSON))
		return nil
	}
}

// processContainers produces DiscoveredImages for each container in the pod.
func processContainers(
	p *corev1.Pod,
	logger *slog.Logger,
) []discovery.DiscoveredImage {
	found := make([]discovery.DiscoveredImage, 0, len(p.Spec.InitContainers)+len(p.Spec.EphemeralContainers)+len(p.Spec.Containers))
	logger.Debug("found a pod!", "name", p.Name)
	for _, c := range p.Spec.Containers {
		logger.Debug("found a container", "name", c.Name, "pod", p.Name, "image", c.Image)
		// TODO: These images should only be considered if they're fully qualified.
		found = append(
			found,
			discovery.DiscoveredImage{
				Image: c.Image,
				Containers: []discovery.DiscoveredContainer{
					{
						Name: c.Name,
						Type: discovery.ContainerTypeStandard,
						Pod: discovery.DiscoveredPod{
							Name:      p.Name,
							Namespace: p.Namespace,
						},
					},
				},
			},
		)
	}
	for _, c := range p.Spec.InitContainers {
		logger.Debug("found an initContainer", "name", c.Name, "pod", p.Name, "image", c.Image)
		// TODO: These images should only be considered if they're fully qualified.
		found = append(
			found,
			discovery.DiscoveredImage{
				Image: c.Image,
				Containers: []discovery.DiscoveredContainer{
					{
						Name: c.Name,
						Type: discovery.ContainerTypeInit,
						Pod: discovery.DiscoveredPod{
							Name:      p.Name,
							Namespace: p.Namespace,
						},
					},
				},
			},
		)
	}
	for _, c := range p.Spec.EphemeralContainers {
		logger.Debug("found an ephemeralContainer", "name", c.Name, "pod", p.Name, "image", c.Image)
		// TODO: These images should only be considered if they're fully qualified.
		found = append(
			found,
			discovery.DiscoveredImage{
				Image: c.Image,
				Containers: []discovery.DiscoveredContainer{
					{
						Name: c.Name,
						Type: discovery.ContainerTypeEphemeral,
						Pod: discovery.DiscoveredPod{
							Name:      p.Name,
							Namespace: p.Namespace,
						},
					},
				},
			},
		)
	}

	return found
}

func appendToManifest(m discovery.Manifest, images ...discovery.DiscoveredImage) discovery.Manifest {
	for _, image := range images {
		idx := slices.IndexFunc(m.DiscoveredImages, func(i discovery.DiscoveredImage) bool {
			return i.Image == image.Image
		})

		if idx == -1 {
			m.DiscoveredImages = append(m.DiscoveredImages, image)
			continue
		}

		for _, container := range image.Containers {
			if !slices.Contains(m.DiscoveredImages[idx].Containers, container) {
				m.DiscoveredImages[idx].Containers = append(m.DiscoveredImages[idx].Containers, container)
			}
		}
	}

	return m
}

func imagesEqual(i1, i2 discovery.DiscoveredImage) bool {
	return i1.Image == i2.Image && slices.Equal(i1.Containers, i2.Containers)
}
