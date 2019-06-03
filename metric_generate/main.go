package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type metric struct {
	Timestamp  int64
	MetricName string
	MetricValue int64
	Tags     map[string]string
}

type options struct {
	GraphiteURL        string
	TimeWindow         string
	Step               string
	EnableTags         bool
	CountUniqueMetrics int64
}

func getEnv(name string, defVal string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defVal
	}
	return val
}

func generateTimeStamp (timeWindow string) (startTime time.Time, endTime time.Time) {

	// не поддерживает дни, только ns", "us" (or "µs"), "ms", "s", "m", "h"
	timeNow := time.Now()

	period, err := time.ParseDuration(timeWindow)
	if err != nil {
		log.Panic()
	}

	endTime = timeNow
	startTime = timeNow.Add(-period)

	return startTime, endTime
}

func fillingTagsVariable (filename string, tagsVariable map[string][]string) {
   file, err := os.Open(filename )
	if err != nil {
		panic(err)
	}
	// close fi on exit and check for its returned error
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}
		stringLine := strings.TrimSpace(string(line))

		if len(stringLine) == 0 {
			continue
		}

		tag := strings.Split(stringLine, " ")
        tagsVariableValue := strings.Split(tag[1], ";")

		tagsVariable[tag[0]]= tagsVariableValue

	}
}

func generateTagsForMetric(maxCountTags int64, tagsVariable map[string][]string , testMetric metric) {

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	countTags := random.Int63n(maxCountTags)
	if countTags == 0 { countTags++ }

	tagNames := make([]string, 0, len(tagsVariable))

	for key, _ := range tagsVariable{
        tagNames = append(tagNames,key)
	}

	if int64(len(tagNames)) < countTags {
		countTags = countTags % int64(len(tagNames))
	}

	for i := int64(0); i <= countTags; i++ {

		randIndexTagName := random.Int63n(int64(len(tagNames)))
		if randIndexTagName > 0 { randIndexTagName-- }

		tagName := tagNames[randIndexTagName]

		randIndexTagValue := random.Int63n(int64(len(tagsVariable[tagName])))
		if randIndexTagValue > 0 { randIndexTagValue-- }

		tagValue := tagsVariable[tagName][randIndexTagValue]
		testMetric.Tags[tagName] = tagValue
	}

}

func fillingMetricsVariable (filename string, metricsVariable map[int64][]string) {
	file, err := os.Open(filename )
	if err != nil {
		panic(err)
	}
	// close fi on exit and check for its returned error
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}
		stringLine := strings.TrimSpace(string(line))

		if len(stringLine) == 0 {
			continue
		}

		tag := strings.Split(stringLine, " ")
		tagsVariableValue := strings.Split(tag[1], ";")
        posNumber, err := strconv.ParseInt(tag[0], 10, 64)
		if err != nil {
			log.Panic()
		}
		metricsVariable[posNumber]= tagsVariableValue

	}
}

func generateNameMetric(maxLevel int64, metricsVariable map[int64][]string , testMetric *metric) {

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	levelNameMetric := random.Int63n(maxLevel)
	if levelNameMetric == 0 { levelNameMetric++ }

	if int64(levelNameMetric) > int64(len(metricsVariable)){
		levelNameMetric = levelNameMetric % int64(len(metricsVariable))
	}

	graphiteMetric := bytes.NewBuffer([]byte(""))

	firstIndexMetric := random.Int63n(int64(len(metricsVariable[0])-1))
	graphiteMetric.WriteString(metricsVariable[0][firstIndexMetric])

	for i :=int64(1); i < levelNameMetric; i++ {
		graphiteMetric.WriteString(".")
		if _, exists := metricsVariable[i]; exists {
			indexName := random.Int63n(int64(len(metricsVariable[i]))-1)
			graphiteMetric.WriteString(metricsVariable[i][indexName])
			} else {
				log.Panic()
		}
	}

	testMetric.MetricName = graphiteMetric.String()
}

func generateButchMetric(url string, testMetric metric, startTime time.Time, endTime time.Time, step string, enableTags bool ) {
	stepMetric, err := time.ParseDuration(step)
	if err != nil {
		log.Panic()
	}
	//отладка
	fmt.Printf(metricToString(testMetric, enableTags))
	butchMetric := bytes.NewBuffer([]byte(""))

	for ; startTime.Before(endTime); startTime=startTime.Add(stepMetric) {
		testMetric.Timestamp = startTime.Unix()
		testMetric.MetricValue = generateMetricValue()
		butchMetric.WriteString(metricToString(testMetric, enableTags))
		butchMetric.WriteString("\n")
	}

	sendMetric(url, butchMetric.String())
}

func generateMetricValue () (metricValue int64) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	metricValue = random.Int63n(100000)
	return metricValue
}

func metricToString(metric metric, enableTags bool) string {
	//echo "local.random.diceroll 4 `date +%s`" | nc localhost 2003
	///echo "disk.used;datacenter=dc1;rack=a1;server=web01 42 `date +%s`" | nc localhost 2003
	graphiteMetric := bytes.NewBuffer([]byte(""))

	graphiteMetric.WriteString(metric.MetricName)

	if enableTags {
		graphiteMetric.WriteString(";")
		countTags := len(metric.Tags)
		index := 0
		for tagName, tagValue := range metric.Tags {
			index++
			graphiteMetric.WriteString(tagName)
			graphiteMetric.WriteString("=")
			graphiteMetric.WriteString(tagValue)
			if index < countTags {
				graphiteMetric.WriteString(";")
			}
		}
	}
	graphiteMetric.WriteString(" ")
	graphiteMetric.WriteString(strconv.FormatInt(metric.MetricValue,10))
	graphiteMetric.WriteString(" ")
	graphiteMetric.WriteString(strconv.FormatInt(metric.Timestamp, 10))
	graphiteMetric.WriteString("\n")
	return graphiteMetric.String()
}

func daysToHours(dateString string) (dateStringHours string, err error){
	runesTimeWindow := []rune(dateString)
	datevalue := string(runesTimeWindow[0:len(runesTimeWindow)-1])
	datestr := string(runesTimeWindow[len(runesTimeWindow)-1:])

	if datestr == "d" {

		dateToInt, err := strconv.ParseInt(datevalue, 10, 64)
		 if err != nil {
		 	return "", err
		 }
		return string(strconv.FormatInt(dateToInt * 24, 10)) + "h", nil
	} else if datestr == "h" || datestr == "m" ||  datestr == "s" {
		return dateString, nil
	} else {
		return "", err
	}
}

func generateMetrics(url string, timeWindow string, step string, enableTags bool) {

	testMetric :=  metric{0, "", 0,map[string]string{} }

	if enableTags {
		tagsFileName := "./tests_tag_name.txt"
		tagsVariable := make(map[string][]string, 10)
		fillingTagsVariable(tagsFileName, tagsVariable)

		maxCountTags := int64(20)
		generateTagsForMetric(maxCountTags, tagsVariable, testMetric)

	}

	metricsFileName := "./tests_metric_name.txt"
	metricsVariable := make(map[int64][]string, 10)

	fillingMetricsVariable(metricsFileName, metricsVariable)

	maxLevel := int64(20)
	generateNameMetric(maxLevel, metricsVariable, &testMetric)

	startTime, endTime := generateTimeStamp(timeWindow)

	generateButchMetric(url, testMetric, startTime, endTime, step, enableTags)

	//fmt.Printf(metricToString(testMetric, enableTags))
}

func sendMetric(url string, metrics string ){
	conn, _ := net.Dial("tcp", url)
	fmt.Fprintf(conn, metrics + "\n")
}

func main(){
	defaultTimeWindow := "1h"
	defaultStep := "1m"
	defaultEnableTags := true
	defaultGraphiteUrl := "localhost:2003"
	defaultCountUniqueMetrics := int64(10000)

	var opts options
	flag.StringVar(&opts.TimeWindow, "time_window", getEnv("TIME_WINDOW", defaultTimeWindow), fmt.Sprintf("defaultTimeWindow, default: 1h", defaultTimeWindow))
	flag.StringVar(&opts.GraphiteURL, "graphite_url", getEnv("GRAPHITE_URL", defaultGraphiteUrl), fmt.Sprintf("default graphite url, default: localhost:2003", defaultGraphiteUrl))
	flag.StringVar(&opts.Step, "step", getEnv("STEP", defaultStep), fmt.Sprintf("TODO, default: 1m", defaultStep))
	flag.BoolVar(&opts.EnableTags, "enableTags", defaultEnableTags, fmt.Sprintf("TODO, default: true", defaultEnableTags))
	flag.Int64Var(&opts.CountUniqueMetrics,"countUniqueMetrics", defaultCountUniqueMetrics, fmt.Sprintf("TODO, default: 10000", defaultCountUniqueMetrics))
	flag.Parse()

	fmt.Printf("GraphiteUrl:%s\n", opts.GraphiteURL)
	fmt.Printf("TimeWindow:%s\n", opts.TimeWindow)
	fmt.Printf("Step:%s\n", opts.Step)
	fmt.Printf("EnableTags:%s\n", opts.EnableTags)
	fmt.Printf("CountUniqueMetrics:%d\n", opts.CountUniqueMetrics)


	timeWindow, err := daysToHours(opts.TimeWindow)
	if err != nil {
		log.Panic()
	}


	for i :=int64(0); i < opts.CountUniqueMetrics ; i++{
		generateMetrics(opts.GraphiteURL, timeWindow, opts.Step, opts.EnableTags)
	}
}
