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

func TestVoteAnswer(t *testing.T) {
	t.Run("successfully adds a vote", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("Vote Test Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Add vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var msgResp MessageResponse
		err = json.NewDecoder(resp.Body).Decode(&msgResp)
		require.NoError(t, err)
		assert.Equal(t, "vote recorded", msgResp.Message)

		// Verify vote count
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, 1, retrieved.Questions[0].Answers[0].Votes)
	})

	t.Run("successfully removes a vote", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("Remove Vote Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Add vote first
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Remove vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteRemove)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify vote count
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, 0, retrieved.Questions[0].Answers[0].Votes)
	})

	t.Run("returns conflict when adding duplicate vote", func(t *testing.T) {
		client := NewTestClient(t)

		_, err := client.SetupRetrospective("Duplicate Vote Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Add vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Try to add duplicate vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Equal(t, "vote already exists", errResp.Error)
	})

	t.Run("returns not found when removing non-existent vote", func(t *testing.T) {
		client := NewTestClient(t)

		_, err := client.SetupRetrospective("No Vote Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Try to remove vote that doesn't exist
		resp, err = client.VoteAnswer(answer.ID, types.VoteRemove)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Equal(t, "vote not found", errResp.Error)
	})

	t.Run("different sessions can vote on same answer", func(t *testing.T) {
		// Create retrospective with first client
		client1 := NewTestClient(t)
		retro, err := client1.SetupRetrospective("Multi Session Vote", "Description")
		require.NoError(t, err)

		question, resp, err := client1.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client1.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// First client votes
		resp, err = client1.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Second client with different session
		client2 := NewTestClient(t)
		_, err = client2.SetupRetrospective("Another Retro", "Desc")
		require.NoError(t, err)

		// Get the same retrospective to set the cookie
		_, resp, err = client2.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		// Second client votes on the same answer
		resp, err = client2.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify total votes
		retrieved, resp, err := client1.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, 2, retrieved.Questions[0].Answers[0].Votes)
	})

	t.Run("fails with invalid action", func(t *testing.T) {
		client := NewTestClient(t)

		_, err := client.SetupRetrospective("Invalid Action Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Try invalid action
		resp, err = client.VoteAnswer(answer.ID, types.VoteAction("INVALID_ACTION"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errResp, err := ParseErrorResponse(resp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "invalid vote action")
	})

	t.Run("fails without retrospective cookie", func(t *testing.T) {
		freshClient := NewTestClient(t)

		resp, err := freshClient.VoteAnswer(uuid.New(), types.VoteAdd)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestVoteFlow(t *testing.T) {
	t.Run("vote add and remove cycle", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("Vote Cycle Retro", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Test Answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Initial state - no votes
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, 0, retrieved.Questions[0].Answers[0].Votes)

		// Add vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Verify 1 vote
		retrieved, resp, err = client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, 1, retrieved.Questions[0].Answers[0].Votes)

		// Remove vote
		resp, err = client.VoteAnswer(answer.ID, types.VoteRemove)
		require.NoError(t, err)
		resp.Body.Close()

		// Verify 0 votes
		retrieved, resp, err = client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, 0, retrieved.Questions[0].Answers[0].Votes)

		// Add vote again
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Verify 1 vote again
		retrieved, resp, err = client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, 1, retrieved.Questions[0].Answers[0].Votes)
	})

	t.Run("multiple answers can be voted by same session", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("Multi Answer Vote", "Description")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test Question?")
		require.NoError(t, err)
		resp.Body.Close()

		answer1, resp, err := client.CreateAnswer(question.ID, "Answer 1")
		require.NoError(t, err)
		resp.Body.Close()

		answer2, resp, err := client.CreateAnswer(question.ID, "Answer 2")
		require.NoError(t, err)
		resp.Body.Close()

		answer3, resp, err := client.CreateAnswer(question.ID, "Answer 3")
		require.NoError(t, err)
		resp.Body.Close()

		// Vote on all three answers
		resp, err = client.VoteAnswer(answer1.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = client.VoteAnswer(answer2.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = client.VoteAnswer(answer3.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify all votes
		retrieved, resp, err := client.GetRetrospective(retro.ID)
		require.NoError(t, err)
		resp.Body.Close()

		for _, a := range retrieved.Questions[0].Answers {
			assert.Equal(t, 1, a.Votes)
		}
	})
}
