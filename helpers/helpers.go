package helpers

import (
	"encoding/json"
	"strconv"
)

// IntToString converts int64 to string.
func IntToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// ToJsonString converts any value to JSON string.
func ToJsonString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func WithLatestFrom(ch, ch2 chan struct{}) (resCh chan struct{}) {
	resCh = make(chan struct{}, 1)
	results := make([]int, 2)

	go func() {
		for {
			select {
			case <-ch:
				results[0] = 1
			case <-ch2:
				results[1] = 1
			}

			if results[0] == 1 && results[1] == 1 {
				resCh <- struct{}{}
				return
			}
		}
	}()

	return resCh
}
