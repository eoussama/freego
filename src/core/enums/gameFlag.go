package enums

import "github.com/eoussama/freego/core/types"

const (
	GameFlagTrash      types.TGameFlag = 1 << iota
	GameFlagThirdparty types.TGameFlag = 1 << iota
)
