package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
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
	data := &shortenMessage{SuccessMessages: []interface{}{"4T"}}
	if err := tmpl.Execute(&rendered, data); err != nil {
		t.Fatalf("execute embedded template: %v", err)
	}

	if !strings.Contains(rendered.String(), "https://wcal.xyz/4T") {
		t.Fatalf("rendered template does not contain legacy short URL: %s", rendered.String())
	}
}
