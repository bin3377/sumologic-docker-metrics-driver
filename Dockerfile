###############################################################################

FROM  golang as builder

WORKDIR /go/src/github.com/sumologic/sumologic-docker-metrics-plugin
COPY . .

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

RUN go get && \
    CGO_ENABLED=0 go build -tags netgo -o sumologic-docker-metrics-plugin && \
    mkdir -p /run/docker

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

COPY --from=builder /go/src/github.com/sumologic/sumologic-docker-metrics-plugin/sumologic-docker-metrics-plugin /usr/bin/sumologic-docker-metrics-plugin
COPY --from=builder /run/docker /run/docker
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
