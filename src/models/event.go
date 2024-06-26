package models

import "github.com/eoussama/freego/src/types"

type Event struct {
	Data   []int        `json:"data"`
	Event  types.TEvent `json:"event"`
	Secret string       `json:"secret,omitempty"`
}
