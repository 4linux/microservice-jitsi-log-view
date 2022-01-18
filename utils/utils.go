package utils

import (
	"fmt"
	"strconv"
	"time"

	"microservice-jitsi-log-view/types"
)

// FindClosestTimeTo finds the Jitsilog whose Timestamp
// is the closest to the passed Duration.
func FindClosestTimeTo(dur time.Duration, logs types.JitsilogIterator) *types.Jitsilog {
	// helper abs function because Go doesn't fucking provide a
	// decent abs() for integers
	abs := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	var closest *types.Jitsilog
	offset := 24 * time.Hour

	for log := range logs {
		h, m, s := log.GetTime().Clock()
		logDur, _ := time.ParseDuration(fmt.Sprintf("%dh%dm%ds", h, m, s))
		tmpOffset := abs(logDur - dur)
		if tmpOffset < offset {
			offset = tmpOffset
			closest = log
		}
	}

	return closest
}

// convertToInt tries to convert all strings passed to int64,
// returning a slice of int64 on success. If any conversion fails,
// the error is returned alongside the index of the non-integer string.
func ConvertToInt(values []string) ([]int64, int, error) {
	ints := make([]int64, len(values))

	for idx, v := range values {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, idx, err
		}
		ints[idx] = i
	}

	return ints, -1, nil
}
