# sumologic-docker-metrics-plugin

## Usage

```bash
# cleanup old installation if necessary
docker plugin disable sumologic/sumologic-docker-metrics-plugin
docker plugin rm sumologic/sumologic-docker-metrics-plugin

# install plugin from docker hub but not enable since we need to set SUMO_URL first
docker plugin install sumologic/sumologic-docker-metrics-plugin --grant-all-permissions --disable
docker plugin set sumologic/sumologic-docker-metrics-plugin:latest SUMO_URL=https://collectors.sumologic.com/receiver/v1/http/XXX
docker plugin enable sumologic/sumologic-docker-metrics-plugin
```