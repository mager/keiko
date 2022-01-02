package sweeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

type SweeperClient struct {
	httpClient *http.Client
	logger     *zap.SugaredLogger
}

// ProvideSweeper provides an HTTP client
func ProvideSweeper(logger *zap.SugaredLogger) SweeperClient {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	return SweeperClient{
		httpClient: &http.Client{
			Transport: tr,
		},
		logger: logger,
	}
}

var Options = ProvideSweeper

type UpdateResp struct {
	Success bool `json:"success"`
}

// AddCollection adds a collection to the database
func (s *SweeperClient) AddCollection(slug string) bool {
	u, err := url.Parse("https://sweeper.floor.report/update")
	if err != nil {
		s.logger.Error(err)
		return false
	}
	q := u.Query()
	u.RawQuery = q.Encode()

	var jsonStr = []byte(fmt.Sprintf("{\"slug\": \"%s\"}", slug))
	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonStr))
	if err != nil {
		s.logger.Error(err)
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(err)
		return false
	}
	defer resp.Body.Close()

	var updateResp UpdateResp
	err = json.NewDecoder(resp.Body).Decode(&updateResp)
	if err != nil {

		s.logger.Error(err)
		return false
	}

	time.Sleep(time.Millisecond * 250)

	return true
}
