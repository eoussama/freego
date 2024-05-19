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
	resp_all_games, _ := client.GetGames(enums.FilterAll)
	resp_free_games, _ := client.GetGames(enums.FilterFree)
	resp_approved_games, _ := client.GetGames(enums.FilterApproved)
	// resp_game := client.GetGameDetails(resp_games[0])

	fmt.Println("ping:", resp_ping)
	fmt.Println("all games:", len(resp_all_games))
	fmt.Println("free games:", len(resp_free_games))
	fmt.Println("approved games:", len(resp_approved_games))
	// fmt.Println(resp_game)
}
