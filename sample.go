package main

import (
	"fmt"
	"net/http"
	"strings"
)

func login(client *http.Client) error {
	reqBody := strings.NewReader("username=foo&password=bar")

	req, _ := http.NewRequest("POST", "https://example.com/login", reqBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %d", resp.StatusCode)
	}

	return nil
}

func doRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Detect expired session (depends on API behavior)
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// refresh cookie
		if err := login(client); err != nil {
			return nil, err
		}

		// retry request
		return client.Do(req)
	}

	return resp, nil
}

type APIClient struct {
	client *http.Client
}

func (c *APIClient) ensureAuth() error {
	// optionally track last login time
	return login(c.client)
}

func (c *APIClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		if err := c.ensureAuth(); err != nil {
			return nil, err
		}

		return c.client.Do(req)
	}

	return resp, nil
}
