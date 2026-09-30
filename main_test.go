package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/wcalandro/base62"
)

func TestLegacyBase62Compatibility(t *testing.T) {
	t.Parallel()

	cases := []struct {
		id   uint64
		code string
	}{
		{id: 1, code: "1"},
		{id: 61, code: "Z"},
		{id: 62, code: "10"},
		{id: 303, code: "4T"},
	}

	for _, tc := range cases {
		if got := base62.ToB62(tc.id); got != tc.code {
			t.Fatalf("base62.ToB62(%d) = %q, want %q", tc.id, got, tc.code)
		}

		got, err := base62.FromB62(tc.code)
		if err != nil {
			t.Fatalf("base62.FromB62(%q): %v", tc.code, err)
		}
		if got != tc.id {
			t.Fatalf("base62.FromB62(%q) = %d, want %d", tc.code, got, tc.id)
		}
	}
}

func TestRootHandlerRoutesByHost(t *testing.T) {
	t.Parallel()

	shortHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("short"))
	})
	websiteHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("website"))
	})
	handler := newRootHandler("wcal.xyz", shortHandler, websiteHandler)

	cases := []struct {
		name string
		host string
		want string
	}{
		{name: "short host", host: "wcal.xyz", want: "short"},
		{name: "short host with port", host: "wcal.xyz:5000", want: "short"},
		{name: "short host case insensitive", host: "WCAL.XYZ", want: "short"},
		{name: "website host", host: "links.wcalandro.com", want: "website"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/example", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if got := recorder.Body.String(); got != tc.want {
				t.Fatalf("body = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRootHandlerHealthzIsHostIndependent(t *testing.T) {
	t.Parallel()

	unreachable := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected application handler", http.StatusInternalServerError)
	})
	handler := newRootHandler("wcal.xyz", unreachable, unreachable)

	for _, host := range []string{"wcal.xyz", "links.wcalandro.com", "127.0.0.1:5000"} {
		req := httptest.NewRequest(http.MethodGet, "http://"+host+"/healthz", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("host %q status = %d, want %d", host, recorder.Code, http.StatusOK)
		}
		if got := recorder.Body.String(); got != "ok\n" {
			t.Fatalf("host %q body = %q, want %q", host, got, "ok\\n")
		}
	}
}

func TestEmbeddedIndexTemplate(t *testing.T) {
	t.Parallel()

	tmpl, err := loadIndexTemplate()
	if err != nil {
		t.Fatalf("loadIndexTemplate: %v", err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, nil); err != nil {
		t.Fatalf("execute embedded template: %v", err)
	}

	for _, want := range []string{
		`name="viewport"`,
		`fetch('/api/links'`,
		`id="result"`,
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Fatalf("rendered template does not contain %q", want)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "adds scheme", input: "example.com/path?q=1", want: "http://example.com/path?q=1"},
		{name: "preserves https", input: " https://example.com/path ", want: "https://example.com/path"},
		{name: "empty", input: "  ", wantErr: "URL field cannot be empty"},
		{name: "invalid", input: "not a url", wantErr: "Invalid URL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeURL(tc.input)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("normalizeURL(%q) error = %v, want %q", tc.input, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeURL(%q): %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("normalizeURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

type createLinkTestDriver struct{}

var registerCreateLinkTestDriver sync.Once

func (createLinkTestDriver) Open(string) (driver.Conn, error) { return createLinkTestConn{}, nil }

type createLinkTestConn struct{}

func (createLinkTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}
func (createLinkTestConn) Close() error { return nil }
func (createLinkTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}
func (createLinkTestConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 1 || args[0].Value != "https://example.com/a" {
		return nil, errors.New("unexpected insert arguments")
	}
	return createLinkTestResult{}, nil
}

type createLinkTestResult struct{}

func (createLinkTestResult) LastInsertId() (int64, error) { return 303, nil }
func (createLinkTestResult) RowsAffected() (int64, error) { return 1, nil }

func TestCreateLinkAPI(t *testing.T) {
	registerCreateLinkTestDriver.Do(func() {
		sql.Register("create-link-test", createLinkTestDriver{})
	})
	db, err := sql.Open("create-link-test", "")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	handler := websiteRouter(db, "wcal.xyz")
	req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"url":"https://example.com/a"}`))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var response createLinkResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != "4T" || response.ShortURL != "https://wcal.xyz/4T" {
		t.Fatalf("response = %+v", response)
	}
}

func TestCreateLinkAPIRejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	handler := websiteRouter(nil, "wcal.xyz")
	for _, body := range []string{
		`{"url":""}`,
		`{"url":"https://example.com","extra":true}`,
		`{"url":"https://example.com"} {}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(body))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %q status = %d, want %d", body, recorder.Code, http.StatusBadRequest)
		}
		if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
			t.Fatalf("content type = %q, want JSON", contentType)
		}
	}
}
