package discover

import (
	"context"
	"log/slog"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ProcessingFunction defines the signature of functions that will be expected
// to handle found workloads
type ProcessingFunction func(ctx context.Context, source <-chan *corev1.Pod, logger *slog.Logger) error

// WatchForWorkloads
func WatchForWorkloads(
	ctx context.Context,
	logger *slog.Logger,
	namespaces []string,
	listOptions metav1.ListOptions,
	k8sclient *kubernetes.Clientset,
	processorFn ProcessingFunction,
) error {
	podProcessing := make(chan *corev1.Pod)
	var wg sync.WaitGroup

	// Pod processing must be in the waitgroup to ensure it
	// is able to complete before this func completes.
	wg.Add(1)
	var startProcessorFnErr error
	go func() {
		defer wg.Done()
		startProcessorFnErr = processorFn(ctx, podProcessing, logger)
	}()

	for _, ns := range namespaces {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := MonitorCreatedPods(ctx, logger.With("namespace", ns), ns, podProcessing, k8sclient, listOptions)
			if err != nil {
				logger.Error("pod monitor failed", "errMsg", err)
			}
		}()
	}

	wg.Wait()
	// TODO: Improve error bubble-up
	//
	// What's the right level of error to bubble up here? If we can't enroll any
	// monitors, then does it make sense for us to continue waiting?
	//
	// Alternatively, if the processor fn fails to start, shouldn't we just
	// halt?
	//
	// For now, we'll start the monitors regardless of whether the processorFn
	// starts successfully. And if it doesn't, we'll get to the error handler
	// below and just return that error. In the future, we probably need to
	// optimize this in some way so that we know for a fact that processor is
	// running before we burn time adding monitors that will never get processes.
	if startProcessorFnErr != nil {
		return startProcessorFnErr
	}
	logger.Info("watch completed")
	return nil
}

// MonitorCreatedPods checks for pods created in namespace inNS matching options
// listOptions, and sends the object to sendTo for processing.
func MonitorCreatedPods(
	ctx context.Context,
	logger *slog.Logger,
	inNS string,
	sendTo chan *corev1.Pod,
	clientset *kubernetes.Clientset,
	listOptions metav1.ListOptions,
) error {
	logger.Debug("generating a kubernetes watcher")
	watcher, err := clientset.CoreV1().Pods(inNS).Watch(ctx, listOptions)
	if err != nil {
		logger.Error("failed to establish a watcher", "errMsg", err)
		return err
	}

	logger.Info("watching for workloads")
	defer logger.Info("done watching for workloads")
	for {
		select {
		case event, stillOpen := <-watcher.ResultChan():
			if !stillOpen {
				logger.Debug("pod monitoring completed because the watch channel closed.")
				return nil
			}
			if event.Type == watch.Added {
				item := event.Object.(*corev1.Pod)
				sendTo <- item
				continue
			}
		case <-ctx.Done():
			logger.Debug("pod monitoring completed because the context completed.")
			return nil
		}
	}
}

// InitializeKubernetesClient uses the kubeconfigPath provided to establish a
// client.
func InitializeKubernetesClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
