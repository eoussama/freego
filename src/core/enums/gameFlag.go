package enums

import "github.com/eoussama/freego/core/types"

const (
	GameFlagTrash      types.GameFlag = 1 << iota
	GameFlagThirdparty types.GameFlag = 1 << iota
)
