package middleware

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var defaultAllowedOrigins = map[string]struct{}{
	"http://localhost:5173": {},
	"http://localhost:5174": {},
	"http://127.0.0.1:5173": {},
	"http://127.0.0.1:5174": {},
}

func CORS() gin.HandlerFunc {
	configuredOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin, configuredOrigins) {
			headers := c.Writer.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Set("Access-Control-Allow-Credentials", "true")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			headers.Set("Access-Control-Allow-Headers", allowedHeaders(c))
			headers.Set("Access-Control-Max-Age", "86400")
			headers.Set("Access-Control-Allow-Private-Network", "true")
			headers.Add("Vary", "Origin")
			headers.Add("Vary", "Access-Control-Request-Headers")
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !isAllowedOrigin(origin, configuredOrigins) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func parseAllowedOrigins(value string) map[string]struct{} {
	origins := make(map[string]struct{}, len(defaultAllowedOrigins))
	for origin := range defaultAllowedOrigins {
		origins[origin] = struct{}{}
	}

	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}

	return origins
}

func isAllowedOrigin(origin string, configuredOrigins map[string]struct{}) bool {
	if origin == "" {
		return false
	}
	if _, ok := configuredOrigins[origin]; ok {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func allowedHeaders(c *gin.Context) string {
	requested := strings.TrimSpace(c.GetHeader("Access-Control-Request-Headers"))
	if requested != "" {
		return requested
	}

	return "Authorization, Content-Type, X-Request-ID"
}
