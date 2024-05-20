<p align="center">
  <img width="100" src="./assets/logo.png">
</p>

<h1 align="center">Freego</h1>
<p align="center">FreeStuff API Go Wrapper.</p>

<p align="center">
    <img src="https://img.shields.io/github/v/tag/eoussama/freego" />
    <img src="https://img.shields.io/github/license/eoussama/freego" />
    <img src="https://img.shields.io/github/languages/code-size/eoussama/freego" />
</p>

## Description

Freego is a go package that wrapps the [FreeStuff API](https://docs.freestuffbot.xyz/).

## Usage

### Prerequisites
* Go `1.22.3` or later
* A [FreeStuff](https://docs.freestuffbot.xyz/) API key.

### Installation

First, ensure you have the Freego package installed.

```sh
go get github.com/eoussama/freego
```

### Environment Variables

The Freego package can be configured using environment variables. Create a `.env` file after the [`.env.example`](./.env.example) or set the following variables in your environment:

```txt
FREEGO_WEBHOOK_PORT=
FREEGO_WEBHOOK_ROUTE=
FREEGO_WEBHOOK_SECRET=
FREEGO_FREESTUFF_API_KEY=
```

### Example

```go
package main

import (
	"fmt"

	"github.com/eoussama/freego"
	"github.com/eoussama/freego/core/enums"
	"github.com/eoussama/freego/core/models"
)

func main() {
	// Configuration (optional)
	config := freego.Config(&models.Options{FreestuffPartner: false})

	// Initialize client
	client, err := freego.Init(config)
	if err != nil {
		panic(fmt.Sprintf("[Init Error] %s", err))
	}

	// Ping the API
	resp_ping, err := client.Ping()
	if err != nil {
		panic(fmt.Sprintf("[Ping Error] %s", err))
	}

	// Get all games
	resp_games_all, err := client.GetGames(enums.FilterAll)
	if err != nil {
		panic(fmt.Sprintf("[All Games Error] %s", err))
	}

	// Get free games
	resp_games_free, err := client.GetGames(enums.FilterFree)
	if err != nil {
		panic(fmt.Sprintf("[Free Games Error] %s", err))
	}

	// Get approved games
	resp_games_approved, err := client.GetGames(enums.FilterApproved)
	if err != nil {
		panic(fmt.Sprintf("[Approved Games Error] %s", err))
	}

	// Get information on the first free game
	resp_game_info, err := client.GetGame(enums.FilterInfo, resp_games_free[0])
	if err != nil {
		panic(fmt.Sprintf("[Game Info Error] %s", err))
	}

	// Print results
	fmt.Println("ping:", resp_ping)
	fmt.Println("all games:", len(resp_games_all))
	fmt.Println("free games:", len(resp_games_free))
	fmt.Println("approved games:", len(resp_games_approved))
	fmt.Println("game details info:", resp_game_info)

	// Event listener for free games
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

```

## Testing

### Webhook

For local testing, the projectcomes with a Docker image that's already set up with [smee](smee.io) for convenient event reception.

```sh
./smee <smee_url> <local_port>
```