package consts

import "github.com/eoussama/freego/src/models"

var EndpointPing = models.Endpoint{Fragments: []interface{}{"ping"}}
var EndpointGames = models.Endpoint{Fragments: []interface{}{"games"}}
var EndpointGame = models.Endpoint{Fragments: []interface{}{"game", "?"}}
