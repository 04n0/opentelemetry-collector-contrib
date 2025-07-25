# RabbitMQ Receiver

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [beta]: metrics   |
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Areceiver%2Frabbitmq%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Areceiver%2Frabbitmq) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Areceiver%2Frabbitmq%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Areceiver%2Frabbitmq) |
| Code coverage | [![codecov](https://codecov.io/github/open-telemetry/opentelemetry-collector-contrib/graph/main/badge.svg?component=receiver_rabbitmq)](https://app.codecov.io/gh/open-telemetry/opentelemetry-collector-contrib/tree/main/?components%5B0%5D=receiver_rabbitmq&displayType=list) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@VenuEmmadi](https://www.github.com/VenuEmmadi) |
| Emeritus      | [@cpheps](https://www.github.com/cpheps) |

[beta]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
<!-- end autogenerated section -->

This receiver fetches stats from a RabbitMQ node using the [RabbitMQ Management Plugin](https://www.rabbitmq.com/management.html).

> :construction: This receiver is in **BETA**. Configuration fields and metric data model are subject to change.
## Prerequisites

This receiver supports RabbitMQ versions `3.8` and `3.9`.

The RabbitMQ Management Plugin must be enabled by following the [official instructions](https://www.rabbitmq.com/management.html#getting-started).

Also, a user with at least [monitoring](https://www.rabbitmq.com/management.html#permissions) level permissions must be used for monitoring.

## Configuration

The following settings are required:
- `username`
- `password`

The following settings are optional:

- `endpoint` (default: `http://localhost:15672`): The URL of the node to be monitored.
- `collection_interval` (default = `10s`): This receiver collects metrics on an interval. Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.
- `tls`: TLS control. [By default, insecure settings are rejected and certificate verification is on](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md).

### Example Configuration

```yaml
receivers:
  rabbitmq:
    endpoint: http://localhost:15672
    username: otelu
    password: ${env:RABBITMQ_PASSWORD}
    collection_interval: 10s
    metrics:  # Enable node metrics by explicitly setting them to true
      rabbitmq.node.disk_free:
        enabled: true
      rabbitmq.node.disk_free_limit:
        enabled: true
      rabbitmq.node.disk_free_alarm:
        enabled: true
      rabbitmq.node.mem_used:
        enabled: true
      rabbitmq.node.mem_limit:
        enabled: true
      rabbitmq.node.mem_alarm:
        enabled: true
      rabbitmq.node.fd_used:
        enabled: true
      rabbitmq.node.fd_total:
        enabled: true
      rabbitmq.node.sockets_used:
        enabled: true
      rabbitmq.node.sockets_total:
        enabled: true
      rabbitmq.node.proc_used:
        enabled: true
      rabbitmq.node.proc_total:
        enabled: true
      rabbitmq.node.disk_free_details.rate:
        enabled: true
      rabbitmq.node.fd_used_details.rate:
        enabled: true
      rabbitmq.node.mem_used_details.rate:
        enabled: true
      rabbitmq.node.proc_used_details.rate:
        enabled: true
      rabbitmq.node.sockets_used_details.rate:
        enabled: true
```

The full list of settings exposed for this receiver are documented in [config.go](./config.go) with detailed sample configurations in [testdata/config.yaml](./testdata/config.yaml). TLS config is documented further under the [opentelemetry collector's configtls package](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md).

## Metrics

This receiver collects RabbitMQ node-level metrics from the `/api/nodes` endpoint. Metrics are categorized into the following groups:

- **Memory usage**: `rabbitmq.node.mem_used`, `rabbitmq.node.mem_limit`, `rabbitmq.node.mem_alarm`
- **Disk space**: `rabbitmq.node.disk_free`, `rabbitmq.node.disk_free_limit`, `rabbitmq.node.disk_free_alarm`
- **File descriptors & sockets**: `rabbitmq.node.fd_used`, `rabbitmq.node.sockets_used`, etc.
- **Process & scheduling**: `rabbitmq.node.proc_used`, `rabbitmq.node.run_queue`, `rabbitmq.node.context_switches`
- **Garbage collection & I/O**: `rabbitmq.node.gc.num`, `rabbitmq.node.io_read_avg_time`, etc.
- **Cluster & node metadata**: `rabbitmq.node.uptime`, `rabbitmq.node.processors`, etc.

Details about the metrics produced by this receiver and full list of supported metrics can be found in [metadata.yaml](./metadata.yaml)