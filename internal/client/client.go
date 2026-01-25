package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

var ErrConnection = errors.New("api connection failed")

func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnection)
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	url := c.BaseURL + path
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnection, err)
	}
	return resp, nil
}

type apiErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func formatAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))

	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil {
		if apiErr.Error.Code != "" || apiErr.Error.Message != "" {
			if apiErr.Error.Code == "" {
				return fmt.Errorf("api error: %d: %s", resp.StatusCode, apiErr.Error.Message)
			}
			return fmt.Errorf("api error: %d %s: %s", resp.StatusCode, apiErr.Error.Code, apiErr.Error.Message)
		}
	}

	msg := strings.TrimSpace(string(body))
	if msg != "" {
		return fmt.Errorf("api error: %d: %s", resp.StatusCode, msg)
	}

	return fmt.Errorf("api error: %d", resp.StatusCode)
}

func expectStatus(resp *http.Response, allowed ...int) error {
	for _, code := range allowed {
		if resp.StatusCode == code {
			return nil
		}
	}
	return formatAPIError(resp)
}

type CommandRecord struct {
	GuildID   string `json:"guildId"`
	Name      string `json:"name"`
	Response  string `json:"response"`
	CreatedAt string `json:"createdAt"`
}

type CommandsListResponse struct {
	Commands []CommandRecord `json:"commands"`
}

type BotCommandResponse struct {
	Response string `json:"response"`
}

func (c *Client) GetCommandResponse(guildID, name string) (string, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/bot/guilds/%s/commands/%s", guildID, name), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // Or custom error
	}
	if err := expectStatus(resp, http.StatusOK); err != nil {
		return "", err
	}

	var cmdResp BotCommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return "", err
	}
	return cmdResp.Response, nil
}

func (c *Client) AddCommand(guildID, name, response string) error {
	body := map[string]string{
		"name":     name,
		"response": response,
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/bot/guilds/%s/commands/add", guildID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusCreated, http.StatusOK); err != nil {
		return err
	}
	return nil
}

func (c *Client) RemoveCommand(guildID, name string) error {
	body := map[string]string{
		"name": name,
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/bot/guilds/%s/commands/remove", guildID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateCommand(guildID, name, response string) error {
	body := map[string]string{
		"name":     name,
		"response": response,
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/bot/guilds/%s/commands/update", guildID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return err
	}
	return nil
}

type BulkCommandInput struct {
	Name     string `json:"name"`
	Response string `json:"response"`
}

func (c *Client) AddBulkCommands(guildID string, commands []BulkCommandInput) error {
	body := map[string]interface{}{
		"commands": commands,
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/bot/guilds/%s/commands/addbulk", guildID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return err
	}
	return nil
}

func (c *Client) ListCommands(guildID string) ([]CommandRecord, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/bot/guilds/%s/commands", guildID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var listResp CommandsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}
	return listResp.Commands, nil
}
