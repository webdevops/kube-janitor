# Kubernetes Janitor (kube-janitor)

[![license](https://img.shields.io/github/license/webdevops/kube-janitor.svg)](https://github.com/webdevops/kube-janitor/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fkube--janitor-blue)](https://hub.docker.com/r/webdevops/kube-janitor/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fkube--janitor-blue)](https://quay.io/repository/webdevops/kube-janitor)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kube-janitor)](https://artifacthub.io/packages/search?repo=kube-janitor)

Kubernetes janitor which deletes resources by TTL annotations/labels and static rules written in Golang

By default the janitor uses the creation timestamp from every resource but can also use a custom timestamp by using a JMES path with `timestampPath`.


## Configuration

see [`example.yaml`](example.yaml) for example configurations

```
Usage:
  kube-janitor [OPTIONS]

Application Options:
      --version                                    Show version
      --version.template=                          Version go template, eg {{.Version}}
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
      --kube.itemsperpage=                         Defines how many items per page janitor should process (default: 100) [$KUBE_ITEMSPERPAGE]
      --server.bind=                               Server address (default: :8080) [$SERVER_BIND]
      --server.timeout.read=                       Server read timeout (default: 5s) [$SERVER_TIMEOUT_READ]
      --server.timeout.write=                      Server write timeout (default: 10s) [$SERVER_TIMEOUT_WRITE]

Help Options:
  -h, --help                                       Show this help message
```

## TTL tag

Supported absolute timestamps

- `2006-01-02 15:04:05 +07:00`
- `2006-01-02 15:04:05 MST`
- `2006-01-02 15:04:05`
- `02 Jan 06 15:04 MST` (RFC822)
- `02 Jan 06 15:04 -0700` (RFC822Z)
- `Monday, 02-Jan-06 15:04:05 MST` (RFC850)
- `Mon, 02 Jan 2006 15:04:05 MST` (RFC1123)
- `Mon, 02 Jan 2006 15:04:05 -0700` (RFC1123Z)
- `2006-01-02T15:04:05Z07:00` (RFC3339)
- `2006-01-02T15:04:05.999999999Z07:00` (RFC3339Nano)
- `2006-01-02`

Supported relative timestamps ([`time.Duration`](https://pkg.go.dev/time) and [`fortio.org/duration`](https://github.com/fortio/duration))

- `1m` (minute)
- `1h` (hour)
- `1d` (day)
- `1d6h` (1 day, 6 hours)
- `1w` (1 week)
- `1w2d6h` (1 week, 2 days, 6 hours)

## Metrics

| Metric                                                | Description                                                                                         |
|-------------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| `kube_janitor_resource_deleted_total`                 | Total number of deleted resources (by namespace, gvk, rule)                                         |
| `kube_janitor_resource_ttl_expiry_timestamp_seconds`  | Expiry date (unix timestamp) for every resource which was detected matching the TTL expiry          |
| `kube_janitor_resource_rule_expiry_timestamp_seconds` | Expiry date (unix timestamp) for every resource which was detected matching the static expiry rules |
