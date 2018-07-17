package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/sdk"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"github.com/tv42/httpunix"
)

var (
	config     *sumoConfig
	mu         sync.Mutex
	mfChan     chan *dto.MetricFamily
	httpClient http.Client
)

func main() {
	fmt.Println(">>>> Start plugin - sumologic-docker-metrics")
	var err error
	config, err = readSumoConfigFromEnv()
	if err != nil {
		panic(err)
	}

	mfChan = make(chan *dto.MetricFamily, 1024)
	go pusher()

	h := sdk.NewHandler(`{"Implements": ["MetricsCollector"]}`)
	handlers(&h)
	err = h.ServeUnix("metrics", 0)
	if err != nil {
		panic(err)
	}

	fmt.Println("<<<< Stop plugin - sumologic-docker-metrics")
}

func handlers(h *sdk.Handler) {
	h.HandleFunc("/MetricsCollector.StartMetrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(">>>>>> Got /MetricsCollector.StartMetrics")
		var err error
		defer func() {
			var res struct{ Err string }
			if err != nil {
				res.Err = err.Error()
			}
			json.NewEncoder(w).Encode(&res)
		}()

		mu.Lock()
		defer mu.Unlock()

		go poller()
	})

	h.HandleFunc("/MetricsCollector.StopMetrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("<<<<<< Got /MetricsCollector.StopMetrics")
		json.NewEncoder(w).Encode(map[string]string{})
	})
}

// Puller - poll metrics data points into mfChan every poll interval
func poller() {
	fmt.Println(">>>>>>>> Start poller")
	defer fmt.Println("<<<<<<<< Stop poller")
	ticker := time.NewTicker(config.pollInterval).C
	for {
		select {
		case <-ticker:
			if err := poll(); err != nil {
				logrus.Errorf("Error when polling data from docker - %s\n", err.Error())
			}
		}
	}
}

// Pusher - push carbon v2 data points to backend from mfChan
func pusher() {
	fmt.Println(">>>>>>>> Start pusher")
	defer fmt.Println("<<<<<<<< Stop pusher")
	if err := ping(); err != nil {
		logrus.Fatalf("Error [%s] when pinging SumoLogic backend\n", err.Error())
	}
	for mf := range mfChan {
		if err := push(mf); err != nil {
			logrus.Errorf("Error when sending to Sumo backend - %s\n", err.Error())
		}
	}
}

func poll() error {
	var err error
	t := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        500 * time.Millisecond,
		ResponseHeaderTimeout: 500 * time.Millisecond,
	}
	t.RegisterLocation("docker", "/run/docker/metrics.sock")

	client := http.Client{
		Transport: t,
	}
	res, err := client.Get("http+unix://docker/metrics")
	if err != nil {
		return err
	}

	prom2json.ParseResponse(res, mfChan)

	return nil
}

func ping() error {
	var reader io.Reader
	reader = strings.NewReader("")
	return postSumo(reader, "text/plain")
}

func push(mf *dto.MetricFamily) error {
	var err error
	var buffer bytes.Buffer

	f := prom2json.NewFamily(mf)
	msgs := toCarbonV2(f)
	if 0 == len(msgs) {
		logrus.Debug("Skipping a MetricFamily with no Value point\n")
		return nil
	}

	for _, msg := range msgs {
		line := msg + "\n"
		if _, err = buffer.WriteString(line); err != nil {
			logrus.Errorf("Error [%s] when writing line: %s\n", err.Error(), line)
		}
		// fmt.Println(line)
	}
	reader := bytes.NewReader(buffer.Bytes())
	var contentType string
	if config.asLog {
		contentType = "text/plain"
	} else {
		contentType = "application/vnd.sumologic.carbon2"
	}
	return postSumo(reader, contentType)
}

func toCarbonV2(f *prom2json.Family) []string {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	res := []string{}
	for _, item := range f.Metrics {
		switch item.(type) {

		case prom2json.Metric:
			m := item.(prom2json.Metric)

			intrinsics := make([]string, len(config.extraIntrinsicTags))
			copy(intrinsics, config.extraIntrinsicTags)
			metas := make([]string, len(config.extraMetaTags))
			copy(metas, config.extraMetaTags)
			appendLabels(m.Labels, &intrinsics, &metas)

			line := strings.TrimSpace(getDataLine(now, m, f.Name, intrinsics, metas))
			if line != "" {
				res = append(res, line)
			}

		default:
			logrus.Debugf("Type '%s' is ignored\n", reflect.TypeOf(item))
		}
	}
	return res
}

func appendLabels(labels map[string]string, pIntrinsics *[]string, pMetas *[]string) {
	s := regexp.MustCompile("\\s")
	for k, v := range labels {
		tag := fmt.Sprintf("%s=%s", k, s.ReplaceAllLiteralString(v, "_"))
		isIntrinsic := len(config.intrinsicLabels) > 0 && any(config.intrinsicLabels, func(p string) bool {
			ret, _ := regexp.MatchString(p, k)
			return ret
		})
		if isIntrinsic {
			*pIntrinsics = append(*pIntrinsics, tag)
		} else {
			*pMetas = append(*pMetas, tag)
		}
	}
}

func getDataLine(ts int64, m prom2json.Metric, name string, intrinsics []string, metas []string) string {
	if !isIncluded("name") {
		logrus.Debugf("Metric %s is filtered out\n", name)
		return ""
	}
	iTags := strings.TrimSpace(strings.Join(append(intrinsics, fmt.Sprintf("metric=%s", name)), " "))
	mTags := strings.TrimSpace(strings.Join(metas, " "))
	value, err := strconv.ParseFloat(m.Value, 64)
	if err != nil {
		logrus.Debugf("Invalid value - %s = '%s'\n", name, m.Value)
		return ""
	}
	var ret string
	if 0 == len(mTags) {
		ret = fmt.Sprintf("%s  %f %d", iTags, value, ts)
	} else {
		ret = fmt.Sprintf("%s  %s %f %d", iTags, mTags, value, ts)
	}
	return ret
}

func any(vs []string, f func(string) bool) bool {
	for _, v := range vs {
		if f(v) {
			return true
		}
	}
	return false
}

func isIncluded(name string) bool {
	inWhiteList := len(config.metricsIncluded) == 0 || any(config.metricsIncluded, func(p string) bool {
		ret, _ := regexp.MatchString(p, name)
		return ret
	})
	inBlackList := len(config.metricsExcluded) > 0 && any(config.metricsExcluded, func(p string) bool {
		ret, _ := regexp.MatchString(p, name)
		return ret
	})
	return inWhiteList && !inBlackList
}

func postSumo(reader io.Reader, contentType string) error {
	var err error
	t := &http.Transport{
		TLSClientConfig: config.tlsConfig,
	}
	if config.proxyURL.Path != "" {
		t.Proxy = http.ProxyURL(config.proxyURL)
	}
	client := &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
	}

	request, err := http.NewRequest("POST", config.sumoURL.String(), reader)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", contentType)
	request.Header.Add("X-Sumo-Category", config.sourceCategory)
	request.Header.Add("X-Sumo-Name", config.sourceName)
	request.Header.Add("X-Sumo-Host", config.sourceHost)
	request.Header.Add("X-Sumo-Client", "docker-metrics-plugin")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("HTTP %s responsed from backend - %s", response.Status, body)
	}
	return nil
}
