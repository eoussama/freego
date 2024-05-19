package helpers

import (
	"strconv"
	"strings"

	"github.com/eoussama/freego/core/consts"
)

func GetPath(endpoint []interface{}) string {
	var parts []string

	for _, part := range endpoint {
		switch v := part.(type) {
		case string:
			parts = append(parts, v)
		case int:
			parts = append(parts, strconv.Itoa(v))
		default:
			continue
		}
	}

	return consts.Config.Url + "/" + strings.Join(parts, "/")
}
