package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drywaters/dejaview/internal/session"
)

func TestLoginSetsSignedSessionCookie(t *testing.T) {
	sessionManager := session.NewManager("secret", 90*24*time.Hour, false)
	handler := NewAuthHandler("secret", sessionManager)

	form := url.Values{}
	form.Set("api_key", "secret")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.Login(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, recorder.Code)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if cookies[0].Value == "secret" {
		t.Fatal("session cookie stored raw API token")
	}
	if !sessionManager.Valid(cookies[0].Value) {
		t.Fatal("session cookie did not validate")
	}
}

func TestLoginPageRejectsLegacyRawTokenCookie(t *testing.T) {
	sessionManager := session.NewManager("secret", 90*24*time.Hour, false)
	handler := NewAuthHandler("secret", sessionManager)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "secret"})
	recorder := httptest.NewRecorder()

	handler.LoginPage(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestLoginRedirectValidation(t *testing.T) {
	cases := []struct {
		destination string
		valid       bool
	}{
		{"/", true}, {"/entries/123?filter=watched#ratings", true}, {"/search?q=https%3A%2F%2Fexample.com", true},
		{"", false}, {"https://example.com", false}, {"//example.com", false}, {`/\example.com`, false},
		{"/%5cexample.com", false}, {"/%2fexample.com", false}, {"/%00bad", false}, {"/bad\npath", false},
		{"/%zz", false}, {"relative", false},
	}
	handler := NewAuthHandler("secret", session.NewManager("secret", time.Hour, false))
	for _, tc := range cases {
		t.Run(tc.destination, func(t *testing.T) {
			if got := isValidRedirect(tc.destination); got != tc.valid {
				t.Fatalf("valid=%v want=%v", got, tc.valid)
			}
			form := url.Values{"api_key": {"secret"}, "redirect": {tc.destination}}
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			result := httptest.NewRecorder()
			handler.Login(result, req)
			want := "/"
			if tc.valid {
				want = tc.destination
			}
			if result.Code != http.StatusSeeOther || result.Header().Get("Location") != want {
				t.Fatalf("status=%d redirect=%q", result.Code, result.Header().Get("Location"))
			}
		})
	}
}
