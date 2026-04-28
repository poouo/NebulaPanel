package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Envelope is the common request/response wrapper carrying ciphertext.
type Envelope struct {
	Data string `json:"data"`
}

// Client is the panel HTTP client with shared comm key.
type Client struct {
	BaseURL string
	CommKey string
	HTTP    *http.Client
}

// New creates a new panel client.
func New(baseURL, commKey string) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		CommKey: commKey,
		HTTP: &http.Client{
			Timeout:   20 * time.Second,
			Transport: tr,
		},
	}
}

// Bootstrap exchanges a per-agent registration token for the shared comm key.
// This is an unencrypted POST because the agent does not yet know the key.
func (c *Client) Bootstrap(token string) (string, int, error) {
	body, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/agent/bootstrap", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NebulaAgent/2.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", 0, fmt.Errorf("panel bootstrap returned %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Code int `json:"code"`
		Data struct {
			CommKey string `json:"comm_key"`
			ID      int    `json:"id"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", 0, fmt.Errorf("invalid bootstrap json: %w", err)
	}
	if out.Code != 0 || out.Data.CommKey == "" {
		return "", 0, fmt.Errorf("bootstrap rejected: %s", out.Message)
	}
	c.CommKey = out.Data.CommKey
	return out.Data.CommKey, out.Data.ID, nil
}

// PostEncrypted encrypts request body and decrypts response. Returns the
// decrypted response payload (may be empty).
func (c *Client) PostEncrypted(path string, payload interface{}) ([]byte, error) {
	plain, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	enc, err := Encrypt(plain, c.CommKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	body, _ := json.Marshal(Envelope{Data: enc})

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NebulaAgent/2.0")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("panel returned %d: %s", resp.StatusCode, string(raw))
	}

	// Standard panel envelope: { "code":0, "data": { "data": "<cipher>" } }
	var outer struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("invalid response json: %w", err)
	}
	if outer.Code != 0 {
		return nil, fmt.Errorf("panel error: %s", string(raw))
	}
	// data may be encrypted envelope or plain object
	var inner Envelope
	if err := json.Unmarshal(outer.Data, &inner); err == nil && inner.Data != "" {
		dec, err := Decrypt(inner.Data, c.CommKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt response: %w", err)
		}
		return dec, nil
	}
	return outer.Data, nil
}
