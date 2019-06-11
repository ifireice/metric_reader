package carbon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

// GetURL generates full URL for prometheus API
func GetURL(url string, metricName string, from time.Time, until time.Time) string {
	return fmt.Sprintf("%s/render/?target=%s&from=%v&until=%v&format=json", url, metricName, from.Unix(), until.Unix())
}

func getAllTagNames(carbonURL string) ([]string, error) {
	timeout := time.Duration(60 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	var tagNames []string

	tagsURL := fmt.Sprintf("%s/tags", carbonURL)
	resp, err := client.Get(tagsURL)
	if err != nil {
		return make([]string, 0), err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return make([]string, 0), err
	}

	err = json.Unmarshal(body, &tagNames)
	if err != nil {
		return make([]string, 0), err
	}
	return tagNames, nil
}

func getTagValues(carbonURL string, tagName string) ([]string, error) {
	var tagValues []string
	url := fmt.Sprintf("%s/tags/autoComplete/values?tag=%s", carbonURL, tagName)
	resp, err := http.Get(url)
	if err != nil {
		return tagValues, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return tagValues, err
	}

	err = json.Unmarshal(body, &tagValues)
	if err != nil {
		return tagValues, err
	}
	return tagValues, nil
}

func getExpressionString(currentTags []string) string {
	ret := bytes.NewBuffer([]byte(""))
	for _, m := range currentTags {
		ret.WriteString(fmt.Sprintf("expr=%s&", m))
	}
	return ret.String()
}

func genAutoCompleteURL(baseURL string, complType string, currentTags []string, tag string) string {
	ret := bytes.NewBuffer([]byte(""))
	ret.WriteString(fmt.Sprintf("%s/tags/autoComplete/%s?", baseURL, complType))
	ret.WriteString(getExpressionString(currentTags))

	if tag != "" {
		ret.WriteString(fmt.Sprintf("tag=%s", tag))
	}

	return ret.String()
}

func genAutoComplete(baseURL string, complType string, currentTags []string, tag string) ([]string, error) {
	var tags []string
	resp, err := http.Get(genAutoCompleteURL(baseURL, complType, currentTags, tag))
	if err != nil {
		return tags, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return tags, err
	}

	err = json.Unmarshal(body, &tags)
	if err != nil {
		return tags, err
	}

	return tags, nil
}

func genAutoCompleteTags(baseURL string, currentTags []string) ([]string, error) {
	return genAutoComplete(baseURL, "tags", currentTags, "")
}

func genAutoCompleteValues(baseURL string, currentTags []string, tag string) ([]string, error) {
	return genAutoComplete(baseURL, "values", currentTags, tag)
}

func joinTags(tags []string) string {
	ret := bytes.NewBuffer([]byte(""))

	first := true
	for _, t := range tags {
		if !first {
			ret.WriteString(",")
		}
		ret.WriteString(fmt.Sprintf("'%s'", t))
		first = false
	}
	return ret.String()
}

func getAllMetricsRecurse(baseURL string, currentTags []string) ([]string, error) {
	nextTags, err := genAutoCompleteTags(baseURL, currentTags)
	if err != nil {
		return make([]string, 0), err
	}

	ret := make([]string, 0)
	if len(nextTags) == 0 {
		ret := make([]string, 0)
		return append(ret, joinTags(currentTags)), nil
	}

	for _, tag := range nextTags {
		nextTagsValues, err := genAutoCompleteValues(baseURL, currentTags, tag)
		if err != nil {
			return ret, err
		}

		if len(nextTagsValues) == 0 {
			continue
		}

		for _, tv := range nextTagsValues {
			nextCurrentTags := append(currentTags, fmt.Sprintf("%s=%s", tag, tv))
			nextMetrics, err := getAllMetricsRecurse(baseURL, nextCurrentTags)
			if err != nil {
				return ret, err
			}

			for _, nct := range nextMetrics {
				ret = append(ret, nct)
			}
		}
	}
	fmt.Println(ret)
	return ret, nil
}

// GetAllMetrics returns all targets with tags
func GetAllMetrics(url string) ([]string, error) {
	tagNames, err := getAllTagNames(url)
	metrics := make([]string, 0)
	if err != nil {
		return metrics, err
	}

	for _, tagName := range tagNames {
		tagValues, err := getTagValues(url, tagName)
		if err != nil {
			return metrics, err
		}

		for _, tagValue := range tagValues {
			currentTags := []string{fmt.Sprintf("%s=%s", tagName, tagValue)}
			nextMetrics, err := getAllMetricsRecurse(url, currentTags)
			if err != nil {
				return metrics, err
			}

			for _, nm := range nextMetrics {
				metrics = append(metrics, nm)
			}
		}

	}
	return metrics, nil
}

// GetAllTagsValues returns map with all tags and values
func GetAllTagsValues(carbonURL string) (map[string][]string, error) {
	tagNames, err := getAllTagNames(carbonURL)
	metrics := make(map[string][]string, 0)
	if err != nil {
		return metrics, err
	}

	for _, tagName := range tagNames {
		tagValues, err := getTagValues(carbonURL, tagName)
		if err != nil {
			return metrics, err
		}
		metrics[tagName] = tagValues
	}
	return metrics, nil
}

func getRandom(a []string) string {
	n := rand.Int63n(int64(len(a)))
	return a[n]
}

// GetRandomTags returns ...
func GetRandomTags(baseURL string) (string, error) {
	allTags, err := getAllTagNames(baseURL)
	if err != nil {
		return "", err
	}

	tag := getRandom(allTags)

	tags := make([]string, 0)

	for {
		fmt.Println(tags)
		nextTagsValues, err := genAutoCompleteValues(baseURL, tags, tag)
		if err != nil {
			return "", err
		}

		if len(nextTagsValues) == 0 {
			tagsStr := joinTags(tags)
			return tagsStr, nil
		}

		tagValue := getRandom(nextTagsValues)
		tags = append(tags, fmt.Sprintf("%s=%s", tag, tagValue))

		allTags, err = genAutoCompleteTags(baseURL, tags)
		if err != nil {
			return "", err
		}

		if len(allTags) == 0 {
			tagsStr := joinTags(tags)
			return tagsStr, nil
		}

		tag = getRandom(allTags)
	}
}
