package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"api/types"

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

func TestExportRetrospective(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully exports retrospective as JSON", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Export JSON Test", "Test Description")
		require.NoError(t, err)

		// Create a question and answer for richer export
		question, resp, err := client.CreateQuestion("What went well?")
		require.NoError(t, err)
		resp.Body.Close()

		_, resp, err = client.CreateAnswer(question.ID, "Great teamwork")
		require.NoError(t, err)
		resp.Body.Close()

		// Export as JSON
		data, resp, err := client.ExportRetrospective(retro.ID, types.ExportTypeJSON)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), ".json")

		// Verify JSON content
		var exportedRetro types.Retrospective
		err = json.Unmarshal(data, &exportedRetro)
		require.NoError(t, err)
		assert.Equal(t, retro.ID, exportedRetro.ID)
		assert.Equal(t, "Export JSON Test", exportedRetro.Name)
		assert.Len(t, exportedRetro.Questions, 1)
		assert.Len(t, exportedRetro.Questions[0].Answers, 1)
	})

	t.Run("successfully exports retrospective as Markdown", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Export Markdown Test", "Markdown Description")
		require.NoError(t, err)

		// Create a question and answer
		question, resp, err := client.CreateQuestion("What could be improved?")
		require.NoError(t, err)
		resp.Body.Close()

		_, resp, err = client.CreateAnswer(question.ID, "Better communication")
		require.NoError(t, err)
		resp.Body.Close()

		// Export as Markdown
		data, resp, err := client.ExportRetrospective(retro.ID, types.ExportTypeMarkdown)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/markdown", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), ".md")

		// Verify Markdown content
		content := string(data)
		assert.Contains(t, content, "# Simple Retro")
		assert.Contains(t, content, "## Export Markdown Test")
		assert.Contains(t, content, "Markdown Description")
		assert.Contains(t, content, "### What could be improved?")
		assert.Contains(t, content, "- Better communication")
	})

	t.Run("successfully exports retrospective as PDF", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Export PDF Test", "PDF Description")
		require.NoError(t, err)

		// Create a question and answer
		question, resp, err := client.CreateQuestion("Action items?")
		require.NoError(t, err)
		resp.Body.Close()

		_, resp, err = client.CreateAnswer(question.ID, "Schedule follow-up meeting")
		require.NoError(t, err)
		resp.Body.Close()

		// Export as PDF
		data, resp, err := client.ExportRetrospective(retro.ID, types.ExportTypePDF)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), ".pdf")

		// Verify PDF content (check PDF magic bytes)
		assert.True(t, len(data) > 4)
		assert.Equal(t, "%PDF", string(data[:4]))
	})

	t.Run("exports empty retrospective", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Empty Export Test", "No questions")
		require.NoError(t, err)

		// Export as Markdown
		data, resp, err := client.ExportRetrospective(retro.ID, types.ExportTypeMarkdown)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		content := string(data)
		assert.Contains(t, content, "# Simple Retro")
		assert.Contains(t, content, "## Empty Export Test")
	})

	t.Run("returns 404 for non-existent retrospective", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, resp, err := client.ExportRetrospective(nonExistentID, types.ExportTypeJSON)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for unknown export type", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Unknown Type Test", "Description")
		require.NoError(t, err)

		// Export with unknown type
		_, resp, err := client.ExportRetrospective(retro.ID, types.ExportType("UNKNOWN"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
