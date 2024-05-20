package helpers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/eoussama/freego/core/models"
)

func MakeRequest(endpoint []interface{}, apiKey string) (*models.Response, error) {
	client := &http.Client{}

	url := GetPath(endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	setHeaders(req, apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response models.Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func setHeaders(req *http.Request, apiKey string) {
	authHeader := "Basic " + apiKey
	req.Header.Set("Authorization", authHeader)
}
