package integration_test

import (
	"net/http"
	"testing"

	"api/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	t.Run("public endpoints work without auth", func(t *testing.T) {
		client := NewTestClient(t)

		// Health endpoint
		resp, err := client.GetHealth()
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Limits endpoint
		resp, err = client.GetLimits()
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Create retrospective
		_, resp, err = client.CreateRetrospective("Auth Test", "Desc")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("protected endpoints require retrospective_id cookie", func(t *testing.T) {
		client := NewTestClient(t)

		// Question endpoints
		_, resp, err := client.CreateQuestion("Test?")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		_, resp, err = client.UpdateQuestion(uuid.New(), "Updated?")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		_, resp, err = client.DeleteQuestion(uuid.New())
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// Answer endpoints
		_, resp, err = client.CreateAnswer(uuid.New(), "Answer")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		_, resp, err = client.UpdateAnswer(uuid.New(), uuid.New(), "Updated")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		_, resp, err = client.DeleteAnswer(uuid.New())
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// Vote endpoint
		resp, err = client.VoteAnswer(uuid.New(), types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("retrospective_id cookie is set on GET retrospective", func(t *testing.T) {
		client := NewTestClient(t)

		// Create retrospective
		created, resp, err := client.CreateRetrospective("Cookie Test", "Desc")
		require.NoError(t, err)
		resp.Body.Close()

		// Get retrospective - should set cookie
		_, resp, err = client.GetRetrospective(created.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check cookie
		var retroCookie *http.Cookie
		for _, c := range resp.Cookies() {
			if c.Name == "retrospective_id" {
				retroCookie = c
				break
			}
		}
		assert.NotNil(t, retroCookie)
		assert.Equal(t, created.ID.String(), retroCookie.Value)
	})

	t.Run("invalid retrospective_id cookie returns unauthorized", func(t *testing.T) {
		client := NewTestClientWithoutCookies(t)

		// Try to create question with invalid cookie
		resp, err := client.DoRequest(
			http.MethodPost,
			"/api/question",
			types.QuestionCreateRequest{Text: "Test?"},
			map[string]string{"retrospective_id": "invalid-uuid"},
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Equal(t, "not in any retrospective", errResp.Error)
	})

	t.Run("valid retrospective_id cookie allows access", func(t *testing.T) {
		client := NewTestClient(t)

		// Setup retrospective
		retro, err := client.SetupRetrospective("Valid Cookie Test", "Desc")
		require.NoError(t, err)

		// Create question should work now
		question, resp, err := client.CreateQuestion("Test question?")
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEqual(t, uuid.Nil, question.ID)

		// Verify question was created under the correct retrospective
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Len(t, retrieved.Questions, 1)
		assert.Equal(t, question.ID, retrieved.Questions[0].ID)
	})
}

func TestAuthenticationErrors(t *testing.T) {
	t.Run("returns proper error message for missing cookie", func(t *testing.T) {
		client := NewTestClient(t)

		resp, err := client.DoRequest(http.MethodPost, "/api/question", types.QuestionCreateRequest{Text: "Test?"}, map[string]string{})
		require.NoError(t, err)
		defer resp.Body.Close()

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Equal(t, "not in any retrospective", errResp.Error)
	})

	t.Run("returns proper error for retrospective not found", func(t *testing.T) {
		client := NewTestClient(t)

		nonExistentID := uuid.New()
		_, resp, err := client.GetRetrospective(nonExistentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "not found")
	})
}
