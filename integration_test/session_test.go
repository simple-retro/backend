package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionID(t *testing.T) {
	t.Run("session cookie is created on first request", func(t *testing.T) {
		client := NewTestClient(t)

		// Make any request
		resp, err := client.GetHealth()
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check for session cookie
		var sessionCookie *http.Cookie
		for _, c := range resp.Cookies() {
			if c.Name == "simple-retro-session" {
				sessionCookie = c
				break
			}
		}

		assert.NotNil(t, sessionCookie, "session cookie should be created")
		assert.NotEmpty(t, sessionCookie.Value, "session cookie should have a value")
	})
}
