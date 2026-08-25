package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

// browserMutationProtection rejects browser-originated state-changing requests
// that are not same-origin. Direct API clients such as curl do not send Origin
// or Sec-Fetch-Site and remain usable; future non-cookie device credentials can
// therefore use the same API without browser-specific headers.
func browserMutationProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMutationMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		if fetchSite := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); fetchSite != "" {
			switch fetchSite {
			case "same-origin", "none":
				// Allowed browser navigation/request context.
			default:
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "same-origin request required"})
				return
			}
		}

		if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" && !requestOriginMatches(r, origin) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "same-origin request required"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isMutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func requestOriginMatches(r *http.Request, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil {
		return false
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return strings.EqualFold(u.Scheme, scheme) && strings.EqualFold(u.Host, r.Host)
}
