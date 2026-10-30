// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func Test_isBlocked(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantMethod  string
		wantBlocked bool
	}{
		{"users.list", "/api/users.list", "users.list", true},
		{"conversations.list", "/api/conversations.list", "conversations.list", true},
		{"users.conversations", "/api/users.conversations", "users.conversations", true},
		{"search prefix", "/api/search.messages", "search.messages", true},
		{"edge search", "/api/search.modules.channels", "search.modules.channels", true},
		{"client.userBoot", "/api/client.userBoot", "client.userBoot", true},
		{"emoji.list", "/api/emoji.list", "emoji.list", true},
		{"trailing slash", "/api/users.list/", "users.list", true},
		{"history allowed", "/api/conversations.history", "conversations.history", false},
		{"replies allowed", "/api/conversations.replies", "conversations.replies", false},
		{"info allowed", "/api/conversations.info", "conversations.info", false},
		{"members allowed", "/api/conversations.members", "conversations.members", false},
		{"users.info allowed", "/api/users.info", "users.info", false},
		{"auth.test allowed", "/api/auth.test", "auth.test", false},
		{"edge channel users allowed", "/cache/T123/users/list", "", false},
		{"file download allowed", "/files-pri/T123-F123/file.png", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMethod, gotBlocked := isBlocked(tt.path)
			if gotMethod != tt.wantMethod || gotBlocked != tt.wantBlocked {
				t.Errorf("isBlocked(%q) = (%q, %v), want (%q, %v)", tt.path, gotMethod, gotBlocked, tt.wantMethod, tt.wantBlocked)
			}
		})
	}
}

type fakeTransport struct {
	calls  int
	closed bool
}

func (f *fakeTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.calls++
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func (f *fakeTransport) Close() error {
	f.closed = true
	return nil
}

func Test_guardTransport_RoundTrip(t *testing.T) {
	t.Run("blocked request is not sent", func(t *testing.T) {
		ft := &fakeTransport{}
		g := &guardTransport{rt: ft, lim: rate.NewLimiter(rate.Inf, 1)}
		req, _ := http.NewRequest(http.MethodPost, "https://slack.com/api/users.list", nil)
		_, err := g.RoundTrip(req)
		if !errors.Is(err, ErrListingBlocked) {
			t.Fatalf("want ErrListingBlocked, got %v", err)
		}
		if ft.calls != 0 {
			t.Errorf("underlying transport called %d times, want 0", ft.calls)
		}
	})
	t.Run("allowed request is sent", func(t *testing.T) {
		ft := &fakeTransport{}
		g := &guardTransport{rt: ft, lim: rate.NewLimiter(rate.Inf, 1)}
		req, _ := http.NewRequest(http.MethodPost, "https://slack.com/api/conversations.history", nil)
		resp, err := g.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resp.Body.Close()
		if ft.calls != 1 {
			t.Errorf("underlying transport called %d times, want 1", ft.calls)
		}
	})
	t.Run("requests are paced", func(t *testing.T) {
		ft := &fakeTransport{}
		g := &guardTransport{rt: ft, lim: rate.NewLimiter(rate.Every(time.Hour), 1)}
		req, _ := http.NewRequest(http.MethodPost, "https://slack.com/api/conversations.history", nil)
		if _, err := g.RoundTrip(req); err != nil {
			t.Fatalf("first request: unexpected error: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if _, err := g.RoundTrip(req.WithContext(ctx)); err == nil {
			t.Fatal("second request: want pacing error, got nil")
		}
		if ft.calls != 1 {
			t.Errorf("underlying transport called %d times, want 1", ft.calls)
		}
	})
}

func Test_guardTransport_Close(t *testing.T) {
	ft := &fakeTransport{}
	g := GuardTransport(ft).(*guardTransport)
	if err := g.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ft.closed {
		t.Error("underlying transport was not closed")
	}
}
