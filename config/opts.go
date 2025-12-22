package config

import (
	"encoding/json"
	"time"
)

type (
	Opts struct {
		Version struct {
			Version  bool    `long:"version" description:"Show version"`
			Template *string `long:"version.template" description:"Version go template, eg {{.Version}}"`
		}

		// logger
		Logger struct {
			Level  string `long:"log.level"    env:"LOG_LEVEL"   description:"Log level" choice:"trace" choice:"debug" choice:"info" choice:"warning" choice:"error" default:"info"`                          // nolint:staticcheck // multiple choices are ok
			Format string `long:"log.format"   env:"LOG_FORMAT"  description:"Log format" choice:"logfmt" choice:"json" default:"logfmt"`                                                                     // nolint:staticcheck // multiple choices are ok
			Source string `long:"log.source"   env:"LOG_SOURCE"  description:"Show source for every log message (useful for debugging and bug reports)" choice:"" choice:"short" choice:"file" choice:"full"` // nolint:staticcheck // multiple choices are ok
			Color  string `long:"log.color"    env:"LOG_COLOR"   description:"Enable color for logs" choice:"" choice:"auto" choice:"yes" choice:"no"`                                                        // nolint:staticcheck // multiple choices are ok
			Time   bool   `long:"log.time"     env:"LOG_TIME"    description:"Show log time"`
		}

		Janitor struct {
			Interval time.Duration `long:"interval"    env:"JANITOR_INTERVAL"  description:"Janitor interval (time.duration)"  default:"1h"`
			Config   string        `long:"config"      env:"JANITOR_CONFIG"    description:"Path to kube-janitor config file" required:"true"`
			DryRun   bool          `long:"dry-run"     env:"JANITOR_DRYRUN"    description:"Dry run (no delete)"`
			Once     bool          `long:"once"        env:"JANITOR_ONCE"      description:"Run once and exit"`
		}

		// kubernetes settings
		Kubernetes struct {
			Config       string `long:"kubeconfig"            env:"KUBECONFIG"               description:"Kuberentes config path (should be empty if in-cluster)"`
			ItemsPerPage int64  `long:"kube.itemsperpage"     env:"KUBE_ITEMSPERPAGE"        description:"Defines how many items per page janitor should process" default:"100"`
		}

		// general options
		Server struct {
			// general options
			Bind         string        `long:"server.bind"              env:"SERVER_BIND"           description:"Server address"        default:":8080"`
			ReadTimeout  time.Duration `long:"server.timeout.read"      env:"SERVER_TIMEOUT_READ"   description:"Server read timeout"   default:"5s"`
			WriteTimeout time.Duration `long:"server.timeout.write"     env:"SERVER_TIMEOUT_WRITE"  description:"Server write timeout"  default:"10s"`
		}
	}
)

func (o *Opts) GetJson() []byte {
	jsonBytes, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return jsonBytes
}
