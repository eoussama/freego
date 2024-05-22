package helpers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/eoussama/freego/src/models"
)

func MakeRequest(method string, endpoint []interface{}, data interface{}, config *models.Config) (*models.Response, error) {
	client := &http.Client{}
	url := GetPath(endpoint)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	body := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	setHeaders(req, config)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response models.Response
	err = json.Unmarshal(result, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func setHeaders(req *http.Request, config *models.Config) {
	var authHeader string

	if config.IsPartner {
		authHeader = "Partner " + config.ApiKey + " 1"
	} else {
		authHeader = "Basic " + config.ApiKey
	}
	req.Header.Set("Authorization", authHeader)
}
