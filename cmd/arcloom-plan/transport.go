package main

import (
	"errors"
	"net/http"
	"strings"
)

// authorizeGitHubRequest is concrete credential transport, not Plan meaning.
// Clone before adding credentials and reject any non-GitHub.com HTTPS origin.
func authorizeGitHubRequest(request *http.Request, token string) (*http.Request, error) {
	if request == nil || request.URL == nil || request.URL.Scheme != "https" || request.URL.Host != "api.github.com" || request.URL.User != nil || request.Method != http.MethodGet || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("invalid GitHub credential transport")
	}
	authorized := request.Clone(request.Context())
	if token != "" {
		authorized.Header.Set("Authorization", "Bearer "+token)
	}
	return authorized, nil
}
