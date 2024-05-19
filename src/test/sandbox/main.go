package main

import (
	"fmt"
	"os"

	freego "github.com/eoussama/freego"
)

func main() {
	apiKey := os.Getenv("FREESTUFF_API_KEY")
	client := freego.Init(apiKey)

	resp_ping := client.Ping()
	resp_games := client.GetGames()
	resp_game := client.GetGameDetails(resp_games[0])

	fmt.Println(resp_ping)
	fmt.Println(resp_games)
	fmt.Println(resp_game)
}
