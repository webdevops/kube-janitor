package kube_janitor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-logr/logr"
	yaml "github.com/goccy/go-yaml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/webdevops/go-common/log/slogger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubelog "sigs.k8s.io/controller-runtime/pkg/log"
)

type (
	Janitor struct {
		kubeconfig string

		config *Config

		dynClient *dynamic.DynamicClient

		logger *slogger.Logger

		dryRun bool

		prometheus JanitorMetrics
	}
)

func New() *Janitor {
	j := &Janitor{}
	j.init()
	return j
}

func (j *Janitor) init() {
	j.setupMetrics()
}

func (j *Janitor) connect() {
	if j.dynClient != nil {
		return
	}

	var err error
	var config *rest.Config

	if j.kubeconfig != "" {
		// KUBECONFIG
		config, err = clientcmd.BuildConfigFromFlags("", j.kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		// K8S in cluster
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	j.dynClient, err = dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// kube logger (with translator)
	logrHandler := logr.NewContextWithSlogLogger(context.Background(), j.logger.Slog())
	kubeLogger, err := logr.FromContext(logrHandler)
	if err != nil {
		panic(err.Error())
	}
	kubelog.SetLogger(kubeLogger)
}

func (j *Janitor) SetKubeconfig(kubeconfig string) {
	j.kubeconfig = kubeconfig
}

func (j *Janitor) GetConfigFromFile(path string) {
	if j.config == nil {
		j.config = NewConfig()
	}

	logger := j.logger.With(slog.String("path", path))

	logger.Info("reading configuration from file")

	/* #nosec */
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Fatal("failed to read config file", slog.Any("error", err.Error()))
	}

	logger.Info("parsing configuration")
	err = yaml.UnmarshalWithOptions(data, j.config, yaml.Strict(), yaml.UseJSONUnmarshaler())
	if err != nil {
		fmt.Println(err)
		logger.Fatal("failed to parse config file")
	}

	err = j.config.Validate()
	if err != nil {
		logger.Fatal("config validation failed", slog.Any("error", err))
	}
}

func (j *Janitor) SetLogger(logger *slogger.Logger) {
	j.logger = logger
}

func (j *Janitor) SetDryRun(val bool) {
	j.dryRun = val
}

func (j *Janitor) Start(interval time.Duration) {
	go func() {
		// wait for settle down
		time.Sleep(10 * time.Second)

		for {
			j.logger.Info("starting janitor run")
			startTime := time.Now()

			err := j.Run()
			if err != nil {
				panic(err)
			}

			j.logger.Info("janitor run finished", slog.Duration("duration", time.Since(startTime)), slog.Time("nextRun", time.Now().Add(interval)))
			time.Sleep(interval)
		}
	}()
}

func (j *Janitor) Run() error {
	j.connect()

	ctx := context.Background()

	metricResourceTtl := prometheusCommon.NewMetricsList()

	for _, resourceType := range j.config.Resources {
		gvkLogger := j.logger.With(slog.Any("gvk", resourceType))
		gvkLogger.Debug("checking resources")

		err := j.kubeEachResource(ctx, resourceType.AsGVR(), func(resource unstructured.Unstructured) error {

			if ttl := resource.GetLabels()[j.config.Label]; ttl != "" {
				resourceLogger := gvkLogger.With(
					slog.String("namespace", resource.GetNamespace()),
					slog.String("resource", resource.GetName()),
					slog.String("ttl", ttl),
				)

				parsedDate, expired, err := j.checkExpiryDate(resource.GetCreationTimestamp().Time, ttl)
				if err != nil {
					resourceLogger.Error("unable to parse expiration date", slog.String("raw", ttl), slog.Any("error", err))
					return nil
				}

				metricResourceTtl.AddTime(
					prometheus.Labels{
						"version":   resource.GetAPIVersion(),
						"kind":      resource.GetKind(),
						"namespace": resource.GetNamespace(),
						"name":      resource.GetName(),
					},
					*parsedDate,
				)

				resourceLogger.Debug("found resource with valid TTL", slog.Time("expirationDate", *parsedDate))

				if expired {
					if j.dryRun {
						resourceLogger.Info("resource is expired, would delete resource (DRY-RUN)", slog.Time("expirationDate", *parsedDate))
					} else {
						resourceLogger.Info("resource is expired, deleting", slog.Time("expirationDate", *parsedDate))
						err := j.dynClient.Resource(resourceType.AsGVR()).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
						if err != nil {
							return err
						}
					}
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	metricResourceTtl.GaugeSet(j.prometheus.ttl)

	return nil
}
