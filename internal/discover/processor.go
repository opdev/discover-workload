package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
)

// NewManifestJSONProcessorFn produces a ProcessingFunction that will write a
// Manifest in JSON to out. This Processor finds all images from containers,
// initContainers, and ephemeralContainers.
func NewManifestJSONProcessorFn(out io.Writer) ProcessingFunction {
	return func(ctx context.Context, source <-chan *corev1.Pod, logger *slog.Logger) error {
		m := Manifest{}

		continueRunning := true
		for continueRunning {
			select {
			case p, stillOpen := <-source:
				if !stillOpen {
					logger.Debug("processorFn completing because the channel is closed")
					continueRunning = false
					break
				}
				m.DiscoveredImages = append(m.DiscoveredImages, processContainers(p, logger)...)
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
) []DiscoveredImage {
	found := make([]DiscoveredImage, 0, len(p.Spec.InitContainers)+len(p.Spec.EphemeralContainers)+len(p.Spec.Containers))
	logger.Debug("found a pod!", "name", p.Name)
	for _, c := range p.Spec.Containers {
		logger.Debug("found a container", "name", c.Name, "pod", p.Name, "image", c.Image)
		// TODO: These images should only be considered if they're fully qualified.
		found = append(
			found,
			DiscoveredImage{
				PodName:       p.Name,
				ContainerName: c.Name,
				Image:         c.Image,
			},
		)
	}
	for _, c := range p.Spec.InitContainers {
		logger.Debug("found an initContainer", "name", c.Name, "pod", p.Name, "image", c.Image)
		// TODO: These images should only be considered if they're fully qualified.
		found = append(
			found,
			DiscoveredImage{
				PodName:       p.Name,
				ContainerName: c.Name,
				Image:         c.Image,
			},
		)
	}
	for _, c := range p.Spec.EphemeralContainers {
		logger.Debug("found an ephemeralContainer", "name", c.Name, "pod", p.Name, "image", c.Image)
		// TODO: These images should only be considered if they're fully qualified.
		found = append(
			found,
			DiscoveredImage{
				PodName:       p.Name,
				ContainerName: c.Name,
				Image:         c.Image,
			},
		)
	}

	return found
}
