FROM golang AS plugin
RUN mkdir -p /go/src/github.com/sumologic/sumologic-docker-metrics-driver
COPY . /go/src/github.com/sumologic/sumologic-docker-metrics-driver
RUN cd /go/src/github.com/sumologic/sumologic-docker-metrics-driver && \
  go get && \
  CGO_ENABLED=0 go build -tags netgo -o metrics
RUN mkdir -p /run/docker

FROM scratch
COPY --from=plugin /go/src/github.com/sumologic/sumologic-docker-metrics-driver/metrics /metrics
COPY --from=plugin /run/docker /run/docker
ENTRYPOINT ["/metrics"]
