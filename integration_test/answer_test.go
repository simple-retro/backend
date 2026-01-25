package integration_test

import (
	"net/http"
	"testing"

	"api/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAnswer(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully creates an answer", func(t *testing.T) {
		_, err := client.SetupRetrospective("Answer Test Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEqual(t, uuid.Nil, answer.ID)
		assert.Equal(t, question.ID, answer.QuestionID)
		assert.Equal(t, "Test Answer", answer.Text)
		assert.Equal(t, 1, answer.Position)
		assert.Equal(t, 0, answer.Votes)
	})

	t.Run("creates multiple answers with incrementing positions", func(t *testing.T) {
		_, err := client.SetupRetrospective("Multi Answer Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		a1, resp, err := client.CreateAnswer(question.ID, "Answer 1")
		require.NoError(t, err)
		resp.Body.Close()

		a2, resp, err := client.CreateAnswer(question.ID, "Answer 2")
		require.NoError(t, err)
		resp.Body.Close()

		a3, resp, err := client.CreateAnswer(question.ID, "Answer 3")
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, 1, a1.Position)
		assert.Equal(t, 2, a2.Position)
		assert.Equal(t, 3, a3.Position)
	})

	t.Run("fails with text exceeding limit", func(t *testing.T) {
		_, err := client.SetupRetrospective("Long Answer Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		longText := GenerateString(601) // ANSWER_LIMIT is 600
		_, resp, err = client.CreateAnswer(question.ID, longText)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "answer text too big")
	})
}

func TestUpdateAnswer(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully updates an answer", func(t *testing.T) {
		_, err := client.SetupRetrospective("Update Answer Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Original text")
		require.NoError(t, err)
		resp.Body.Close()

		updated, resp, err := client.UpdateAnswer(answer.ID, question.ID, "Updated text")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, answer.ID, updated.ID)
		assert.Equal(t, "Updated text", updated.Text)
	})

	t.Run("preserves text when updating with empty text", func(t *testing.T) {
		// Note: The API preserves the original text when updating with empty text
		_, err := client.SetupRetrospective("Empty Update Answer", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Original text")
		require.NoError(t, err)
		resp.Body.Close()

		updated, resp, err := client.UpdateAnswer(answer.ID, question.ID, "")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// API preserves original text when empty string is sent
		assert.Equal(t, "Original text", updated.Text)
	})

	t.Run("fails with text exceeding limit", func(t *testing.T) {
		_, err := client.SetupRetrospective("Long Update Answer", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Original text")
		require.NoError(t, err)
		resp.Body.Close()

		longText := GenerateString(601)
		_, resp, err = client.UpdateAnswer(answer.ID, question.ID, longText)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent answer", func(t *testing.T) {
		_, err := client.SetupRetrospective("Non-existent Update", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		nonExistentID := uuid.New()
		_, resp, err = client.UpdateAnswer(nonExistentID, question.ID, "New text")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		_, err := client.SetupRetrospective("Invalid UUID Answer", "Description")
		require.NoError(t, err)

		resp, err := client.DoRequest(http.MethodPatch, "/api/answer/invalid-uuid", types.AnswerCreateRequest{
			QuestionID: uuid.New(),
			Text:       "text",
		}, map[string]string{})
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestDeleteAnswer(t *testing.T) {
	client := NewTestClient(t)

	t.Run("successfully deletes an answer", func(t *testing.T) {
		retro, err := client.SetupRetrospective("Delete Answer Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "To be deleted")
		require.NoError(t, err)
		resp.Body.Close()

		deleted, resp, err := client.DeleteAnswer(answer.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, answer.ID, deleted.ID)

		// Verify deletion
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		for _, q := range retrieved.Questions {
			for _, a := range q.Answers {
				assert.NotEqual(t, answer.ID, a.ID)
			}
		}
	})

	t.Run("deletes answer with votes (cascade)", func(t *testing.T) {
		_, err := client.SetupRetrospective("Cascade Delete Answer", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Answer with votes")
		require.NoError(t, err)
		resp.Body.Close()

		// Add a vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Delete the answer
		_, resp, err = client.DeleteAnswer(answer.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent answer", func(t *testing.T) {
		_, err := client.SetupRetrospective("Non-existent Delete", "Description")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, resp, err := client.DeleteAnswer(nonExistentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
