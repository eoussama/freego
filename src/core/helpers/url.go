package helpers

import (
	"strconv"
	"strings"

	"github.com/eoussama/freego/core/types"
)

func GetPath(fragments []interface{}) string {
	var parts []string

	for _, fragment := range fragments {
		switch v := fragment.(type) {
		case string:
			parts = append(parts, string(v))
		case types.TFilter:
			parts = append(parts, string(v))
		case int:
			parts = append(parts, strconv.Itoa(v))
		default:
			continue
		}
	}

	return strings.Join(parts, "/")
}
