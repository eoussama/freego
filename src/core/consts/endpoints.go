package consts

import "github.com/eoussama/freego/core/models"

var EndpointPing = models.Endpoint{Fragments: []interface{}{"ping"}}

var AllGames = models.Endpoint{Fragments: []interface{}{"games", "all"}}
var EndpointFreeGames = models.Endpoint{Fragments: []interface{}{"games", "free"}}
var ApprovedGames = models.Endpoint{Fragments: []interface{}{"games", "approved"}}

var GameDetailsAll = models.Endpoint{Fragments: []interface{}{"game", "?", "all"}}
var GameDetailsInfo = models.Endpoint{Fragments: []interface{}{"game", "?", "info"}}
var GameDetailsAnalytics = models.Endpoint{Fragments: []interface{}{"game", "?", "analytics"}}
