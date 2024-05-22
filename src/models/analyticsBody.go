package models

import "github.com/eoussama/freego/src/types"

type AnalyticsBody struct {
	Data    any            `json:"data"`
	Suid    uint           `json:"suid"`
	Service types.TService `json:"service"`
}
