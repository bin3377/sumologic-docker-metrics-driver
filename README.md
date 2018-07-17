# SumoLogic Docker Metrics Plugin

Experimental

## Installation

```bash
# cleanup old installation if necessary
docker plugin disable sumologic/sumologic-docker-metrics-plugin
docker plugin rm sumologic/sumologic-docker-metrics-plugin

# install plugin from docker hub but not enable since we need to set SUMO_URL first
docker plugin install sumologic/sumologic-docker-metrics-plugin --grant-all-permissions --disable
docker plugin set sumologic/sumologic-docker-metrics-plugin:latest SUMO_URL=https://collectors.sumologic.com/receiver/v1/http/XXX
docker plugin enable sumologic/sumologic-docker-metrics-plugin
```

## Supported Parameters

Parameters can be configured with `docker plugin set` Cli:

| Option                      | Description
| --------------------------- | ----------------------------------------------------- |
| `SUMO_URL`                  | Sumo Logic HTTP source URL (required)
| `SUMO_POLL_INTERVAL`        | The time interval for polling metrics (default: `2s`)
| `SUMO_SOURCE_CATEGORY`      | Override source category
| `SUMO_SOURCE_NAME`          | Override source name
| `SUMO_SOURCE_HOST`          | Override source host
| `SUMO_METRICS_INCLUDED`     | Only the metrics matching these RegEx pattern(s) will be included; split by comma
| `SUMO_METRICS_EXCLUDED`     | The metrics matching these RegEx pattern(s) will be ignored; split by comma
| `SUMO_INTRINSIC_LABELS`     | The labels matching these RegEx pattern(s) will be intrinsic, others will be meta; split by comma
| `SUMO_EXTRA_INTRINSIC_TAGS` | Addtional intrinsic tags in format `%k=%v`; split by comma
| `SUMO_EXTRA_META_TAGS`      | Addtional meta tags in format `%k=%v`; split by comma
| `SUMO_PROXY_URL`            | URL of proxy server if applying
