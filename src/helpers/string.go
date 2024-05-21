package helpers

import (
	"strconv"
	"strings"
)

func JoinNumbers(nums []int, separator string) string {
	strNums := make([]string, len(nums))

	for i, id := range nums {
		strNums[i] = strconv.Itoa(id)
	}

	return strings.Join(strNums, separator)
}
