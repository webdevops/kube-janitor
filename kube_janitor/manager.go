package kube_janitor

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/go-logr/logr"
	yaml "github.com/goccy/go-yaml"
	"github.com/webdevops/go-common/log/slogger"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubelog "sigs.k8s.io/controller-runtime/pkg/log"
)

type (
	Janitor struct {
		kubeconfig string

		config *Config

		kubeClient *kubernetes.Clientset
		dynClient  *dynamic.DynamicClient

		logger *slogger.Logger

		dryRun bool

		prometheus JanitorMetrics

		kubePageLimit int64
	}
)

func New() *Janitor {
	j := &Janitor{}
	j.init()
	return j
}

func (j *Janitor) init() {
	j.setupMetrics()
	j.kubePageLimit = KubeDefaultListLimit
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

	j.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
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

func (j *Janitor) SetKubeconfig(kubeconfig string) *Janitor {
	j.kubeconfig = kubeconfig
	return j
}

func (j *Janitor) GetConfigFromFile(path string) *Janitor {
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
		logger.Fatal("failed to parse config file")
	}

	err = j.config.Validate()
	if err != nil {
		logger.Fatal("config validation failed", slog.Any("error", err))
	}
	return j
}

func (j *Janitor) SetLogger(logger *slogger.Logger) *Janitor {
	j.logger = logger
	return j
}

func (j *Janitor) SetDryRun(val bool) *Janitor {
	j.dryRun = val
	return j
}

func (j *Janitor) SetKubePageSize(val int64) *Janitor {
	j.kubePageLimit = val
	return j
}

func (j *Janitor) Start(interval time.Duration) *Janitor {
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

	return j
}

func (j *Janitor) Run() error {
	j.connect()
	ctx := context.Background()

	if j.config.Ttl.Label != "" || j.config.Ttl.Annotation != "" {
		if err := j.runTtlResources(ctx); err != nil {
			return err
		}
	} else {
		j.logger.Debug("skipping TTL run, no label or annotation defined")
	}

	if len(j.config.Rules) > 0 {
		if err := j.runRules(ctx); err != nil {
			return err
		}
	} else {
		j.logger.Debug("skipping rules run, no rules defined")
	}

	return nil
}
