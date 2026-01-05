package ui

import (
	"fmt"
	"strconv"
)

func IntToStr(n int) string {
	return strconv.Itoa(n)
}

func FormatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
