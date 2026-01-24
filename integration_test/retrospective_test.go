package integration_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRetrospective(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully creates a retrospective", func(t *testing.T) {
		retro, resp, err := client.CreateRetrospective("Test Retro", "Test Description")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEqual(t, uuid.Nil, retro.ID)
		assert.Equal(t, "Test Retro", retro.Name)
		assert.Equal(t, "Test Description", retro.Description)
		assert.Empty(t, retro.Questions)
	})

	t.Run("creates retrospective with empty description", func(t *testing.T) {
		retro, resp, err := client.CreateRetrospective("Retro Without Desc", "")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Retro Without Desc", retro.Name)
		assert.Equal(t, "", retro.Description)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		_, resp, err := client.CreateRetrospective("", "Description")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "name cannot be empty")
	})

	t.Run("fails with name exceeding limit", func(t *testing.T) {
		longName := GenerateString(101) // NAME_LIMIT is 100
		_, resp, err := client.CreateRetrospective(longName, "Description")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "name too big")
	})

	t.Run("fails with description exceeding limit", func(t *testing.T) {
		longDesc := GenerateString(301) // DESC_LIMIT is 300
		_, resp, err := client.CreateRetrospective("Valid Name", longDesc)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "description too big")
	})
}

func TestGetRetrospective(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully gets a retrospective", func(t *testing.T) {
		// Create a retrospective first
		created, resp, err := client.CreateRetrospective("Get Test", "Get Test Description")
		require.NoError(t, err)
		resp.Body.Close()

		// Get the retrospective
		retro, resp, err := client.GetRetrospective(created.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, created.ID, retro.ID)
		assert.Equal(t, created.Name, retro.Name)
		assert.Equal(t, created.Description, retro.Description)

		// Check that retrospective_id cookie is set
		cookies := resp.Cookies()
		var retroCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "retrospective_id" {
				retroCookie = c
				break
			}
		}
		assert.NotNil(t, retroCookie)
		assert.Equal(t, created.ID.String(), retroCookie.Value)
	})

	t.Run("returns 404 for non-existent retrospective", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, resp, err := client.GetRetrospective(nonExistentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		resp, err := client.DoRequest(http.MethodGet, "/api/retrospective/invalid-uuid", nil, map[string]string{})
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateRetrospective(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully updates retrospective name", func(t *testing.T) {
		// Setup retrospective with auth cookie
		retro, err := client.SetupRetrospective("Original Name", "Original Desc")
		require.NoError(t, err)

		// Update the retrospective
		updated, resp, err := client.UpdateRetrospective(retro.ID, "Updated Name", "Original Desc")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Updated Name", updated.Name)
	})

	t.Run("successfully updates retrospective description", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Name", "Original Desc")
		require.NoError(t, err)

		updated, resp, err := client.UpdateRetrospective(retro.ID, "Name", "Updated Desc")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Updated Desc", updated.Description)
	})

	t.Run("fails when both name and description are empty", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Name", "Desc")
		require.NoError(t, err)

		_, resp, err := client.UpdateRetrospective(retro.ID, "", "")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "nothing to do")
	})

	t.Run("returns 404 for non-existent retrospective", func(t *testing.T) {
		// First setup a retrospective to get auth cookie
		_, err := client.SetupRetrospective("Setup", "Desc")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, resp, err := client.UpdateRetrospective(nonExistentID, "Name", "Desc")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteRetrospective(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully deletes a retrospective", func(t *testing.T) {
		retro, err := client.SetupRetrospective("To Delete", "Description")
		require.NoError(t, err)

		deleted, resp, err := client.DeleteRetrospective(retro.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, retro.ID, deleted.ID)

		// Verify it's deleted
		_, getResp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		defer getResp.Body.Close()
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})

	t.Run("deletes retrospective with questions and answers (cascade)", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Cascade Delete", "Description")
		require.NoError(t, err)

		// Create question
		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		// Create answer
		_, resp, err = client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Delete retrospective
		_, resp, err = client.DeleteRetrospective(retro.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify retrospective is deleted
		_, getResp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		defer getResp.Body.Close()
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})

	t.Run("returns 404 for non-existent retrospective", func(t *testing.T) {
		_, err := client.SetupRetrospective("Setup", "Desc")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, resp, err := client.DeleteRetrospective(nonExistentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
