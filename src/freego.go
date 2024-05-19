package freego

import (
	"errors"
	"strconv"

	"github.com/eoussama/freego/core/consts"
	"github.com/eoussama/freego/core/helpers"
	"github.com/eoussama/freego/core/models"
	"github.com/eoussama/freego/core/types"
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

func (c Client) GetGames(filter types.Filter) ([]int, error) {
	endpoint, err := consts.EndpointGames.Append(filter).Build()

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

func (c Client) GetGame(filter types.Filter, gameId int) (*models.GameInfo, error) {
	endpoint, err := consts.EndpointGame.Append(filter).Build(gameId)
	if err != nil {
		return nil, err
	}

	response, err := helpers.MakeRequest(endpoint, c.ApiKey)
	if err != nil {
		return nil, err
	} else if !response.Success {
		return nil, errors.New(response.Error)
	}

	if responseData, ok := response.Data.(map[string]interface{}); ok {
		var key string = strconv.Itoa(gameId)
		var data map[string]interface{} = responseData[key].(map[string]interface{})

		result, err := models.GameInfo{}.From(data)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else {
		return nil, errors.New("invalid payload")
	}
}
