FROM golang AS plugin
RUN mkdir -p /go/src/github.com/sumologic/sumologic-docker-metrics-plugin
COPY . /go/src/github.com/sumologic/sumologic-docker-metrics-plugin
RUN cd /go/src/github.com/sumologic/sumologic-docker-metrics-plugin && \
  go get && \
  CGO_ENABLED=0 go build -tags netgo -o metrics
RUN mkdir -p /run/docker

FROM scratch
COPY --from=plugin /go/src/github.com/sumologic/sumologic-docker-metrics-plugin/metrics /metrics
COPY --from=plugin /run/docker /run/docker
ENTRYPOINT ["/metrics"]
