package helpers

import (
	"strconv"
	"strings"
)

func Join(items []interface{}, separator string) string {
	strItems := make([]string, len(items))

	for i, item := range items {
		switch v := item.(type) {
		case int:
			strItems[i] = strconv.Itoa(v)
		case string:
			strItems[i] = v
		}
	}

	return strings.Join(strItems, separator)
}
