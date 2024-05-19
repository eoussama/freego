package main

import (
	"fmt"
	"os"

	"github.com/eoussama/freego"
	"github.com/eoussama/freego/core/enums"
)

func main() {
	apiKey := os.Getenv("FREESTUFF_API_KEY")
	client := freego.Init(apiKey)

	resp_ping, _ := client.Ping()
	resp_games_all, _ := client.GetGames(enums.FilterAll)
	resp_games_free, _ := client.GetGames(enums.FilterFree)
	resp_games_approved, _ := client.GetGames(enums.FilterApproved)
	resp_game_all, _ := client.GetGame(enums.FilterAll, resp_games_all[0])
	resp_game_info, _ := client.GetGame(enums.FilterInfo, resp_games_all[0])
	resp_game_analytics, _ := client.GetGame(enums.Filteranalytics, resp_games_all[0])

	fmt.Println("ping:", resp_ping)
	fmt.Println("all games:", len(resp_games_all))
	fmt.Println("free games:", len(resp_games_free))
	fmt.Println("approved games:", len(resp_games_approved))
	fmt.Println("game details all:", resp_game_all)
	fmt.Println("game details info:", resp_game_info)
	fmt.Println("game details analytics:", resp_game_analytics)
}
