package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
	Store       RateLimitStore
}

type RateLimitStore interface {
	Increment(ctx context.Context, key string, window time.Duration) (RateLimitState, error)
}

type RateLimitState struct {
	Count      int
	ResetAfter time.Duration
}

type visitorState struct {
	Count     int
	ResetTime time.Time
}

func RateLimiter(config RateLimitConfig) gin.HandlerFunc {
	if config.MaxRequests <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}

	var mu sync.Mutex
	visitors := make(map[string]visitorState)
	lastCleanup := time.Now()

	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()

		if config.Store != nil {
			state, err := config.Store.Increment(c.Request.Context(), "rate_limit:"+key, config.Window)
			if err == nil {
				writeRateLimitHeaders(c, config.MaxRequests, config.MaxRequests-state.Count, state.ResetAfter)
				if state.Count > config.MaxRequests {
					retryAfter := int(state.ResetAfter.Seconds())
					if retryAfter < 0 {
						retryAfter = 0
					}
					c.Writer.Header().Set("Retry-After", strconv.Itoa(retryAfter))
					response.Error(c, http.StatusTooManyRequests, "too many requests")
					c.Abort()
					return
				}
				c.Next()
				return
			}
		}

		mu.Lock()
		if now.Sub(lastCleanup) > config.Window {
			for ip, state := range visitors {
				if now.After(state.ResetTime) {
					delete(visitors, ip)
				}
			}
			lastCleanup = now
		}

		state := visitors[key]
		if state.ResetTime.IsZero() || now.After(state.ResetTime) {
			state = visitorState{ResetTime: now.Add(config.Window)}
		}
		state.Count++
		visitors[key] = state

		remaining := config.MaxRequests - state.Count
		resetSeconds := int(time.Until(state.ResetTime).Seconds())
		if resetSeconds < 0 {
			resetSeconds = 0
		}
		mu.Unlock()

		writeRateLimitHeaders(c, config.MaxRequests, remaining, time.Duration(resetSeconds)*time.Second)

		if state.Count > config.MaxRequests {
			c.Writer.Header().Set("Retry-After", strconv.Itoa(resetSeconds))
			response.Error(c, http.StatusTooManyRequests, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}

func writeRateLimitHeaders(c *gin.Context, limit, remaining int, resetAfter time.Duration) {
	resetSeconds := int(resetAfter.Seconds())
	if resetSeconds < 0 {
		resetSeconds = 0
	}
	headers := c.Writer.Header()
	headers.Set("X-RateLimit-Limit", strconv.Itoa(limit))
	headers.Set("X-RateLimit-Remaining", strconv.Itoa(max(remaining, 0)))
	headers.Set("X-RateLimit-Reset", strconv.Itoa(resetSeconds))
}
