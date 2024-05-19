package freego

import (
	"fmt"

	"github.com/eoussama/freego/core/helpers"
)

// func Test() {
// 	// Public API request
// 	publicAuthHeader := fmt.Sprintf("Basic %s", apiKey)
// 	publicResponse, err := makeRequest("GET", publicPingURL, publicAuthHeader)
// 	if err != nil {
// 		fmt.Println("Error making public API request:", err)
// 		os.Exit(1)
// 	}
// 	fmt.Println("Public API Response:", publicResponse)

// 	// Partner API request
// 	partnerAuthHeader := fmt.Sprintf("Partner %s %s", apiKey, serviceUID)
// 	partnerResponse, err := makeRequest("GET", partnerPingURL, partnerAuthHeader)
// 	if err != nil {
// 		fmt.Println("Error making partner API request:", err)
// 		os.Exit(1)
// 	}
// 	fmt.Println("Partner API Response:", partnerResponse)
// }

type Client struct {
	ApiKey string
}

func Init(apiKey string) Client {
	return Client{ApiKey: apiKey}
}

func (c Client) Ping() any {
	response, err := helpers.MakeRequest("ping", c.ApiKey)
	if err != nil {
		return false
	}

	return response.Data
}

func (c Client) GetGames() []int {
	response, err := helpers.MakeRequest("games/free", c.ApiKey)
	if err != nil {
		return make([]int, 0)
	}

	// Use type assertion to check and convert the Data field to []int
	if data, ok := response.Data.([]interface{}); ok {
		intData := make([]int, len(data))
		for i, v := range data {
			if floatVal, ok := v.(float64); ok {
				intData[i] = int(floatVal)
			} else {
				fmt.Println("Error: Expected float64 but got a different type")
				return make([]int, 0)
			}
		}
		return intData
	}

	fmt.Println("Error: Data is not of type []int")
	return make([]int, 0)
}

func (c Client) GetGameDetails(gameId int) any {
	s := fmt.Sprintf("%d", gameId)

	response, err := helpers.MakeRequest("game/"+s+"/info", c.ApiKey)
	if err != nil {
		return make([]int, 0)
	}

	return response.Data
}
