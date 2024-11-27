package discover

// NOTE:
// 
// These types are incomplete, and likely will not live here in the long
// run. They're here temporarily for development purposes.

// Manifest represents the discovered components for a given application.
type Manifest struct {
	DiscoveredImages []DiscoveredImage
}

// DiscoveredImage is a container image that' been discovered based on the
// workload watchers.
type DiscoveredImage struct {
	// PodName is the pod.metadata.name value where the image was discovered.
	PodName       string
	// ContainerName represents the container in the pod that had the image.
	ContainerName string
	// Image is a fully qualified container image name and tag or digest.
	Image         string
}
