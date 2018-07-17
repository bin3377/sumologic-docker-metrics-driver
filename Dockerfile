###############################################################################

FROM  golang as builder

RUN mkdir -p /go/src/github.com/sumologic/sumologic-docker-metrics-plugin
COPY . /go/src/github.com/sumologic/sumologic-docker-metrics-plugin
RUN cd /go/src/github.com/sumologic/sumologic-docker-metrics-plugin && \
  go get && \
  CGO_ENABLED=0 go build -tags netgo -o metrics
RUN mkdir -p /run/docker

###############################################################################

FROM debian as certs

RUN apt-get update &&  \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN cp /etc/ca-certificates.conf /tmp/caconf && cat /tmp/caconf | \
  grep -v "mozilla/CNNIC_ROOT\.crt" > /etc/ca-certificates.conf && \
  update-ca-certificates --fresh

###############################################################################

FROM scratch

COPY --from=builder /go/src/github.com/sumologic/sumologic-docker-metrics-plugin/metrics /metrics
COPY --from=builder /run/docker /run/docker
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/metrics"]
