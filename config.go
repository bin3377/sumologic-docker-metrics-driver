package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type sumoConfig struct {
	sumoURL            *url.URL
	pollInterval       time.Duration
	asLog              bool
	sourceCategory     string
	sourceName         string
	sourceHost         string
	metricsIncluded    []string
	metricsExcluded    []string
	intrinsicLabels    []string
	extraIntrinsicTags []string
	extraMetaTags      []string
	tlsConfig          *tls.Config
	proxyURL           *url.URL
}

func readSumoConfigFromEnv() (*sumoConfig, error) {
	var err error
	newConfig := &sumoConfig{}

	if newConfig.sumoURL, err = url.Parse(os.Getenv("SUMO_URL")); err == nil && newConfig.sumoURL.Path != "" {
		fmt.Printf("SUMO_URL was set to '%s'\n", newConfig.sumoURL.Path)
	} else {
		fmt.Printf("SUMO_URL must be a valid URL '%s' - Error: %s", os.Getenv("SUMO_URL"), err.Error())
		return nil, err
	}
	if newConfig.pollInterval, err = time.ParseDuration(os.Getenv("SUMO_POLL_INTERVAL")); err != nil {
		newConfig.pollInterval = 10000 * time.Millisecond
		fmt.Printf("SUMO_POLL_INTERVAL was set to %d (Default) - Error: %s\n", newConfig.pollInterval, err.Error())
	} else {
		fmt.Printf("SUMO_POLL_INTERVAL was set to %d\n", newConfig.pollInterval)
	}
	if newConfig.asLog, err = strconv.ParseBool(os.Getenv("SUMO_AS_LOG")); err != nil {
		newConfig.asLog = false
	}
	fmt.Printf("SUMO_AS_LOG was set to '%t'\n", newConfig.asLog)
	newConfig.sourceCategory = os.Getenv("SUMO_SOURCE_CATEGORY")
	fmt.Printf("SUMO_SOURCE_CATEGORY was set to '%s'\n", newConfig.sourceCategory)
	newConfig.sourceName = os.Getenv("SUMO_SOURCE_NAME")
	fmt.Printf("SUMO_SOURCE_NAME was set to '%s'\n", newConfig.sourceName)
	newConfig.sourceHost = os.Getenv("SUMO_SOURCE_HOST")
	if newConfig.sourceHost == "" {
		if newConfig.sourceHost, err = os.Hostname(); err != nil {
			newConfig.sourceHost = "localhost"
		}
	}
	fmt.Printf("SUMO_SOURCE_HOST was set to '%s'\n", newConfig.sourceHost)

	newConfig.metricsIncluded = split(os.Getenv("SUMO_METRICS_INCLUDED"), ',')
	fmt.Printf("SUMO_METRICS_INCLUDED were set to '%s'\n", newConfig.metricsIncluded)
	newConfig.metricsExcluded = split(os.Getenv("SUMO_METRICS_EXCLUDED"), ',')
	fmt.Printf("SUMO_METRICS_EXCLUDED were set to '%s'\n", newConfig.metricsExcluded)
	newConfig.intrinsicLabels = split(os.Getenv("SUMO_INTRINSIC_LABELS"), ',')
	fmt.Printf("SUMO_INTRINSIC_LABELS were set to '%s'\n", newConfig.intrinsicLabels)
	newConfig.extraIntrinsicTags = split(os.Getenv("SUMO_EXTRA_INTRINSIC_TAGS"), ',')
	fmt.Printf("SUMO_EXTRA_INTRINSIC_TAGS were set to '%s'\n", newConfig.extraIntrinsicTags)
	newConfig.extraMetaTags = split(os.Getenv("SUMO_EXTRA_META_TAGS"), ',')
	fmt.Printf("SUMO_EXTRA_META_TAGS were set to '%s'\n", newConfig.extraMetaTags)

	newConfig.tlsConfig = &tls.Config{}
	if "" != os.Getenv("SUMO_ROOT_CA_PATH") {
		if rootCA, err := ioutil.ReadFile(os.Getenv("SUMO_ROOT_CA_PATH")); err == nil {
			rootCAs := x509.NewCertPool()
			rootCAs.AppendCertsFromPEM(rootCA)
			newConfig.tlsConfig.RootCAs = rootCAs
			fmt.Printf("SUMO_ROOT_CA_PATH was set to '%s'\n", os.Getenv("SUMO_ROOT_CA_PATH"))
		} else {
			fmt.Printf("Error when load SUMO_ROOT_CA_PATH '%s' - %s\n", os.Getenv("SUMO_ROOT_CA_PATH"), err.Error())
		}
	}
	if proxy, err := url.Parse(os.Getenv("SUMO_PROXY_URL")); err == nil {
		newConfig.proxyURL = proxy
		fmt.Printf("SUMO_PROXY_URL was set to '%s'\n", newConfig.proxyURL)
	}

	return newConfig, nil
}

func split(input string, splitor rune) []string {
	f := func(c rune) bool {
		return c == splitor
	}
	return strings.FieldsFunc(input, f)
}
