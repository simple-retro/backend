package integration_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"api/types"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WebSocketMessage represents a message from the WebSocket
type WebSocketMessage struct {
	Action string          `json:"action"`
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value"`
}

// connectWebSocket establishes a WebSocket connection for a retrospective
func connectWebSocket(t *testing.T, retroID uuid.UUID) *websocket.Conn {
	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = wsURL + "/api/hello/" + retroID.String()

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Logf("WebSocket dial failed with status: %d", resp.StatusCode)
		}
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	return conn
}

// readWebSocketMessage reads a message from WebSocket with timeout
func readWebSocketMessage(conn *websocket.Conn, timeout time.Duration) (*WebSocketMessage, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))

	_, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var msg WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func TestWebSocketConnection(t *testing.T) {
	t.Run("successfully connects to WebSocket", func(t *testing.T) {
		client := NewTestClient(t)

		// Create retrospective
		retro, resp, err := client.CreateRetrospective("WS Test", "Desc")
		require.NoError(t, err)
		resp.Body.Close()

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Connection should be established
		assert.NotNil(t, conn)
	})

	t.Run("fails to connect with invalid retrospective ID", func(t *testing.T) {
		wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
		wsURL = wsURL + "/api/hello/invalid-uuid"

		dialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
		}

		_, resp, err := dialer.Dial(wsURL, nil)
		// Should fail - either connection error or bad response
		if err == nil {
			t.Log("Connection succeeded unexpectedly")
		}
		if resp != nil {
			assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode)
		}
	})

	t.Run("ping pong works", func(t *testing.T) {
		client := NewTestClient(t)

		retro, resp, err := client.CreateRetrospective("Ping Pong Test", "Desc")
		require.NoError(t, err)
		resp.Body.Close()

		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Send ping message (server checks message.Type == "ping")
		pingMsg := map[string]string{"type": "ping"}
		err = conn.WriteJSON(pingMsg)
		require.NoError(t, err)

		// Read pong response (server only sets Type, not Action)
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "pong", msg.Type)
	})
}

func TestWebSocketBroadcasts(t *testing.T) {
	t.Run("receives question create broadcast", func(t *testing.T) {
		client := NewTestClient(t)

		// Setup retrospective
		retro, err := client.SetupRetrospective("WS Question Create", "Desc")
		require.NoError(t, err)

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Create question via HTTP
		question, resp, err := client.CreateQuestion("Broadcast test question?")
		require.NoError(t, err)
		resp.Body.Close()

		// Read broadcast message
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "create", msg.Action)
		assert.Equal(t, "question", msg.Type)

		// Parse the value
		var receivedQuestion types.Question
		err = json.Unmarshal(msg.Value, &receivedQuestion)
		require.NoError(t, err)
		assert.Equal(t, question.ID, receivedQuestion.ID)
		assert.Equal(t, question.Text, receivedQuestion.Text)
	})

	t.Run("receives question update broadcast", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("WS Question Update", "Desc")
		require.NoError(t, err)

		// Create question first
		question, resp, err := client.CreateQuestion("Original question")
		require.NoError(t, err)
		resp.Body.Close()

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Update question via HTTP
		_, resp, err = client.UpdateQuestion(question.ID, "Updated question")
		require.NoError(t, err)
		resp.Body.Close()

		// Read broadcast message
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "update", msg.Action)
		assert.Equal(t, "question", msg.Type)
	})

	t.Run("receives question delete broadcast", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("WS Question Delete", "Desc")
		require.NoError(t, err)

		// Create question first
		question, resp, err := client.CreateQuestion("To be deleted")
		require.NoError(t, err)
		resp.Body.Close()

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Delete question via HTTP
		_, resp, err = client.DeleteQuestion(question.ID)
		require.NoError(t, err)
		resp.Body.Close()

		// Read broadcast message
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "delete", msg.Action)
		assert.Equal(t, "question", msg.Type)
	})

	t.Run("receives answer create broadcast", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("WS Answer Create", "Desc")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test?")
		require.NoError(t, err)
		resp.Body.Close()

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Create answer via HTTP
		answer, resp, err := client.CreateAnswer(question.ID, "Broadcast answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Read broadcast message
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "create", msg.Action)
		assert.Equal(t, "answer", msg.Type)

		var receivedAnswer types.Answer
		err = json.Unmarshal(msg.Value, &receivedAnswer)
		require.NoError(t, err)
		assert.Equal(t, answer.ID, receivedAnswer.ID)
	})

	t.Run("receives vote add broadcast", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("WS Vote Add", "Desc")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Voteable answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Add vote via HTTP
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Read broadcast message
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "add_vote", msg.Action)
		assert.Equal(t, "answer", msg.Type)

		var receivedAnswer types.Answer
		err = json.Unmarshal(msg.Value, &receivedAnswer)
		require.NoError(t, err)
		assert.Equal(t, 1, receivedAnswer.Votes)
	})

	t.Run("receives vote remove broadcast", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("WS Vote Remove", "Desc")
		require.NoError(t, err)

		question, resp, err := client.CreateQuestion("Test?")
		require.NoError(t, err)
		resp.Body.Close()

		answer, resp, err := client.CreateAnswer(question.ID, "Voteable answer")
		require.NoError(t, err)
		resp.Body.Close()

		// Add vote first
		resp, err = client.VoteAnswer(answer.ID, types.VoteAdd)
		require.NoError(t, err)
		resp.Body.Close()

		// Connect to WebSocket
		conn := connectWebSocket(t, retro.ID)
		defer conn.Close()

		// Remove vote via HTTP
		resp, err = client.VoteAnswer(answer.ID, types.VoteRemove)
		require.NoError(t, err)
		resp.Body.Close()

		// Read broadcast message
		msg, err := readWebSocketMessage(conn, 5*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "remove_vote", msg.Action)
		assert.Equal(t, "answer", msg.Type)

		var receivedAnswer types.Answer
		err = json.Unmarshal(msg.Value, &receivedAnswer)
		require.NoError(t, err)
		assert.Equal(t, 0, receivedAnswer.Votes)
	})
}

func TestWebSocketMultipleClients(t *testing.T) {
	t.Run("multiple clients receive broadcasts", func(t *testing.T) {
		client := NewTestClient(t)

		retro, err := client.SetupRetrospective("WS Multi Client", "Desc")
		require.NoError(t, err)

		// Connect multiple WebSocket clients
		conn1 := connectWebSocket(t, retro.ID)
		defer conn1.Close()

		conn2 := connectWebSocket(t, retro.ID)
		defer conn2.Close()

		conn3 := connectWebSocket(t, retro.ID)
		defer conn3.Close()

		// Create question via HTTP
		_, resp, err := client.CreateQuestion("Multi-client test?")
		require.NoError(t, err)
		resp.Body.Close()

		// All clients should receive the message
		msg1, err := readWebSocketMessage(conn1, 5*time.Second)
		require.NoError(t, err)
		assert.Equal(t, "create", msg1.Action)

		msg2, err := readWebSocketMessage(conn2, 5*time.Second)
		require.NoError(t, err)
		assert.Equal(t, "create", msg2.Action)

		msg3, err := readWebSocketMessage(conn3, 5*time.Second)
		require.NoError(t, err)
		assert.Equal(t, "create", msg3.Action)
	})

	t.Run("broadcasts are isolated by retrospective", func(t *testing.T) {
		client := NewTestClient(t)

		// Create two retrospectives
		retro1, err := client.SetupRetrospective("WS Isolated 1", "Desc")
		require.NoError(t, err)

		client2 := NewTestClient(t)
		retro2, err := client2.SetupRetrospective("WS Isolated 2", "Desc")
		require.NoError(t, err)

		// Connect to different retrospectives
		conn1 := connectWebSocket(t, retro1.ID)
		defer conn1.Close()

		conn2 := connectWebSocket(t, retro2.ID)
		defer conn2.Close()

		// Create question in retro1
		_, resp, err := client.CreateQuestion("Retro 1 question")
		require.NoError(t, err)
		resp.Body.Close()

		// conn1 should receive the message
		msg1, err := readWebSocketMessage(conn1, 5*time.Second)
		require.NoError(t, err)
		assert.Equal(t, "create", msg1.Action)

		// conn2 should NOT receive the message (different retro)
		conn2.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, _, err = conn2.ReadMessage()
		assert.Error(t, err, "conn2 should timeout as it shouldn't receive messages from retro1")
	})
}
