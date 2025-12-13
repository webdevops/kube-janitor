# Kubernetes Janitor (kube-janitor)

[![license](https://img.shields.io/github/license/webdevops/kube-janitor.svg)](https://github.com/webdevops/kube-janitor/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fkube--janitor-blue)](https://hub.docker.com/r/webdevops/kube-janitor/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fkube--janitor-blue)](https://quay.io/repository/webdevops/kube-janitor)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kube-janitor)](https://artifacthub.io/packages/search?repo=kube-janitor)

Kubernetes janitor which deletes resources by TTL label written in Golang

## Configuration

see [`example.yaml`](example.yaml) for example configurations

```
Usage:
  kube-janitor [OPTIONS]

Application Options:
      --log.level=[trace|debug|info|warning|error] Log level (default: info) [$LOG_LEVEL]
      --log.format=[logfmt|json]                   Log format (default: logfmt) [$LOG_FORMAT]
      --log.source=[|short|file|full]              Show source for every log message (useful for debugging and bug reports) [$LOG_SOURCE]
      --log.color=[|auto|yes|no]                   Enable color for logs [$LOG_COLOR]
      --log.time                                   Show log time [$LOG_TIME]
      --interval=                                  Janitor interval (time.duration) (default: 1h) [$JANITOR_INTERVAL]
      --config=                                    Path to kube-janitor config file [$JANITOR_CONFIG]
      --dry-run                                    Dry run (no delete) [$JANITOR_DRYRUN]
      --once                                       Run once and exit [$JANITOR_ONCE]
      --kubeconfig=                                Kuberentes config path (should be empty if in-cluster) [$KUBECONFIG]
      --server.bind=                               Server address (default: :8080) [$SERVER_BIND]
      --server.timeout.read=                       Server read timeout (default: 5s) [$SERVER_TIMEOUT_READ]
      --server.timeout.write=                      Server write timeout (default: 10s) [$SERVER_TIMEOUT_WRITE]

Help Options:
  -h, --help                                       Show this help message
```
## Metrics

| Metric                            | Description                                                        |
|-----------------------------------|--------------------------------------------------------------------|
| `kube_janitor_resource_ttl`       | Expiry date (unix timestamp) for every resource which was detected |
