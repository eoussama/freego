package consts

import "github.com/eoussama/freego/src/models"

var EndpointPing = models.Endpoint{Fragments: []interface{}{"ping"}}
var EndpointGames = models.Endpoint{Fragments: []interface{}{"games"}}
var EndpointGameInfo = models.Endpoint{Fragments: []interface{}{"game", "?", "info"}}
var EndpointGameAnalytics = models.Endpoint{Fragments: []interface{}{"game", "?", "analytics"}}
