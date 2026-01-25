package integration_test

import (
	"net/http"
	"testing"

	"api/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateQuestion(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully creates a question", func(t *testing.T) {
		// Setup retrospective first
		_, err := client.SetupRetrospective("Question Test Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("What went well?")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEqual(t, uuid.Nil, question.ID)
		assert.Equal(t, "What went well?", question.Text)
		assert.Empty(t, question.Answers)
	})

	t.Run("creates multiple questions for same retrospective", func(t *testing.T) {
		_, err := client.SetupRetrospective("Multi Question Retro", "Description")
		require.NoError(t, err)

		q1, resp, err := client.CreateQuestion("Question 1")
		require.NoError(t, err)
		resp.Body.Close()

		q2, resp, err := client.CreateQuestion("Question 2")
		require.NoError(t, err)
		resp.Body.Close()

		q3, resp, err := client.CreateQuestion("Question 3")
		require.NoError(t, err)
		resp.Body.Close()

		assert.NotEqual(t, q1.ID, q2.ID)
		assert.NotEqual(t, q2.ID, q3.ID)
	})

	t.Run("fails with empty text", func(t *testing.T) {
		_, err := client.SetupRetrospective("Empty Question Retro", "Description")
		require.NoError(t, err)

		_, resp, err := client.CreateQuestion("")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "text cannot be empty")
	})

	t.Run("fails with text exceeding limit", func(t *testing.T) {
		_, err := client.SetupRetrospective("Long Question Retro", "Description")
		require.NoError(t, err)

		longText := GenerateString(301) // DESC_LIMIT is 300
		_, resp, err := client.CreateQuestion(longText)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "question too big")
	})
}

func TestUpdateQuestion(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully updates a question", func(t *testing.T) {
		_, err := client.SetupRetrospective("Update Question Retro", "Description")
		require.NoError(t, err)

		// Create question
		question, resp, err := client.CreateQuestion("Original text")
		require.NoError(t, err)
		resp.Body.Close()

		// Update question
		updated, resp, err := client.UpdateQuestion(question.ID, "Updated text")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, question.ID, updated.ID)
		assert.Equal(t, "Updated text", updated.Text)
	})

	t.Run("fails with empty text", func(t *testing.T) {
		_, err := client.SetupRetrospective("Empty Update Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Original text")
		require.NoError(t, err)
		resp.Body.Close()

		_, resp, err = client.UpdateQuestion(question.ID, "")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("fails with text exceeding limit", func(t *testing.T) {
		_, err := client.SetupRetrospective("Long Update Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Original text")
		require.NoError(t, err)
		resp.Body.Close()

		longText := GenerateString(301)
		_, resp, err = client.UpdateQuestion(question.ID, longText)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent question", func(t *testing.T) {
		_, err := client.SetupRetrospective("Non-existent Update Retro", "Description")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, resp, err := client.UpdateQuestion(nonExistentID, "New text")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		_, err := client.SetupRetrospective("Invalid UUID Retro", "Description")
		require.NoError(t, err)

		resp, err := client.DoRequest(http.MethodPatch, "/api/question/invalid-uuid", types.QuestionCreateRequest{Text: "text"}, map[string]string{})
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestDeleteQuestion(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully deletes a question", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Delete Question Retro", "Description")
		require.NoError(t, err)

		// Create question
		question, resp, err := client.CreateQuestion("To be deleted")
		require.NoError(t, err)
		resp.Body.Close()

		// Delete question
		deleted, resp, err := client.DeleteQuestion(question.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, question.ID, deleted.ID)

		// Verify deletion by getting the retrospective
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		// Question should not be in the list
		for _, q := range retrieved.Questions {
			assert.NotEqual(t, question.ID, q.ID)
		}
	})

	t.Run("deletes question with answers (cascade)", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Cascade Delete Question", "Description")
		require.NoError(t, err)

		// Create question
		question, resp, err := client.CreateQuestion("Question with answers")
		require.NoError(t, err)
		resp.Body.Close()

		// Create answers
		_, resp, err = client.CreateAnswer(question.ID, "Answer 1")
		require.NoError(t, err)
		resp.Body.Close()

		_, resp, err = client.CreateAnswer(question.ID, "Answer 2")
		require.NoError(t, err)
		resp.Body.Close()

		// Delete question
		_, resp, err = client.DeleteQuestion(question.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify question and answers are deleted
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		for _, q := range retrieved.Questions {
			assert.NotEqual(t, question.ID, q.ID)
		}
	})

	t.Run("returns 404 for non-existent question", func(t *testing.T) {
		_, err := client.SetupRetrospective("Non-existent Delete Retro", "Description")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, resp, err := client.DeleteQuestion(nonExistentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
