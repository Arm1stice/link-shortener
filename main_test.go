package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
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

func TestRootHandlerRoutesByPath(t *testing.T) {
	t.Parallel()

	handler := newRootHandler(nil, "wcal.xyz")

	cases := []struct {
		method string
		path   string
		status int
		body   string
	}{
		{http.MethodGet, "/", http.StatusOK, `<form id="shorten-form">`},
		{http.MethodPost, "/api/links", http.StatusBadRequest, "Request body must be valid JSON"},
		{http.MethodGet, "/api/links", http.StatusMethodNotAllowed, ""},
		{http.MethodGet, "/unknown/path", http.StatusNotFound, ""},
	}

	for _, host := range []string{"wcal.xyz", "wcal.xyz:5000", "WCAL.XYZ", "links.wcalandro.com"} {
		for _, tc := range cases {
			t.Run(host+"/"+tc.method+tc.path, func(t *testing.T) {
				req := httptest.NewRequest(tc.method, "http://"+host+tc.path, nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, req)

				if recorder.Code != tc.status || !strings.Contains(recorder.Body.String(), tc.body) {
					t.Fatalf("response = %d %q, want %d containing %q", recorder.Code, recorder.Body.String(), tc.status, tc.body)
				}
			})
		}
	}
}

func TestRootHandlerHealthzIsHostIndependent(t *testing.T) {
	t.Parallel()

	handler := newRootHandler(nil, "wcal.xyz")

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

func (createLinkTestDriver) Open(string) (driver.Conn, error) {
	return &createLinkTestConn{views: 7}, nil
}

type createLinkTestConn struct{ views int64 }

func (c *createLinkTestConn) Prepare(query string) (driver.Stmt, error) {
	if query != "SELECT * from links WHERE id = ?" && query != "UPDATE links SET views=views+1 WHERE id = ?" {
		return nil, errors.New("unexpected prepared statement")
	}
	return &createLinkTestStmt{conn: c, query: query}, nil
}
func (*createLinkTestConn) Close() error { return nil }
func (*createLinkTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}
func (*createLinkTestConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if query != "INSERT INTO links (url, views) VALUES (?, 0)" || len(args) != 1 || args[0].Value != "https://example.com/a" {
		return nil, errors.New("unexpected insert arguments")
	}
	return createLinkTestResult{}, nil
}

type createLinkTestStmt struct {
	conn  *createLinkTestConn
	query string
}

func (*createLinkTestStmt) Close() error  { return nil }
func (*createLinkTestStmt) NumInput() int { return 1 }
func (s *createLinkTestStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.query != "UPDATE links SET views=views+1 WHERE id = ?" || len(args) != 1 || args[0] != int64(303) {
		return nil, errors.New("unexpected update arguments")
	}
	s.conn.views++
	return createLinkTestResult{}, nil
}
func (s *createLinkTestStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.query != "SELECT * from links WHERE id = ?" || len(args) != 1 || args[0] != int64(303) {
		return nil, errors.New("unexpected select arguments")
	}
	return &createLinkTestRows{views: s.conn.views}, nil
}

type createLinkTestRows struct {
	views int64
	done  bool
}

func (*createLinkTestRows) Columns() []string { return []string{"id", "url", "views"} }
func (*createLinkTestRows) Close() error      { return nil }
func (r *createLinkTestRows) Next(values []driver.Value) error {
	if r.done {
		return io.EOF
	}
	values[0], values[1], values[2] = int64(303), "https://example.com/a", r.views
	r.done = true
	return nil
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

	db.SetMaxOpenConns(1)
	handler := newRootHandler(db, "wcal.xyz")
	req := httptest.NewRequest(http.MethodPost, "https://wcal.xyz/api/links", strings.NewReader(`{"url":"https://example.com/a"}`))
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

	for _, tc := range []struct {
		path     string
		status   int
		body     string
		location string
	}{
		{"/stats/4T", http.StatusOK, "Link: https://example.com/a | Views: 7", ""},
		{"/4T", http.StatusFound, "", "https://example.com/a"},
		{"/stats/4T", http.StatusOK, "Link: https://example.com/a | Views: 8", ""},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://wcal.xyz"+tc.path, nil))
		if recorder.Code != tc.status || !strings.Contains(recorder.Body.String(), tc.body) || recorder.Header().Get("Location") != tc.location {
			t.Fatalf("%s: response = %d %q, location %q; want %d containing %q, location %q", tc.path, recorder.Code, recorder.Body.String(), recorder.Header().Get("Location"), tc.status, tc.body, tc.location)
		}
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
