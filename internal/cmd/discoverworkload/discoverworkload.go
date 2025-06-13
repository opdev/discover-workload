package discoverworkload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/opdev/discover-workload/internal/discover"
	"github.com/opdev/discover-workload/internal/version"
)

const (
	shortDesc = "Discover images from a deployed workload in a namespace."
	longDesc  = shortDesc + `

Deploy your workload to a given cluster, and then run this utility to detect
exactly what images are being used by the deployed workloads.`
)

type config struct {
	Timeout time.Duration
	// LogLevel becomes a slog.Level, so it must be one of the predefined levels
	LogLevel       string
	KubeconfigPath string
	LabelSelector  string
	FieldSelector  string
	CompactOutput  bool
}

func NewCommand(ctx context.Context) *cobra.Command {
	cfg := &config{}

	c := &cobra.Command{
		Use:     "discover-workload [flags] namespace1 namespace2",
		Short:   shortDesc,
		Long:    longDesc,
		Version: fmt.Sprintf("%s (%s)", version.Version, version.Commit),
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, namespaces []string) error {
			logger, err := newLogger(cfg.LogLevel, os.Stderr)
			if err != nil {
				return fmt.Errorf("failed to build a logger: %w", err)
			}

			k8sclient, err := discover.InitializeKubernetesClient(cfg.KubeconfigPath)
			if err != nil {
				logger.Error("unable to initialize a kubernetes client", "errMsg", err)
				return err
			}

			_, err = metav1.ParseToLabelSelector(cfg.LabelSelector)
			if err != nil {
				logger.Error("failed to parse label selector", "selectorValue", cfg.LabelSelector)
				return err
			}

			_, err = fields.ParseSelector(cfg.FieldSelector)
			if err != nil {
				logger.Error("failed to parse field selector", "fieldSelectorValue", cfg.FieldSelector)
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)

			go discover.StartNotifier(ctx, logger, 15*time.Second, 30*time.Second)

			logger.Info("starting to watch for workloads", "duration", cfg.Timeout)

			go gracefulShutdown(cancel)

			var buffer bytes.Buffer

			opts := discover.NewManifestJSONProcessorFnOptions{
				CompactOutput: cfg.CompactOutput,
			}
			err = discover.WatchForWorkloads(
				ctx,
				logger,
				namespaces,
				metav1.ListOptions{
					LabelSelector: cfg.LabelSelector,
					FieldSelector: cfg.FieldSelector,
				},
				k8sclient,
				discover.NewManifestJSONProcessorFn(&buffer, opts),
			)
			if err != nil {
				switch {
				case errors.Is(err, context.DeadlineExceeded):
					cancel()
					logger.Info("completed execution because the max watch duration was reached.")
					return nil
				default:
					cancel()
					return err
				}
			}

			_, err = buffer.WriteTo(cmd.OutOrStdout())
			if err != nil {
				logger.Error("failed to write manifest output", "errMsg", err)
				return err
			}

			cancel()
			return nil
		},
	}
	c.SetContext(ctx)

	flags := c.Flags()
	flags.StringVarP(&cfg.LogLevel, "log-level", "v", "INFO", "How verbose you want this tool to be")
	flags.DurationVarP(&cfg.Timeout, "duration", "d", 1*time.Minute, "How long this tool should continue to watch for workloads.")
	flags.StringVarP(&cfg.KubeconfigPath, "kubeconfig", "k", clientcmd.RecommendedHomeFile, "The kubeconfig to use for cluster access.")
	flags.StringVarP(&cfg.LabelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.")
	flags.StringVar(&cfg.FieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	flags.BoolVarP(&cfg.CompactOutput, "compact", "c", false, "Print JSON in compact format instead of pretty-printed output")

	return c
}

// newLogger returns a structured logger given the provided inputs.
func newLogger(level string, out io.Writer) (*slog.Logger, error) {
	var loggerLevel slog.Level
	unrecognizedLevel := false
	switch level {
	case "DEBUG", "INFO", "WARN", "ERROR":
		err := loggerLevel.UnmarshalText([]byte(level))
		if err != nil {
			return nil, err
		}
	default: // The input log level was unrecognized
		unrecognizedLevel = true
		loggerLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{
		Level: loggerLevel,
	}))

	if unrecognizedLevel {
		logger.Warn("fallback log level was used because user-provided value was unrecognized", "provided", level)
	}

	return logger, nil
}

func gracefulShutdown(cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}
