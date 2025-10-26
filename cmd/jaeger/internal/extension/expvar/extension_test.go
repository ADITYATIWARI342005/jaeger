// Copyright (c) 2024 The Jaeger Authors.
// SPDX-License-Identifier: Apache-2.0

package expvar

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/storagetest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.uber.org/zap/zaptest"
)

func TestExpvarExtension(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{
			name:   "good storage",
			status: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "0.0.0.0:27777",
				},
			}
			s := newExtension(config, component.TelemetrySettings{
				Logger: zaptest.NewLogger(t),
			})
			require.NoError(t, s.Start(context.Background(), storagetest.NewStorageHost()))
			defer s.Shutdown(context.Background())

			addr := fmt.Sprintf("http://0.0.0.0:%d/", Port)
			client := &http.Client{}
			require.Eventually(t, func() bool {
				r, err := http.NewRequest(http.MethodPost, addr, http.NoBody)
				require.NoError(t, err)
				resp, err := client.Do(r)
				require.NoError(t, err)
				defer resp.Body.Close()
				return test.status == resp.StatusCode
			}, 5*time.Second, 100*time.Millisecond)
		})
	}
}

func TestExpvarExtension_StartError(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "0.0.0.0:27777",
			Auth: configoptional.Some(confighttp.AuthConfig{
				Config: configauth.Config{
					AuthenticatorID: component.MustNewID("invalid_auth"),
				},
			}),
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})
	err := s.Start(context.Background(), storagetest.NewStorageHost())
	require.ErrorContains(t, err, "invalid_auth")
}

// TestExpvarExtension_ShutdownWithNilServer tests shutdown when server is nil
func TestExpvarExtension_ShutdownWithNilServer(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "0.0.0.0:27777",
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})
	// server is nil
	err := s.Shutdown(context.Background())
	require.NoError(t, err)
}

// TestExpvarExtension_ShutdownWithServer tests shutdown with active server
func TestExpvarExtension_ShutdownWithServer(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "0.0.0.0:27778", // Use different port to avoid conflicts
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})

	// Start the server
	err := s.Start(context.Background(), storagetest.NewStorageHost())
	require.NoError(t, err)

	// Shutdown should work without error
	err = s.Shutdown(context.Background())
	require.NoError(t, err)
}

// TestExpvarExtension_ShutdownWithTimeout tests shutdown with timeout context
func TestExpvarExtension_ShutdownWithTimeout(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "0.0.0.0:27779", // Use different port to avoid conflicts
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})

	// Start the server
	err := s.Start(context.Background(), storagetest.NewStorageHost())
	require.NoError(t, err)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Shutdown should work with timeout context
	err = s.Shutdown(ctx)
	require.NoError(t, err)
}

// TestExpvarExtension_StartWithInvalidEndpoint tests start with invalid endpoint
func TestExpvarExtension_StartWithInvalidEndpoint(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "invalid://endpoint",
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})
	err := s.Start(context.Background(), storagetest.NewStorageHost())
	require.Error(t, err)
}

// TestExpvarExtension_StartWithToServerError tests start when ToServer fails
func TestExpvarExtension_StartWithToServerError(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "0.0.0.0:27780",
			Auth: configoptional.Some(confighttp.AuthConfig{
				Config: configauth.Config{
					AuthenticatorID: component.MustNewID("nonexistent_auth"),
				},
			}),
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})
	err := s.Start(context.Background(), storagetest.NewStorageHost())
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent_auth")
}

// TestExpvarExtension_StartWithToListenerError tests start when ToListener fails
func TestExpvarExtension_StartWithToListenerError(t *testing.T) {
	config := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "invalid://listener",
		},
	}
	s := newExtension(config, component.TelemetrySettings{
		Logger: zaptest.NewLogger(t),
	})
	err := s.Start(context.Background(), storagetest.NewStorageHost())
	require.Error(t, err)
}
