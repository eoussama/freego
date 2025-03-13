package freego

import (
	"encoding/json"
	"errors"
	"io"
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

	response, err := helpers.MakeRequest(http.MethodGet, endpoint, nil, c.Config)
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

	response, err := helpers.MakeRequest(http.MethodGet, endpoint, nil, c.Config)
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

func (c Client) GetGameInfo(gameIds []int, languages []string) ([]*models.GameInfo, error) {
	const batchSize = 5
	var allResults []*models.GameInfo

	if len(gameIds) == 0 {
		return nil, errors.New("missing game id(s)")
	}

	for i := 0; i < len(gameIds); i += batchSize {
		end := i + batchSize
		if end > len(gameIds) {
			end = len(gameIds)
		}

		batch := gameIds[i:end]
		idsStr := helpers.Join(helpers.IntToInterfaceSlice(batch), "+")
		langs := helpers.Join(helpers.StringToInterfaceSlice(languages), "+")

		endpoint, err := consts.EndpointGameInfo.Prepend(c.Config.Url).Append("?lang=" + langs).Build(idsStr)
		if err != nil {
			return nil, err
		}

		response, err := helpers.MakeRequest(http.MethodGet, endpoint, nil, c.Config)
		if err != nil {
			return nil, err
		} else if !response.Success {
			return nil, errors.New(response.Error)
		}

		if responseData, ok := response.Data.(map[string]any); ok {
			for _, id := range batch {
				key := strconv.Itoa(id)

				data, ok := responseData[key].(map[string]any)
				if !ok {
					continue
				}

				result, err := models.GameInfo{}.From(data)
				if err != nil {
					return nil, err
				}

				allResults = append(allResults, result)
			}
		} else {
			return nil, errors.New("invalid payload")
		}
	}

	return allResults, nil
}

func (c Client) GetGameAnalytics(gameId int, serviceId uint, service types.TService, data any) (any, error) {
	if !c.Config.IsPartner {
		return nil, errors.New("unauthorized endpoint")
	}

	id := strconv.Itoa(gameId)

	endpoint, err := consts.EndpointGameAnalytics.Prepend(c.Config.Url).Build(id)
	if err != nil {
		return nil, err
	}

	body := models.AnalyticsBody{
		Data:    data,
		Service: service,
		Suid:    serviceId,
	}

	response, err := helpers.MakeRequest(http.MethodPost, endpoint, body, c.Config)
	if err != nil {
		return nil, err
	} else if !response.Success {
		return nil, errors.New(response.Error)
	}

	return models.AnalyticsResponse{Success: true}, nil
}

func (c Client) GetEvent(body io.ReadCloser) (*models.Event, error) {
	var event models.Event

	if err := json.NewDecoder(body).Decode(&event); err != nil {
		return nil, errors.New("bad request")
	}

	if event.Secret != c.Config.Secret {
		return nil, errors.New("unauthorized source")
	}

	return &event, nil
}

func (c Client) On(event types.TEvent, callback func(*models.Event, error)) error {
	handler := func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			callback(nil, errors.New("invalid request method"))
			return
		}

		result, err := c.GetEvent(r.Body)
		if err != nil {
			callback(nil, err)
		}

		if event == result.Event {
			callback(result, nil)
		}
	}

	http.HandleFunc(c.Config.Route, handler)
	return http.ListenAndServe(":"+c.Config.Port, nil)
}
