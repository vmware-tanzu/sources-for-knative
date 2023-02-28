/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package horizon

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-resty/resty/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/logging"
)

const (
	// domainSecretKey is the key (filename) of the projected secret containing the
	// Horizon Active Directory Domain to use
	domainSecretKey = "domain"

	// DefaultSecretMountPath is the default mount path of the Kubernetes Secret
	// containing Horizon credentials
	//nolint:gosec
	DefaultSecretMountPath = "/var/bindings/horizon" // filepath.Join isn't const.

	// HTTP client
	defaultTimeout = time.Second * 5
	defaultRetries = 3

	// Horizon API
	loginPath   = "/rest/login"
	logoutPath  = "/rest/logout"
	refreshPath = "/rest/refresh"
	eventsPath  = "/rest/external/v1/audit-events"
)

var errTokenExpired = errors.New("refresh token expired")

// Client gets events from the configured Horizon API REST server
type Client interface {
	GetEvents(ctx context.Context, since Timestamp) ([]AuditEventSummary, error)
	Logout(ctx context.Context) error
}

type horizonClient struct {
	client      *resty.Client
	credentials AuthLoginRequest
	tokens      AuthTokens
	logger      *zap.SugaredLogger
}

var _ Client = (*horizonClient)(nil)

// readSecretKey reads the key from a Kubernetes secret
func readSecretKey(key string) (string, error) {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		return "", fmt.Errorf("process environment variables: %w", err)
	}

	mountPath := DefaultSecretMountPath
	if env.SecretPath != "" {
		mountPath = env.SecretPath
	}

	data, err := os.ReadFile(filepath.Join(mountPath, key))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func newHorizonClient(ctx context.Context) (*horizonClient, error) {
	user, err := readSecretKey(corev1.BasicAuthUsernameKey)
	if err != nil {
		return nil, fmt.Errorf("read secret key %q: %w", corev1.BasicAuthUsernameKey, err)
	}

	pass, err := readSecretKey(corev1.BasicAuthPasswordKey)
	if err != nil {
		return nil, fmt.Errorf("read secret key %q: %w", corev1.BasicAuthPasswordKey, err)
	}

	domain, err := readSecretKey(domainSecretKey)
	if err != nil {
		return nil, fmt.Errorf("read secret key %q: %w", domainSecretKey, err)
	}

	creds := AuthLoginRequest{
		Domain:   domain,
		Username: user,
		Password: pass,
	}

	emptyCredentials := func() bool {
		if creds.Domain == "" || creds.Username == "" || creds.Password == "" {
			return true
		}
		return false
	}

	if emptyCredentials() {
		return nil, fmt.Errorf("invalid credentials: domain, username and password must be set")
	}

	var env envConfig
	if err = envconfig.Process("", &env); err != nil {
		return nil, fmt.Errorf("process environment variables: %w", err)
	}

	rc := newRESTClient(ctx, env.Address, env.Insecure)
	c := horizonClient{
		client:      rc,
		logger:      logging.FromContext(ctx),
		credentials: creds,
	}

	if env.Insecure {
		c.logger.Warnw("using potentially insecure connection to Horizon API server", "address", env.Address, "insecure", env.Insecure)
	}

	c.logger.Debug("authenticating against Horizon API")
	if err = c.login(ctx); err != nil {
		return nil, fmt.Errorf("horizon API login failure: %w", err)
	}

	return &c, nil
}

func newRESTClient(ctx context.Context, server string, insecure bool) *resty.Client {
	// REST global client defaults
	r := resty.New().SetLogger(logging.FromContext(ctx))
	r.SetBaseURL(server)
	r.SetHeader("content-type", cloudevents.ApplicationJSON)
	r.SetAuthScheme("Bearer")
	r.SetRetryCount(defaultRetries).SetRetryMaxWaitTime(defaultTimeout)
	//nolint:gosec
	r.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: insecure})

	return r
}

// login performs an authentication request to the Horizon API server, sets and
// stores the returned auth and refresh tokens
func (h *horizonClient) login(ctx context.Context) error {
	/* Access tokens would be valid for 30 minutes while the refresh token would be
	valid for 8 hours. Once the access token has expired, the user will get a 401
	response from the APIs and would need to get a new access token from the refresh
	endpoint using the refresh token. If the Refresh token is also expired (after 8
	hours and when user gets a 400), it indicates that user needs to fully
	re-authenticate using login endpoint due to invalid refresh token.
	*/

	// check if we can use an existing refresh token
	if h.tokens.RefreshToken != "" {
		err := h.refresh(ctx)

		// success
		if err == nil {
			return nil
		}

		if !errors.Is(err, errTokenExpired) {
			return fmt.Errorf("refresh token: %w", err)
		}
	}

	// perform full login
	res, err := h.client.R().SetContext(ctx).SetBody(h.credentials).Post(loginPath)
	if err != nil {
		return err
	}

	if !res.IsSuccess() {
		return fmt.Errorf("horizon API login returned non-success status code: %d", res.StatusCode())
	}

	var tokens AuthTokens
	err = json.Unmarshal(res.Body(), &tokens)
	if err != nil {
		return fmt.Errorf("unmarshal JSON authentication token response: %w", err)
	}

	h.tokens = tokens
	h.client.SetAuthToken(h.tokens.AccessToken)
	h.logger.Debug("Horizon API login successful")

	return nil
}

// refresh attempts to refresh an expired auth token. If the refresh token has
// expired, errTokenExpired will be returned.
func (h *horizonClient) refresh(ctx context.Context) error {
	request := RefreshTokenRequest{h.tokens.RefreshToken}
	res, err := h.client.R().SetContext(ctx).SetBody(request).Post(refreshPath)
	if err != nil {
		return err
	}

	if !res.IsSuccess() {
		switch res.StatusCode() {
		case http.StatusBadRequest:
			return errTokenExpired

		default:
			return fmt.Errorf("unexpected HTTP response: %d %s", res.StatusCode(), string(res.Body()))
		}
	}

	var accessToken AccessToken
	err = json.Unmarshal(res.Body(), &accessToken)
	if err != nil {
		return fmt.Errorf("unmarshal JSON access token response: %w", err)
	}

	token := accessToken.AccessToken
	h.tokens.AccessToken = token
	h.client.SetAuthToken(token)
	h.logger.Debug("auth token refresh successful")
	return nil
}

// GetEvents returns a list of AuditEventSummary from the Horizon API
func (h *horizonClient) GetEvents(ctx context.Context, since Timestamp) ([]AuditEventSummary, error) {
	var (
		res     *resty.Response
		retries int
		err     error

		timeRange string
		params    map[string]string
	)

	// handle auth expired cases
	for retries < 2 {
		if since == 0 {
			// return last (up to) 10 initial events if no timestamp is specified
			params = map[string]string{
				"size": "10",
				"page": "1",
			}
		} else {
			timeRange, err = timeRangeFilter(since, 0)
			h.logger.Debugw("using time range filter", "filter", timeRange)
			if err != nil {
				return nil, fmt.Errorf("create time range query filter: %w", err)
			}

			params = map[string]string{
				"filter": timeRange,
			}
		}

		req := h.client.R().SetContext(ctx).SetQueryParams(params)
		res, err = req.Get(eventsPath)
		if err != nil {
			return nil, err
		}

		h.logger.Debugw("Horizon GetEvents response headers", zap.Any("headers", res.Header()))
		h.logger.Debugw("Horizon GetEvents response body", zap.String("body", string(res.Body())))

		if !res.IsSuccess() {
			switch res.StatusCode() {
			// perform re-auth
			case http.StatusUnauthorized:
				if err = h.login(ctx); err != nil {
					h.logger.Error(string(res.Body()))
					return nil, fmt.Errorf("not authenticated: %w: %s", err, string(res.Body()))
				}
				h.logger.Debugw("retrying get events after re-authentication", zap.Int("retried", retries))
				retries++
				continue

			// 	conflict (note: should never happen on GET and incorrectly used in spec for
			// 	DB missing error)
			case http.StatusConflict:
				return nil, errors.New("HTTP conflict error: 401 (DB not initialized?)")

			// 	not defined in spec
			default:
				return nil, fmt.Errorf("unexpected status code: %d %s", res.StatusCode(), string(res.Body()))
			}
		}

		var events []AuditEventSummary
		err = json.Unmarshal(res.Body(), &events)
		if err != nil {
			return nil, fmt.Errorf("unmarshal JSON audit events response: %w", err)
		}

		return events, nil
	}

	return nil, fmt.Errorf("get events status code: %d %s", res.StatusCode(), string(res.Body()))
}

// timeRangeFilter returns the JSON-encoded query string for the given timestamp
// range. Both values are interpreted as inclusive range values. If to is 0 an
// arbitrary time (UTC) in the future is used as the upper range bound.
func timeRangeFilter(from, to Timestamp) (string, error) {
	// avoid small clock sync issues between client and server and use 1d as future
	// timestamp buffer
	timeBuffer := time.Hour * 24

	if to == 0 {
		to = Timestamp(time.Now().Add(timeBuffer).Unix() * 1000) // milliseconds
	}

	f := BetweenFilter{
		Type:      "Between",
		Name:      "time",
		FromValue: from,
		ToValue:   to,
	}

	filter, err := json.Marshal(f)
	if err != nil {
		return "", fmt.Errorf("JSON marshal filter: %w", err)
	}

	return string(filter), nil
}

// Logout performs a logout against the Horizon API
func (h *horizonClient) Logout(ctx context.Context) error {
	request := RefreshTokenRequest{h.tokens.RefreshToken}
	res, err := h.client.R().SetContext(ctx).SetBody(request).Post(logoutPath)
	if err != nil {
		return err
	}

	if !res.IsSuccess() {
		switch res.StatusCode() {
		case http.StatusBadRequest:
			return errors.New("auth token already expired")

		default:
			return fmt.Errorf("unexpected status code: code: %d error: %s", res.StatusCode(), string(res.Body()))
		}
	}

	return nil
}
