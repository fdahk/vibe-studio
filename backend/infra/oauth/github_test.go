package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			_, _ = w.Write([]byte(`{"access_token":"gho_abc"}`))
		case "/user":
			if r.Header.Get("Authorization") != "Bearer gho_abc" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"id":12345,"login":"octocat","name":"The Octocat","email":"octo@x.com","avatar_url":"http://a"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	g := NewGitHub("cid", "secret", "redir")
	g.tokenURL = srv.URL + "/login/oauth/access_token"
	g.apiBase = srv.URL

	p, err := g.Exchange(context.Background(), "code123")
	require.NoError(t, err)
	assert.Equal(t, "12345", p.ID)
	assert.Equal(t, "octocat", p.Login)
	assert.Equal(t, "The Octocat", p.Name)
	assert.Equal(t, "octo@x.com", p.Email)
}

func TestGitHubExchangeFailsWithoutToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"bad_verification_code"}`))
	}))
	defer srv.Close()

	g := NewGitHub("cid", "secret", "redir")
	g.tokenURL = srv.URL
	g.apiBase = srv.URL
	_, err := g.Exchange(context.Background(), "bad")
	assert.Error(t, err)
}

func TestGitHubConfigured(t *testing.T) {
	assert.True(t, NewGitHub("a", "b", "r").Configured())
	assert.False(t, NewGitHub("", "", "r").Configured())
}

func TestGitHubAuthorizeURL(t *testing.T) {
	u := NewGitHub("cid", "secret", "https://app/cb").AuthorizeURL("st8")
	assert.Contains(t, u, "client_id=cid")
	assert.Contains(t, u, "state=st8")
	assert.Contains(t, u, "redirect_uri=https%3A%2F%2Fapp%2Fcb")
}
