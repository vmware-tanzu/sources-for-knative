/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package horizon

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	testDomain   = "corp"
	testUsername = "user"
	testPassword = "password"
)

func Test_horizonClient_login(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zaptest.NewLogger(t).Sugar()

	t.Run("successful login", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			credentials: AuthLoginRequest{
				Domain:   testDomain,
				Username: testUsername,
				Password: testPassword,
			},
			logger: logger,
		}

		err := h.login(ctx)
		require.NoError(t, err)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			credentials: AuthLoginRequest{
				Domain:   testDomain,
				Username: "unknown",
				Password: "wrong",
			},
			logger: logger,
		}

		err := h.login(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "401")
	})

	t.Run("client with valid refresh token", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		tsRefreshToken := ts.getTokens().RefreshToken

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			tokens: AuthTokens{
				RefreshToken: tsRefreshToken,
			},
			logger: logger,
		}

		err := h.login(ctx)
		require.NoError(t, err)
		require.Equal(t, tsRefreshToken, h.tokens.RefreshToken)
	})

	t.Run("client with invalid refresh token triggers re-auth", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		tsRefreshToken := ts.getTokens().RefreshToken

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			credentials: AuthLoginRequest{
				Domain:   testDomain,
				Username: testUsername,
				Password: testPassword,
			},
			tokens: AuthTokens{
				RefreshToken: "invalid",
			},
			logger: logger,
		}

		err := h.login(ctx)
		require.NoError(t, err)
		require.Equal(t, tsRefreshToken, h.tokens.RefreshToken)
	})

	t.Run("client with expired refresh token triggers re-auth", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		currentTokens := ts.getTokens()
		ts.rotateTokens()
		newTokens := ts.getTokens()

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			credentials: AuthLoginRequest{
				Domain:   testDomain,
				Username: testUsername,
				Password: testPassword,
			},
			tokens: AuthTokens{
				RefreshToken: currentTokens.RefreshToken,
			},
			logger: logger,
		}

		err := h.login(ctx)
		require.NoError(t, err)
		require.Equal(t, newTokens.RefreshToken, h.tokens.RefreshToken)
	})
}

func Test_horizonClient_Logout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zaptest.NewLogger(t).Sugar()

	t.Run("successful logout", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		tsRefreshToken := ts.getTokens().RefreshToken

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			tokens: AuthTokens{
				RefreshToken: tsRefreshToken,
			},
			logger: logger,
		}

		err := h.Logout(ctx)
		require.NoError(t, err)
	})

	t.Run("logout throws error with invalid refresh token", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			tokens: AuthTokens{
				RefreshToken: "invalid",
			},
			logger: logger,
		}

		err := h.Logout(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expired")
	})

	t.Run("logout with expired context", func(t *testing.T) {
		ts := newTestServer(ctx)
		defer ts.httpSrv.Close()

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		h := &horizonClient{
			client: newRESTClient(ctx, ts.httpSrv.URL, false),
			logger: logger,
		}

		err := h.Logout(canceledCtx)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

type horizonAPIMock struct {
	httpSrv *httptest.Server

	sync.RWMutex
	tokens AuthTokens
}

func newTestServer(_ context.Context) *horizonAPIMock {
	mux := http.NewServeMux()
	ts := horizonAPIMock{
		httpSrv: httptest.NewServer(mux),
		tokens: AuthTokens{
			AccessToken:  randomToken(10),
			RefreshToken: randomToken(20),
		},
	}

	mux.HandleFunc(loginPath, ts.loginHandler)
	mux.HandleFunc(logoutPath, ts.logoutHandler)
	mux.HandleFunc(refreshPath, ts.refreshHandler)

	return &ts
}

func (h *horizonAPIMock) rotateTokens() {
	h.Lock()
	h.tokens.AccessToken = randomToken(10)
	h.tokens.RefreshToken = randomToken(20)
	h.Unlock()
}

func (h *horizonAPIMock) getTokens() AuthTokens {
	h.RLock()
	defer h.RUnlock()
	return h.tokens
}

func (h *horizonAPIMock) loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds AuthLoginRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&creds); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if creds.Domain != testDomain ||
		creds.Username != testUsername ||
		creds.Password != testPassword {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	enc := json.NewEncoder(w)
	w.Header().Set("content-type", "application/json")

	h.RLock()
	defer h.RUnlock()
	if err := enc.Encode(h.tokens); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *horizonAPIMock) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var refresh RefreshTokenRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&refresh); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	h.Lock()
	defer h.Unlock()

	if h.tokens.RefreshToken != refresh.RefreshToken {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	w.Header().Set("content-type", "application/json")

	accessToken := AccessToken{
		AccessToken: h.tokens.AccessToken,
	}

	if err := enc.Encode(accessToken); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *horizonAPIMock) logoutHandler(w http.ResponseWriter, r *http.Request) {
	var refresh RefreshTokenRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&refresh); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	h.Lock()
	defer h.Unlock()

	if refresh.RefreshToken == h.tokens.RefreshToken {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}

func randomToken(n int) string {
	rand.Seed(time.Now().Unix())

	letter := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
