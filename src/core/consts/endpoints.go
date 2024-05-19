package consts

import "github.com/eoussama/freego/core/models"

var EndpointPing = models.Endpoint{Fragments: []interface{}{"ping"}}

var Games = models.Endpoint{Fragments: []interface{}{"games"}}

var GameDetailsAll = models.Endpoint{Fragments: []interface{}{"game", "?", "all"}}
var GameDetailsInfo = models.Endpoint{Fragments: []interface{}{"game", "?", "info"}}
var GameDetailsAnalytics = models.Endpoint{Fragments: []interface{}{"game", "?", "analytics"}}
