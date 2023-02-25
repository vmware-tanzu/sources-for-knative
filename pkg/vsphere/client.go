/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/session/keepalive"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"knative.dev/pkg/logging"

	corev1 "k8s.io/api/core/v1"
)

const (
	VolumeName        = "vsphere-binding"
	DefaultMountPath  = "/var/bindings/vsphere" // filepath.Join isn't const.
	keepaliveInterval = 5 * time.Minute         // vCenter APIs keep-alive
)

type EnvConfig struct {
	Insecure   bool   `envconfig:"VC_INSECURE" default:"false"`
	Address    string `envconfig:"VC_URL" required:"true"`
	SecretPath string `envconfig:"VC_SECRET_PATH" default:""`
}

// ReadKey reads the key from the secret.
func ReadKey(key string) (string, error) {
	var env EnvConfig
	if err := envconfig.Process("", &env); err != nil {
		return "", err
	}

	mountPath := DefaultMountPath
	if env.SecretPath != "" {
		mountPath = env.SecretPath
	}

	data, err := os.ReadFile(filepath.Join(mountPath, key))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NewSOAPClient returns a vCenter SOAP API client with active keep-alive. Use
// Logout() to release resources and perform a clean logout from vCenter.
func NewSOAPClient(ctx context.Context) (*govmomi.Client, error) {
	var env EnvConfig
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}

	parsedURL, err := soap.ParseURL(env.Address)
	if err != nil {
		return nil, err
	}

	// Read the username and password from the filesystem.
	username, err := ReadKey(corev1.BasicAuthUsernameKey)
	if err != nil {
		return nil, err
	}
	password, err := ReadKey(corev1.BasicAuthPasswordKey)
	if err != nil {
		return nil, err
	}
	parsedURL.User = url.UserPassword(username, password)

	return soapWithKeepalive(ctx, parsedURL, env.Insecure)
}

func soapWithKeepalive(ctx context.Context, url *url.URL, insecure bool) (*govmomi.Client, error) {
	soapClient := soap.NewClient(url, insecure)
	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, err
	}
	vimClient.RoundTripper = keepalive.NewHandlerSOAP(vimClient.RoundTripper, keepaliveInterval, soapKeepAliveHandler(ctx, vimClient))

	// explicitly create session to activate keep-alive handler via Login
	m := session.NewManager(vimClient)
	err = m.Login(ctx, url.User)
	if err != nil {
		return nil, err
	}

	c := govmomi.Client{
		Client:         vimClient,
		SessionManager: m,
	}

	return &c, nil
}

func soapKeepAliveHandler(ctx context.Context, c *vim25.Client) func() error {
	logger := logging.FromContext(ctx).With("rpc", "keepalive")

	return func() error {
		logger.Info("Executing SOAP keep-alive handler")
		t, err := methods.GetCurrentTime(ctx, c)
		if err != nil {
			return err
		}

		logger.Infof("vCenter current time: %s", t.String())
		return nil
	}
}

// NewRESTClient returns a vCenter REST API client with active keep-alive. Use
// Logout() to release resources and perform a clean logout from vCenter.
func NewRESTClient(ctx context.Context) (*rest.Client, error) {
	var env EnvConfig
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}

	parsedURL, err := soap.ParseURL(env.Address)
	if err != nil {
		return nil, err
	}

	// Read the username and password from the filesystem.
	username, err := ReadKey(corev1.BasicAuthUsernameKey)
	if err != nil {
		return nil, err
	}
	password, err := ReadKey(corev1.BasicAuthPasswordKey)
	if err != nil {
		return nil, err
	}
	parsedURL.User = url.UserPassword(username, password)

	soapclient, err := soapWithKeepalive(ctx, parsedURL, env.Insecure)
	if err != nil {
		return nil, err
	}

	restclient := rest.NewClient(soapclient.Client)
	restclient.Transport = keepalive.NewHandlerREST(restclient, keepaliveInterval, restKeepAliveHandler(ctx, restclient))

	// Login activates the keep-alive handler
	if err := restclient.Login(ctx, parsedURL.User); err != nil {
		return nil, err
	}
	return restclient, nil
}

func restKeepAliveHandler(ctx context.Context, restclient *rest.Client) func() error {
	logger := logging.FromContext(ctx).With("rpc", "keepalive")

	return func() error {
		logger.Info("Executing REST keep-alive handler")
		ctx := context.Background()

		s, err := restclient.Session(ctx)
		if err != nil {
			return err
		}
		if s != nil {
			return nil
		}
		return errors.New(http.StatusText(http.StatusUnauthorized))
	}
}
