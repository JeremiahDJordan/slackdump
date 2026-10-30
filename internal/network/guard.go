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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// This build runs in single-channel mode: every Slack request passes through
// GuardTransport, which refuses workspace-wide listing and search endpoints and
// paces the remaining requests to roughly what a person browsing a channel in
// the Slack client would generate.

const (
	// PaceInterval is the minimum sustained interval between two requests to
	// Slack, across all clients and workers in the process.
	PaceInterval = 2 * time.Second
	// PaceBurst is how many requests may go out back-to-back before the
	// PaceInterval spacing applies.
	PaceBurst = 3
)

// ErrListingBlocked is returned for requests to workspace-wide listing or
// search endpoints.
var ErrListingBlocked = errors.New("workspace-wide listing and search are disabled in single-channel mode")

// blockedMethods are Slack Web API (and web client API) methods that enumerate
// the whole workspace rather than a single conversation.
var blockedMethods = map[string]struct{}{
	"users.list":          {},
	"users.conversations": {},
	"conversations.list":  {},
	"client.userBoot":     {},
	"client.counts":       {},
	"client.dms":          {},
	"im.list":             {},
	"mpim.list":           {},
	"files.list":          {},
	"emoji.list":          {},
	"emoji.adminList":     {},
	"admin.emoji.list":    {},
}

// pacer is shared by all guarded transports, so that the overall request
// rate stays bounded regardless of how many clients or workers are running.
var pacer = rate.NewLimiter(rate.Every(PaceInterval), PaceBurst)

// GuardTransport wraps rt with the single-channel mode policy.  If rt
// implements io.Closer, so does the returned transport.
func GuardTransport(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &guardTransport{rt: rt, lim: pacer}
}

type guardTransport struct {
	rt  http.RoundTripper
	lim *rate.Limiter
}

func (g *guardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if method, blocked := isBlocked(req.URL.Path); blocked {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, fmt.Errorf("%s: %w", method, ErrListingBlocked)
	}
	if err := g.lim.Wait(req.Context()); err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	return g.rt.RoundTrip(req)
}

// Close closes the underlying transport, if it is closable.
func (g *guardTransport) Close() error {
	if c, ok := g.rt.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// isBlocked reports whether the request path addresses a blocked API method,
// and returns the method name.
func isBlocked(path string) (string, bool) {
	i := strings.LastIndex(path, "/api/")
	if i < 0 {
		return "", false
	}
	method := strings.TrimSuffix(path[i+len("/api/"):], "/")
	if strings.HasPrefix(method, "search.") {
		return method, true
	}
	_, blocked := blockedMethods[method]
	return method, blocked
}
