package prometheus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// GetURL generates full URL for prometheus API
func GetURL(url string, metricName string, from time.Time, until time.Time) string {
	return fmt.Sprintf("%s/api/v1/query?query=%s", url, metricName)
}

// GetAllTagsValues returns all targets with tags
func GetAllTagsValues(carbonURL string) (map[string][]string, error) {
	return make(map[string][]string, 0), fmt.Errorf("Not implemeted")
}

// GetAllMetrics retuens all metric names from prometheus
func GetAllMetrics(promURL string) ([]string, error) {
	type Metrics struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	url := fmt.Sprintf("%s/api/v1/label/__name__/values", promURL)
	resp, err := http.Get(url)
	if err != nil {
		return make([]string, 0), err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return make([]string, 0), err
	}

	var reply Metrics
	err = json.Unmarshal(body, &reply)
	if err != nil {
		return make([]string, 0), err
	}

	return reply.Data, nil
}
