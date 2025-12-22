package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/webdevops/kube-janitor/config"
	"github.com/webdevops/kube-janitor/kube_janitor"
)

const (
	Author = "webdevops.io"

	UserAgent = "kube-janitor/"
)

var (
	argparser *flags.Parser
	Opts      config.Opts
)

func main() {
	initArgparser()
	initLogger()

	printStartup("kube-janitor", Author)

	initSystem()

	janitor := kube_janitor.New()
	janitor.SetKubeconfig(Opts.Kubernetes.Config).
		SetLogger(logger).
		Connect().
		SetKubePageSize(Opts.Kubernetes.ItemsPerPage).
		LoadConfigFromFile(Opts.Janitor.Config).
		SetDryRun(Opts.Janitor.DryRun)

	if Opts.Janitor.Once {
		err := janitor.Run()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		logger.Info("finished once run, existing")
	} else {
		janitor.Start(Opts.Janitor.Interval)

		logger.Info("starting http server", slog.String("bind", Opts.Server.Bind))
		startHttpServer()
	}
}

// initArgparser inits the argument parser
func initArgparser() {
	argparser = flags.NewParser(&Opts, flags.Default)
	_, err := argparser.Parse()

	// check if there is an parse error
	if err != nil {
		var flagsErr *flags.Error
		if ok := errors.As(err, &flagsErr); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println()
			argparser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}
}

// startHttpServer start and handle prometheus handler
func startHttpServer() {
	mux := http.NewServeMux()

	// healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			logger.Error(err.Error())
		}
	})

	// readyz
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			logger.Error(err.Error())
		}
	})

	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         Opts.Server.Bind,
		Handler:      mux,
		ReadTimeout:  Opts.Server.ReadTimeout,
		WriteTimeout: Opts.Server.WriteTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal(err.Error())
	}
}
