package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/novapanel/novapanel/internal/models"
)

type Client struct {
	apiURL   string
	token    string
	serverID string
	client   *http.Client
}

func NewClient(apiURL, token, serverID string) *Client {
	return &Client{
		apiURL:   apiURL,
		token:    token,
		serverID: serverID,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *Client) SendHeartbeat(metrics *models.ServerMetrics) error {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/servers/%s/heartbeat", c.apiURL, c.serverID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}
