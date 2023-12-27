package misc

import (
	"strconv"
	"strings"
)

func FirstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return "" // sorry :(
}

func FirstNonNegative(sx ...int64) int64 {
	for _, x := range sx {
		if x >= 0 {
			return x
		}
	}
	return -1 // sorry :(
}

func ToInt64(s string) int64 {
	x, err := strconv.Atoi(s)
	if err != nil {
		return int64(-1)
	}
	return int64(x)
}

func Concat(ss ...string) string {
	return strings.Join(ss, "")
}
