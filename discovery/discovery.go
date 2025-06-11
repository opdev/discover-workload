package discovery

// Manifest represents the discovered components for a given application.
type Manifest struct {
	DiscoveredImages []DiscoveredImage
}

// DiscoveredImage is a container image which was discovered in one or more workloads.
type DiscoveredImage struct {
	// Image is a fully qualified container image name and tag or digest.
	Image string

	// Containers is a list of DiscoveredContainer objects which are using
	// the discovered image.
	Containers []DiscoveredContainer
}

// DiscoveredContainer is a container which was observed during the discovery process.
type DiscoveredContainer struct {
	// Name is the name of a container in a pod.
	Name string

	// Type is the ContainerType of the container in its pod.
	Type ContainerType

	// Pod is the DiscoveredPod which this container is a part of.
	Pod DiscoveredPod
}

// ContainerType is the type of a container in a pod.
type ContainerType = string

const (
	ContainerTypeStandard  ContainerType = "Container"
	ContainerTypeInit      ContainerType = "InitContainer"
	ContainerTypeEphemeral ContainerType = "EphemeralContainer"
)

// DiscoveredPod is a pod that contains a discovered image.
type DiscoveredPod struct {
	// Name is the pod.metadata.name value where the image was discovered.
	Name string

	// Namespace is the pod.metadata.namespace value of the pod.
	Namespace string
}
