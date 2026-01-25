package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HealthResponse represents the health endpoint response
type HealthResponse struct {
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

// LimitsResponse represents the limits endpoint response
type LimitsResponse struct {
	Retrospective struct {
		Name        int `json:"name"`
		Description int `json:"description"`
	} `json:"retrospective"`
	Question struct {
		Text int `json:"text"`
	} `json:"question"`
	Answer struct {
		Text int `json:"text"`
	} `json:"answer"`
}

func TestHealthEndpoint(t *testing.T) {
	client := NewTestClient(t)

	t.Run("returns health information", func(t *testing.T) {
		resp, err := client.GetHealth()
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.NotEmpty(t, health.Name)
		assert.GreaterOrEqual(t, health.CPU, float64(0))
		assert.GreaterOrEqual(t, health.Memory, float64(0))
	})
}

func TestLimitsEndpoint(t *testing.T) {
	client := NewTestClient(t)

	t.Run("returns API limits", func(t *testing.T) {
		resp, err := client.GetLimits()
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var limits LimitsResponse
		err = json.NewDecoder(resp.Body).Decode(&limits)
		require.NoError(t, err)

		// Verify expected limits
		assert.Equal(t, 100, limits.Retrospective.Name)
		assert.Equal(t, 300, limits.Retrospective.Description)
		assert.Equal(t, 300, limits.Question.Text)
		assert.Equal(t, 600, limits.Answer.Text)
	})
}
