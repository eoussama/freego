package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/eoussama/freego/core/consts"
	"github.com/eoussama/freego/core/models"
)

func MakeRequest(endpoint string, apiKey string) (*models.Response, error) {
	client := &http.Client{}
	url := consts.Config.Url + "/" + endpoint
	authHeader := "Basic " + apiKey
	fmt.Println("url=", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResponse models.Response
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, err
	}

	return &apiResponse, nil
}
