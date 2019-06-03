package main

import (
	"testing"
)

func CheckParseRule(rule string, expected *Rule, t *testing.T) {
	parsed, err := ParseRule(rule)
	if err != nil {
		t.Error(err)
		panic(err)
	}

	if *expected != *parsed {
		t.Errorf("Not expected result: %s != %s", *parsed, expected)
	}
}

func CheckParseBadRule(rule string, t *testing.T) {
	_, err := ParseRule(rule)
	if err == nil {
		t.Errorf("Error shuold not be mil for rule '%s'", rule)
	}
}

func TestParseRuleOk(t *testing.T) {
	CheckParseRule("test(%s)[1m]", MakeRule("test(%s)", "1m"), t)
	CheckParseRule("MySuperTest(%s)[1s]", MakeRule("MySuperTest(%s)", "1s"), t)
	CheckParseRule("%s[1m]", MakeRule("%s", "1m"), t)
	CheckParseRule("%s[48h]", MakeRule("%s", "48h"), t)
}

func TestParseRuleBad(t *testing.T) {
	CheckParseBadRule("", t)
	CheckParseBadRule("qweqweqwe", t)
	CheckParseBadRule("", t)
	CheckParseBadRule("haha[1m", t)
	CheckParseBadRule("haha1m]", t)
	CheckParseBadRule("haha[1mqwerqwerqw]", t)
}
