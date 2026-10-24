package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BrunoKrugel/echo-mcp/pkg/transport"
	"github.com/BrunoKrugel/echo-mcp/pkg/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// callTool runs initialize + tools/call through the mounted HTTP transport.
func callTool(t *testing.T, e *echo.Echo, tool string, args map[string]any, headers map[string]string) string {
	t.Helper()
	post := func(body string, session string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		if session != "" {
			req.Header.Set("Mcp-Session-Id", session)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	init := post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, "")
	require.Equal(t, http.StatusOK, init.Code)
	session := init.Header().Get("Mcp-Session-Id")

	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)
	rec := post(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"`+tool+`","arguments":`+string(argsJSON)+`}}`, session)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Result.Content)
	return resp.Result.Content[0].Text
}

func newForwardingServer(t *testing.T, forward []string) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.GET("/whoami", func(c echo.Context) error {
		return c.String(http.StatusOK, "auth="+c.Request().Header.Get("Authorization"))
	})
	mcp := NewWithConfig(e, &Config{ForwardHeaders: forward})
	require.NoError(t, mcp.Mount("/mcp"))
	return e
}

func TestForwardHeaders(t *testing.T) {
	t.Run("configured header reaches the tool request", func(t *testing.T) {
		e := newForwardingServer(t, []string{"Authorization"})
		got := callTool(t, e, "GET_whoami", map[string]any{}, map[string]string{"Authorization": "Bearer abc"})
		assert.Equal(t, "auth=Bearer abc", got)
	})

	t.Run("headers are not forwarded by default", func(t *testing.T) {
		e := newForwardingServer(t, nil)
		got := callTool(t, e, "GET_whoami", map[string]any{}, map[string]string{"Authorization": "Bearer abc"})
		assert.Equal(t, "auth=", got)
	})

	t.Run("absent incoming header is not invented", func(t *testing.T) {
		e := newForwardingServer(t, []string{"Authorization"})
		got := callTool(t, e, "GET_whoami", map[string]any{}, nil)
		assert.Equal(t, "auth=", got)
	})
}

func TestForwardedHeaderOverridesToolArgument(t *testing.T) {
	e := echo.New()
	e.GET("/whoami", func(c echo.Context) error {
		return c.String(http.StatusOK, "auth="+c.Request().Header.Get("Authorization"))
	})
	mcp := NewWithConfig(e, &Config{ForwardHeaders: []string{"Authorization"}})
	mcp.operations = map[string]types.Operation{
		"GET_whoami": {Method: http.MethodGet, Path: "/whoami", HeaderParams: []string{"Authorization"}},
	}

	incoming := httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody)
	incoming.Header.Set("Authorization", "Bearer real")
	ctx := transport.ContextWithRequest(incoming.Context(), incoming)

	got, err := mcp.defaultExecuteTool(ctx, "GET_whoami", map[string]any{"Authorization": "Bearer spoofed"})
	require.NoError(t, err)
	assert.Equal(t, "auth=Bearer real", got)
}
