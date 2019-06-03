package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"time"
)

// Rule is a parsed rule string
type Rule struct {
	MetricQueryTemplate string
	Period              time.Duration
}

// MakeRule returs Rule from metricQuery and period str
func MakeRule(metricQuery string, periodStr string) *Rule {
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		return nil
	}

	return &Rule{metricQuery, period}
}

// ParseRule parses rule string and returns Rule struct
func ParseRule(rule string) (*Rule, error) {
	ruleRe := regexp.MustCompile("^(.*)\\[(.*)\\]$")

	ruleParsed := ruleRe.FindStringSubmatch(rule)
	if len(ruleParsed) != 3 {
		return nil, fmt.Errorf("Cant parse rule: '%s'", rule)
	}
	metricQueryTemplate := ruleParsed[1]
	periodStr := ruleParsed[2]

	period, err := time.ParseDuration(periodStr)
	if err != nil {
		return nil, err
	}

	ret := Rule{metricQueryTemplate, period}

	return &ret, nil
}

// Template2Metric apply metric name to template
func Template2Metric(metricTemplate string, metricName string) string {
	return fmt.Sprintf(metricTemplate, metricName)
}

// GetDefaultRule returns default rule
func GetDefaultRule() Rule {
	return *MakeRule("%s", "24h")
}

// ReadRules reads rules from file and parse it
func ReadRules(filepath string) ([]Rule, error) {
	ret := make([]Rule, 0)
	file, err := os.Open(filepath)
	if err != nil {
		return ret, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}

		rule, err := ParseRule(scanner.Text())
		if err != nil {
			return ret, err
		}
		ret = append(ret, *rule)
	}
	return ret, nil
}
