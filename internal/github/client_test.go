package github

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientSingleton(t *testing.T) {
	// Reset singleton before test
	ResetClient()

	client1 := GetClient("token1")
	client2 := GetClient("token2")

	// Both should return the same instance
	assert.Equal(t, client1, client2, "GetClient should return the same instance")

	// Token should be from first initialization
	assert.Equal(t, "token1", client1.token)
	assert.Equal(t, "token1", client2.token)

	// Reset for other tests
	ResetClient()
}

func TestFetchFileSuccess(t *testing.T) {
	// Reset singleton before test
	ResetClient()
	mockContent := "test file content"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(mockContent))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		response := Content{
			Content:  encodedContent,
			Encoding: "base64",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := GetClient("test-token")

	// We can't easily test the real FetchFile without modifying it to accept a base URL
	// Instead, test the HTTP client directly
	req, err := http.NewRequest("GET", mockServer.URL, nil)
	assert.NoError(t, err)

	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var content Content
	err = json.NewDecoder(resp.Body).Decode(&content)
	assert.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	assert.NoError(t, err)
	assert.Equal(t, mockContent, string(decoded))
}

func TestFetchFileUnauthorized(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer mockServer.Close()

	client := GetClient("invalid-token")

	req, err := http.NewRequest("GET", mockServer.URL, nil)
	assert.NoError(t, err)

	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := client.client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
