package main

import (
	"regexp"
	"strings"
)

type pattern struct {
	re     *regexp.Regexp
	negate bool
}

const patternNegatePrefix = "_"

func newPattern(s string) (pattern, error) {
	var p pattern

	if after, ok := strings.CutPrefix(s, patternNegatePrefix); ok {
		s = after
		p.negate = true
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return p, err
	}
	p.re = re

	return p, nil
}

func (p pattern) match(s string) bool {
	return p.negate != p.re.MatchString(s)
}
