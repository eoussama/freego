package main

import (
	"fmt"

	"github.com/eoussama/freego"
	"github.com/eoussama/freego/src/enums"
	"github.com/eoussama/freego/src/models"
)

func main() {

	config := freego.Config(&models.Options{FreestuffPartner: false})

	client, err := freego.Init(config)
	if err != nil {
		panic(fmt.Sprintf("[Init Error] %s", err))
	}

	resp_ping, err := client.Ping()
	if err != nil {
		panic(fmt.Sprintf("[Ping Error] %s", err))
	}

	resp_games_all, err := client.GetGames(enums.FilterAll)
	if err != nil {
		panic(fmt.Sprintf("[All Games Error] %s", err))
	}

	resp_games_free, err := client.GetGames(enums.FilterFree)
	if err != nil {
		panic(fmt.Sprintf("[Free Games Error] %s", err))
	}

	resp_games_approved, err := client.GetGames(enums.FilterApproved)
	if err != nil {
		panic(fmt.Sprintf("[Approved Games Error] %s", err))
	}

	resp_game_info, err := client.GetGameInfo(resp_games_free, []string{"fr-FR", "de-DE"})
	if err != nil {
		panic(fmt.Sprintf("[Game Info Error] %s", err))
	}

	fmt.Println("ping:", resp_ping)
	fmt.Println("all games:", len(resp_games_all))
	fmt.Println("free games:", len(resp_games_free))
	fmt.Println("approved games:", len(resp_games_approved))

	for i, game_info := range resp_game_info {
		fmt.Println("game details info", i, ":", game_info)
	}

	go func() {
		err = client.On(enums.EventFreeGames, func(e *models.Event, err error) {
			if err != nil {
				panic(fmt.Sprintf("[On Free Games Error] %s", err))
			}

			fmt.Println("on free games", e.Data)
		})

		if err != nil {
			panic(fmt.Sprintf("[On Free Games Error] %s", err))
		}
	}()
	select {}
}
