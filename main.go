package freego

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/eoussama/freego/src/consts"
	"github.com/eoussama/freego/src/helpers"
	"github.com/eoussama/freego/src/models"
	"github.com/eoussama/freego/src/types"
)

type Client struct {
	Config *models.Config
}

func Config(options *models.Options) *models.Config {
	return models.Config{}.Build(options)
}

func Init(config ...*models.Config) (*Client, error) {
	switch len(config) {
	case 0:
		var config = models.Config{}.Build(&models.Options{})
		return &Client{Config: config}, nil
	case 1:
		return &Client{Config: config[0]}, nil
	default:
		return nil, errors.New("too many arguments")
	}
}

func (c Client) Ping() (bool, error) {
	endpoint, err := consts.EndpointPing.Prepend(c.Config.Url).Build()
	if err != nil {
		return false, errors.New("invalid endpoint")
	}

	response, err := helpers.MakeRequest(endpoint, c.Config)
	if err != nil {
		return false, err
	} else if !response.Success {
		return false, errors.New(response.Error)
	}

	return response.Success, nil
}

func (c Client) GetGames(filter types.TFilter) ([]int, error) {
	endpoint, err := consts.EndpointGames.Prepend(c.Config.Url).Append(filter).Build()

	if err != nil {
		return make([]int, 0), errors.New("invalid endpoint")
	}

	response, err := helpers.MakeRequest(endpoint, c.Config)
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

func (c Client) GetGame(filter types.TFilter, gameId int) (*models.GameInfo, error) {
	endpoint, err := consts.EndpointGame.Prepend(c.Config.Url).Append(filter).Build(gameId)
	if err != nil {
		return nil, err
	}

	response, err := helpers.MakeRequest(endpoint, c.Config)
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

func (c Client) On(event types.TEvent, callback func(*models.Event, error)) error {
	handler := func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			callback(nil, errors.New("invalid request method"))
			return
		}

		var reqBody models.Event
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			callback(nil, errors.New("bad request"))
			return
		}

		if reqBody.Secret == c.Config.Secret {
			callback(&reqBody, nil)
		}
	}

	http.HandleFunc(c.Config.Route, handler)
	return http.ListenAndServe(":"+c.Config.Port, nil)
}
