package freego

import (
	"errors"
	"fmt"

	"github.com/eoussama/freego/core/consts"
	"github.com/eoussama/freego/core/helpers"
)

type Client struct {
	ApiKey string
}

func Init(apiKey string) Client {
	return Client{ApiKey: apiKey}
}

func (c Client) Ping() (bool, error) {
	endpoint, err := consts.EndpointPing.Build()
	if err != nil {
		return false, errors.New("invalid endpoint")
	}

	response, err := helpers.MakeRequest(endpoint, c.ApiKey)
	if err != nil {
		return false, err
	}

	return response.Success, nil
}

func (c Client) GetGames(filter string) []int {
	endpoint, err := consts.Games.Append(filter).Build()
	if err != nil {
		panic("dddd")
	}

	response, err := helpers.MakeRequest(endpoint, c.ApiKey)
	if err != nil {
		return make([]int, 0)
	}

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
	endpoint, err := consts.GameDetailsInfo.Build(gameId)
	if err != nil {
		panic("dddd")
	}

	response, err := helpers.MakeRequest(endpoint, c.ApiKey)
	if err != nil {
		return make([]int, 0)
	}

	return response.Data
}
