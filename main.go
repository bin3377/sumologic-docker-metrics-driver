package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
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
				logrus.Fatalln(err)
			}
		}
	}

}

// Pusher - push carbon v2 data points to backend from mfChan
func pusher() {
	fmt.Println(">>>>>>>> Start pusher")
	defer fmt.Println("<<<<<<<< Stop pusher")
	for mf := range mfChan {
		if err := push(mf); err != nil {
			logrus.Fatalln(err)
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

func push(mf *dto.MetricFamily) error {
	var err error
	t := &http.Transport{
		Proxy:           http.ProxyURL(config.proxyURL),
		TLSClientConfig: config.tlsConfig,
	}

	client := &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
	}
	var buffer bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&buffer, -1)
	if err != nil {
		return err
	}
	defer gzipWriter.Close()

	f := prom2json.NewFamily(mf)
	msgs := toCarbonV2(f)
	for _, msg := range msgs {
		line := msg + "\n"
		fmt.Print(line)
		if _, err := gzipWriter.Write([]byte(line)); err != nil {
			logrus.Errorln(err)
		}
	}
	request, err := http.NewRequest("POST", config.sumoURL.String(), bytes.NewBuffer(buffer.Bytes()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Encoding", "gzip")
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

			line := getDataLine(now, m, f.Name, intrinsics, metas)
			res = append(res, line)

		default:
			logrus.Infof("Type '%s' is ignored\n", reflect.TypeOf(item))
		}
	}
	return res
}

func appendLabels(labels map[string]string, pIntrinsics *[]string, pMetas *[]string) {
	for k, v := range labels {
		tag := fmt.Sprintf("%s=%s", k, v)
		if _, exist := config.intrinsicLabels[k]; exist {
			*pIntrinsics = append(*pIntrinsics, tag)
		} else {
			*pMetas = append(*pMetas, tag)
		}
	}
}

func getDataLine(ts int64, m prom2json.Metric, name string, intrinsics []string, metas []string) string {
	iTags := strings.Join(append(intrinsics, fmt.Sprintf("metric=%s", name)), " ")
	mTags := strings.Join(metas, " ")
	return fmt.Sprintf("%s  %s %s %d", iTags, mTags, m.Value, ts)
}
