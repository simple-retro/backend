package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"

	"api/types"

	"github.com/google/uuid"
)

var baseURL string

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Get base URL from environment or use default
	baseURL = os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}

	os.Exit(m.Run())
}

// TestClient wraps http.Client with helper methods for testing
type TestClient struct {
	*http.Client
	t               *testing.T
	retrospectiveID string // Manually tracked because server sets Secure cookie over HTTP
	sessionCookie   *http.Cookie
}

// NewTestClient creates a new test client with cookie jar
func NewTestClient(t *testing.T) *TestClient {
	jar, _ := cookiejar.New(nil)
	return &TestClient{
		Client: &http.Client{
			Jar: jar,
		},
		t: t,
	}
}

// NewTestClientWithoutCookies creates a client without cookie jar (for testing auth)
func NewTestClientWithoutCookies(t *testing.T) *TestClient {
	return &TestClient{
		Client: &http.Client{},
		t:      t,
	}
}

// extractCookies extracts and stores relevant cookies from response
func (c *TestClient) extractCookies(resp *http.Response) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "retrospective_id" {
			c.retrospectiveID = cookie.Value
		}
		if cookie.Name == "simple-retro-session" {
			c.sessionCookie = cookie
		}
	}
}

// DoRequest performs an HTTP request
func (c *TestClient) DoRequest(method, path string, body interface{}, cookies map[string]string) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Add retrospective_id cookie manually (server sets it as Secure which doesn't work over HTTP)
	if c.retrospectiveID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "retrospective_id",
			Value: c.retrospectiveID,
		})
	}

	for name, value := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	// Extract cookies from response
	c.extractCookies(resp)

	return resp, nil
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse represents a success message response from the API
type MessageResponse struct {
	Message string `json:"message"`
}

// CreateRetrospective creates a new retrospective and returns it
func (c *TestClient) CreateRetrospective(name, description string) (*types.Retrospective, *http.Response, error) {
	reqBody := types.RetrospectiveCreateRequest{
		Name:        name,
		Description: description,
	}

	resp, err := c.DoRequest(http.MethodPost, "/api/retrospective", reqBody, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var retro types.Retrospective
	if err := json.NewDecoder(resp.Body).Decode(&retro); err != nil {
		return nil, resp, err
	}

	return &retro, resp, nil
}

// GetRetrospective retrieves a retrospective (also sets retrospective_id cookie)
func (c *TestClient) GetRetrospective(id uuid.UUID) (*types.Retrospective, *http.Response, error) {
	resp, err := c.DoRequest(http.MethodGet, "/api/retrospective/"+id.String(), nil, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var retro types.Retrospective
	if err := json.NewDecoder(resp.Body).Decode(&retro); err != nil {
		return nil, resp, err
	}

	return &retro, resp, nil
}

// UpdateRetrospective updates a retrospective
func (c *TestClient) UpdateRetrospective(id uuid.UUID, name, description string) (*types.Retrospective, *http.Response, error) {
	reqBody := types.RetrospectiveCreateRequest{
		Name:        name,
		Description: description,
	}

	resp, err := c.DoRequest(http.MethodPatch, "/api/retrospective/"+id.String(), reqBody, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var retro types.Retrospective
	if err := json.NewDecoder(resp.Body).Decode(&retro); err != nil {
		return nil, resp, err
	}

	return &retro, resp, nil
}

// DeleteRetrospective deletes a retrospective
func (c *TestClient) DeleteRetrospective(id uuid.UUID) (*types.Retrospective, *http.Response, error) {
	resp, err := c.DoRequest(http.MethodDelete, "/api/retrospective/"+id.String(), nil, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var retro types.Retrospective
	if err := json.NewDecoder(resp.Body).Decode(&retro); err != nil {
		return nil, resp, err
	}

	return &retro, resp, nil
}

// SetupRetrospective creates a retrospective and gets it to set auth cookies
func (c *TestClient) SetupRetrospective(name, description string) (*types.Retrospective, error) {
	retro, resp, err := c.CreateRetrospective(name, description)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		resp.Body.Close()
	}

	if retro == nil {
		return nil, fmt.Errorf("failed to create retrospective")
	}

	// Get the retrospective to set the auth cookie
	retro, resp, err = c.GetRetrospective(retro.ID)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		resp.Body.Close()
	}

	return retro, nil
}

// CreateQuestion creates a question for the current retrospective
func (c *TestClient) CreateQuestion(text string) (*types.Question, *http.Response, error) {
	reqBody := types.QuestionCreateRequest{
		Text: text,
	}

	resp, err := c.DoRequest(http.MethodPost, "/api/question", reqBody, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var question types.Question
	if err := json.NewDecoder(resp.Body).Decode(&question); err != nil {
		return nil, resp, err
	}

	return &question, resp, nil
}

// UpdateQuestion updates a question
func (c *TestClient) UpdateQuestion(id uuid.UUID, text string) (*types.Question, *http.Response, error) {
	reqBody := types.QuestionCreateRequest{
		Text: text,
	}

	resp, err := c.DoRequest(http.MethodPatch, "/api/question/"+id.String(), reqBody, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var question types.Question
	if err := json.NewDecoder(resp.Body).Decode(&question); err != nil {
		return nil, resp, err
	}

	return &question, resp, nil
}

// DeleteQuestion deletes a question
func (c *TestClient) DeleteQuestion(id uuid.UUID) (*types.Question, *http.Response, error) {
	resp, err := c.DoRequest(http.MethodDelete, "/api/question/"+id.String(), nil, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var question types.Question
	if err := json.NewDecoder(resp.Body).Decode(&question); err != nil {
		return nil, resp, err
	}

	return &question, resp, nil
}

// CreateAnswer creates an answer for a question
func (c *TestClient) CreateAnswer(questionID uuid.UUID, text string) (*types.Answer, *http.Response, error) {
	reqBody := types.AnswerCreateRequest{
		QuestionID: questionID,
		Text:       text,
	}

	resp, err := c.DoRequest(http.MethodPost, "/api/answer", reqBody, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var answer types.Answer
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return nil, resp, err
	}

	return &answer, resp, nil
}

// UpdateAnswer updates an answer
func (c *TestClient) UpdateAnswer(id, questionID uuid.UUID, text string) (*types.Answer, *http.Response, error) {
	reqBody := types.AnswerCreateRequest{
		QuestionID: questionID,
		Text:       text,
	}

	resp, err := c.DoRequest(http.MethodPatch, "/api/answer/"+id.String(), reqBody, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var answer types.Answer
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return nil, resp, err
	}

	return &answer, resp, nil
}

// DeleteAnswer deletes an answer
func (c *TestClient) DeleteAnswer(id uuid.UUID) (*types.Answer, *http.Response, error) {
	resp, err := c.DoRequest(http.MethodDelete, "/api/answer/"+id.String(), nil, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, nil
	}

	var answer types.Answer
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return nil, resp, err
	}

	return &answer, resp, nil
}

// VoteAnswer adds or removes a vote on an answer
func (c *TestClient) VoteAnswer(answerID uuid.UUID, action types.VoteAction) (*http.Response, error) {
	reqBody := types.AnswerVoteRequest{
		AnswerID: answerID,
		Action:   action,
	}

	return c.DoRequest(http.MethodPost, "/api/answer/vote", reqBody, map[string]string{})
}

// GetHealth gets the health endpoint
func (c *TestClient) GetHealth() (*http.Response, error) {
	return c.DoRequest(http.MethodGet, "/api/health", nil, map[string]string{})
}

// GetLimits gets the limits endpoint
func (c *TestClient) GetLimits() (*http.Response, error) {
	return c.DoRequest(http.MethodGet, "/api/limits", nil, map[string]string{})
}

// ParseErrorResponse parses an error response
func ParseErrorResponse(resp *http.Response) (*ErrorResponse, error) {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return nil, err
	}
	return &errResp, nil
}

// GenerateString generates a string of specified length
func GenerateString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}
