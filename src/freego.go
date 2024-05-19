package freego

import (
	"errors"

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
	} else if !response.Success {
		return false, errors.New(response.Error)
	}

	return response.Success, nil
}

func (c Client) GetGames(filter string) ([]int, error) {
	endpoint, err := consts.Games.Append(filter).Build()
	if err != nil {
		return make([]int, 0), errors.New("invalid endpoint")
	}

	response, err := helpers.MakeRequest(endpoint, c.ApiKey)
	if err != nil {
		return make([]int, 0), err
	} else if !response.Success {
		return make([]int, 0), errors.New(response.Error)
	}

	if data, ok := response.Data.([]interface{}); ok {
		intData := make([]int, len(data))

		for i, v := range data {
			if intVal, ok := v.(float64); ok {
				intData[i] = int(intVal)
			} else {
				return make([]int, 0), errors.New("expected int but got a different type")
			}
		}

		return intData, nil
	}

	return make([]int, 0), errors.New("data is not of type []int")
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
