package codeforces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/e0-7/cfx/internal/model"
)

const defaultAPI = "https://codeforces.com/api"

type Client struct {
	BaseURL string
	HTTP    *http.Client
	mu      sync.Mutex
	last    time.Time
}

func NewClient() *Client {
	return &Client{
		BaseURL: defaultAPI,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

type apiResponse[T any] struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
	Result  T      `json:"result"`
}

type problemsetResult struct {
	Problems          []model.Problem `json:"problems"`
	ProblemStatistics []struct {
		ContestID   int    `json:"contestId"`
		Index       string `json:"index"`
		SolvedCount int    `json:"solvedCount"`
	} `json:"problemStatistics"`
}

func (c *Client) Problems(ctx context.Context) ([]model.Problem, error) {
	var result problemsetResult
	if err := c.get(ctx, "problemset.problems", nil, &result); err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(result.ProblemStatistics))
	for _, stat := range result.ProblemStatistics {
		counts[fmt.Sprintf("%d/%s", stat.ContestID, stat.Index)] = stat.SolvedCount
	}
	for i := range result.Problems {
		key := fmt.Sprintf("%d/%s", result.Problems[i].ContestID, result.Problems[i].Index)
		result.Problems[i].SolvedCount = counts[key]
	}
	return result.Problems, nil
}

func (c *Client) Ping(ctx context.Context) error {
	var result json.RawMessage
	return c.get(ctx, "system.status", nil, &result)
}

func (c *Client) UserStatus(ctx context.Context, handle string, from, count int) ([]model.Submission, error) {
	params := url.Values{"handle": {handle}}
	if from > 0 {
		params.Set("from", strconv.Itoa(from))
	}
	if count > 0 {
		params.Set("count", strconv.Itoa(count))
	}
	var result []model.Submission
	if err := c.get(ctx, "user.status", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, method string, params url.Values, out any) error {
	c.mu.Lock()
	wait := 2*time.Second - time.Since(c.last)
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			c.mu.Unlock()
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.last = time.Now()
	c.mu.Unlock()
	u := c.BaseURL + "/" + method
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "cfx/0.1 (+https://github.com/e0-7/cfx)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("Codeforces API %s: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Codeforces API %s: HTTP %s", method, resp.Status)
	}
	var envelope apiResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode Codeforces API %s: %w", method, err)
	}
	if envelope.Status != "OK" {
		return fmt.Errorf("Codeforces API %s: %s", method, envelope.Comment)
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("decode Codeforces API %s result: %w", method, err)
	}
	return nil
}
