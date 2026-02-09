package sysinfo

import (
	"strconv"
)

// siParseFloat parses a string as float64, returning 0 on error.
func siParseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
