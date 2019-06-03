package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ifireice/metric_reader/metric_reader/carbon"
	"github.com/ifireice/metric_reader/metric_reader/prometheus"
)

// Source constants
const (
	PROMETHEUS = "Prometheus"
	CARBON     = "Carbon"
)

type requestData struct {
	URL        string
	MetricName string
	Elapsed    time.Duration
	Failed     bool
}

type average struct {
	Elapsed time.Duration
	Count   uint64
}

func generataRandomTagQuery(metrics map[string][]string) string {
	ret := bytes.NewBuffer([]byte(""))

	first := true
	for k, v := range metrics {
		i := rand.Int63n(int64(len(v)))
		if !first {
			ret.WriteString(",")
		}
		ret.WriteString(fmt.Sprintf("'%s=%s'", k, v[i]))
		first = false
	}

	return ret.String()
}

func getFromUntil(minFrom time.Time, period time.Duration) (time.Time, time.Time) {
	minF := minFrom.Unix()
	maxF := time.Now().Unix()

	randUnixTime := rand.Int63n(maxF-minF) + minF
	randTime := time.Unix(randUnixTime, 0)

	from := randTime
	until := randTime.Add(period)

	return from, until
}

func generateRequests(url string, metrics []string, rules []Rule, count uint64, maxPeriod time.Duration, getURLFunc func(string, string, time.Time, time.Time) string, outChan chan requestData) error {
	cache := make(map[string]requestData, 0)
	i := uint64(0)
	for {
		coin := rand.Int63n(4)
		if coin > 0 && len(cache) != 0 {
			keys := make([]string, 0)
			for k := range cache {
				keys = append(keys, k)
			}
			kn := rand.Int63n(int64(len(keys)))
			outChan <- cache[keys[kn]]
			continue
		}

		ruleN := rand.Int63n(int64(len(rules)))
		// metricN := rand.Int63n(int64(len(metrics)))
		metric, err := carbon.GetRandomTags(url)
		if err != nil {
			return err
		}

		query := Template2Metric(rules[ruleN].MetricQueryTemplate, metric)
		fmt.Printf(">>>>>>%s %s %s", metric, query, rules[ruleN])

		minTime := time.Now().Add(-maxPeriod)
		from, until := getFromUntil(minTime, rules[ruleN].Period)

		var request requestData
		request.URL = getURLFunc(url, query, from, until)
		request.MetricName = query
		request.Failed = false

		cache[request.URL] = request
		outChan <- request

		if count > 0 {
			if i >= count {
				break
			}
			i++
		}
	}
	close(outChan)
	return nil
}

func makeHTTPRequest(inChan chan requestData, resultChan chan requestData, doneChan chan bool) {
	timeout := time.Duration(30 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	for {
		request, more := <-inChan
		if !more {
			doneChan <- true
			return
		}

		start := time.Now()
		resp, err := client.Get(request.URL)
		if err != nil {
			fmt.Printf("%s failed\n", request.URL)
			request.Failed = true
			resultChan <- request
			continue
		}

		// body, err := ioutil.ReadAll(resp.Body)
		// _, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%s read failed\n", request.URL)
			request.Failed = true
			resultChan <- request
			continue
		}
		resp.Body.Close()

		fmt.Println("====")
		fmt.Println(request.URL)
		// fmt.Println(string(body))
		fmt.Println("====")

		t := time.Now()
		request.Elapsed = t.Sub(start)
		request.Failed = false

		resultChan <- request
	}
}

func resultPrinter(resultChan chan requestData, doneChan chan bool) {
	for {
		result, more := <-resultChan
		if !more {
			break
		}
		fmt.Printf("%s %s\n", result.URL, result.Elapsed)
	}
	doneChan <- true
}

func resultAverage(resultChan chan requestData, doneChan chan bool) {
	var averageRes average
	metricAvrg := make(map[string]average)
	failedCount := uint64(0)
	for {
		result, more := <-resultChan
		if !more {
			break
		}

		if result.Failed {
			failedCount++
		}

		averageRes.Count++
		averageRes.Elapsed += result.Elapsed

		ma := metricAvrg[result.MetricName]
		ma.Count++
		ma.Elapsed += result.Elapsed
		metricAvrg[result.MetricName] = ma
	}
	fmt.Printf("Average %d nanoseconds\n", (averageRes.Elapsed.Nanoseconds() / int64(averageRes.Count)))
	fmt.Printf("Failed count: %d\n", failedCount)
	fmt.Println()

	for k, v := range metricAvrg {
		fmt.Printf("%s: %d nanoseconds\n", k, (v.Elapsed.Nanoseconds() / int64(v.Count)))
	}

	doneChan <- true
}

func getEnv(name string, defVal string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defVal
	}
	return val
}

type options struct {
	Source        string // "Prometheus" OR "Carbon"
	URL           string
	Count         uint64
	ParallelCount uint64
	RulesPath     string
	PeriodStr     string
}

func main() {
	requestsChan := make(chan requestData)
	resultsChan := make(chan requestData)
	doneChan := make(chan bool)
	defaultPromURL := "http://localhost:9090"

	defaultCount, err := strconv.ParseUint(getEnv("REQ_COUNT", "0"), 10, 64)
	if err != nil {
		defaultCount = 0
	}

	defaultParCount, err := strconv.ParseUint(getEnv("REQ_PAR", "10"), 10, 64)
	if err != nil {
		defaultParCount = 10
	}

	var opts options
	flag.StringVar(&opts.Source, "source", getEnv("SOURCE", CARBON), "Source type: Prometeus OR Carbon")
	flag.StringVar(&opts.URL, "url", getEnv("PROM_URL", defaultPromURL), fmt.Sprintf("URL, default:%s", defaultPromURL))
	flag.Uint64Var(&opts.Count, "count", defaultCount, fmt.Sprintf("Number of requests, default: inf"))
	flag.Uint64Var(&opts.ParallelCount, "parallel", defaultParCount, fmt.Sprintf("Number of parallel requests, default: 10"))
	flag.StringVar(&opts.RulesPath, "rules", getEnv("RULES_PATH", ""), fmt.Sprintf("Path to rules file"))
	flag.StringVar(&opts.PeriodStr, "period", getEnv("PERIOD", "168h"), fmt.Sprintf("Max period for metrics, default 168h (one week)"))
	flag.Parse()

	fmt.Printf("Source:%s\n", opts.Source)
	fmt.Printf("URL:%s\n", opts.URL)
	fmt.Printf("Count:%d\n", opts.Count)
	fmt.Printf("Parallel count:%d\n", opts.ParallelCount)
	fmt.Printf("Rules path:%s\n", opts.RulesPath)
	fmt.Printf("Period:%s\n", opts.PeriodStr)

	if opts.Source != "Prometheus" && opts.Source != "Carbon" {
		panic("Source should be 'Prometheus' OR 'Carbon")
	}

	maxPeriod, err := time.ParseDuration(opts.PeriodStr)
	if err != nil {
		panic(err)
	}

	// getAllMetricsFunc := prometheus.GetAllMetrics
	getURLFunc := prometheus.GetURL
	if opts.Source == CARBON {
		// getAllMetricsFunc = carbon.GetAllMetrics
		getURLFunc = carbon.GetURL
	}

	var rules []Rule
	if opts.RulesPath != "" {
		rules, err = ReadRules(opts.RulesPath)
		if err != nil {
			fmt.Printf("Error while parsing file:%s", opts.RulesPath)
			panic(err)
		}
	} else {
		rules = []Rule{GetDefaultRule()}
	}

	// fmt.Println("Collecting all metrics ...")
	// metrics, err := getAllMetricsFunc(opts.URL)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(metrics)
	// fmt.Println("Collecting all metrics ... DONE")
	metrics := make([]string, 0)

	go generateRequests(opts.URL, metrics, rules, opts.Count, maxPeriod, getURLFunc, requestsChan)

	for i := uint64(0); i < opts.ParallelCount; i++ {
		go makeHTTPRequest(requestsChan, resultsChan, doneChan)
	}

	go resultAverage(resultsChan, doneChan)

	for i := uint64(0); i < opts.ParallelCount; i++ {
		<-doneChan
	}
	close(resultsChan)
	<-doneChan

	close(doneChan)
	fmt.Println("All DONE!")
}
