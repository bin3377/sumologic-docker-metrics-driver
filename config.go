package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type sumoConfig struct {
	sumoURL            *url.URL
	sourceCategory     string
	sourceName         string
	sourceHost         string
	intrinsicLabels    map[string]interface{}
	extraIntrinsicTags []string
	extraMetaTags      []string
	raw                bool
	compress           bool
	compressLevel      int
	pollInterval       time.Duration
	tlsConfig          *tls.Config
	proxyURL           *url.URL
}

func readSumoConfigFromEnv() (*sumoConfig, error) {
	var err error
	newConfig := &sumoConfig{}

	if newConfig.sumoURL, err = url.Parse(os.Getenv("SUMO_URL")); err != nil {
		return nil, err
	}
	newConfig.sourceCategory = os.Getenv("SUMO_SOURCE_CATEGORY")
	newConfig.sourceName = os.Getenv("SUMO_SOURCE_NAME")
	newConfig.sourceHost = os.Getenv("SUMO_SOURCE_HOST")
	labels := strings.Split(os.Getenv("SUMO_INTRINSIC_LABELS"), ",")
	newConfig.intrinsicLabels = make(map[string]interface{})
	for _, label := range labels {
		newConfig.intrinsicLabels[label] = true
	}
	newConfig.extraIntrinsicTags = strings.Split(os.Getenv("SUMO_EXTRA_INTRINSIC_TAGS"), ",")
	newConfig.extraMetaTags = strings.Split(os.Getenv("SUMO_EXTRA_META_TAGS"), ",")
	if newConfig.sourceHost == "" {
		if newConfig.sourceHost, err = os.Hostname(); err != nil {
			newConfig.sourceHost = "localhost"
		}
	}

	if newConfig.compress, err = strconv.ParseBool(os.Getenv("SUMO_RAW")); err != nil {
		newConfig.compress = false
	}
	if newConfig.compress, err = strconv.ParseBool(os.Getenv("SUMO_COMPRESS")); err != nil {
		newConfig.compress = true
	}
	if newConfig.compressLevel, err = strconv.Atoi(os.Getenv("SUMO_COMPRESS_LEVEL")); err != nil {
		newConfig.compressLevel = -1
	}
	if newConfig.pollInterval, err = time.ParseDuration(os.Getenv("SUMO_POLL_INTERVAL")); err != nil {
		newConfig.pollInterval = 10000 * time.Millisecond
	}
	newConfig.tlsConfig = &tls.Config{}
	if rootCA, err := ioutil.ReadFile(os.Getenv("SUMO_ROOT_CA_PATH")); err == nil {
		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(rootCA)
		newConfig.tlsConfig.RootCAs = rootCAs
	}
	if newConfig.sumoURL, err = url.Parse(os.Getenv("SUMO_PROXY_URL")); err != nil {
		return nil, err
	}

	return newConfig, nil
}
