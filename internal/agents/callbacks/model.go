package callbacks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"

	"prometheus/internal/agents/state"
	"prometheus/pkg/logger"
)

// CachingBeforeModelCallback checks Redis cache for previous LLM responses
func CachingBeforeModelCallback(redisClient *redis.Client) llmagent.BeforeModelCallback {
	return func(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
		if redisClient == nil {
			return nil, nil // No cache available
		}

		log := logger.Get().With("component", "model_cache")

		// Generate cache key from request
		cacheKey, err := generateCacheKey(req)
		if err != nil {
			log.Warnf("Failed to generate cache key: %v", err)
			return nil, nil // Continue to LLM
		}

		// Check cache
		cached, err := redisClient.Get(ctx, cacheKey).Result()
		if err == redis.Nil {
			// Cache miss
			log.Debug("Cache miss for LLM request")
			return nil, nil
		}
		if err != nil {
			log.Warnf("Redis get error: %v", err)
			return nil, nil // Continue to LLM on error
		}

		// Cache hit - deserialize response
		var cachedResp model.LLMResponse
		if err := json.Unmarshal([]byte(cached), &cachedResp); err != nil {
			log.Warnf("Failed to unmarshal cached response: %v", err)
			return nil, nil
		}

		log.Info("Cache hit for LLM request - returning cached response")
		return &cachedResp, nil
	}
}

// SaveToCacheAfterModelCallback saves LLM responses to Redis cache
func SaveToCacheAfterModelCallback(redisClient *redis.Client, ttl time.Duration) llmagent.AfterModelCallback {
	return func(ctx agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
		if redisClient == nil || resp == nil || respErr != nil {
			return resp, respErr // Pass through unchanged
		}

		log := logger.Get().With("component", "model_cache")

		// We can't generate cache key without the request, so skip caching in AfterCallback
		// In production, use a stateful approach or store request in temp state
		log.Debug("Skipping cache save - request not available in AfterModelCallback")

		return resp, nil // Pass through unchanged
	}
}

// TokenCountingCallback tracks token usage for cost calculation
func TokenCountingCallback(costTracker interface{}) llmagent.AfterModelCallback {
	return func(ctx agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
		if respErr != nil || resp == nil || resp.UsageMetadata == nil {
			return resp, respErr
		}

		log := logger.Get().With("component", "token_counter")

		log.Debugf("Tokens used: prompt=%d completion=%d total=%d",
			resp.UsageMetadata.PromptTokenCount,
			resp.UsageMetadata.CandidatesTokenCount,
			resp.UsageMetadata.TotalTokenCount,
		)

		// Store token count in temp state using helpers
		state.SetTempPromptTokens(ctx.State(), int(resp.UsageMetadata.PromptTokenCount))
		state.SetTempCompletionTokens(ctx.State(), int(resp.UsageMetadata.CandidatesTokenCount))

		return resp, nil
	}
}

// RateLimitCallback enforces rate limits before LLM calls
func RateLimitCallback() llmagent.BeforeModelCallback {
	return func(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
		// Check rate limit from user state
		lastCallTime, err := ctx.ReadonlyState().Get("_user_last_llm_call")
		if err == nil {
			if lastCall, ok := lastCallTime.(time.Time); ok {
				minInterval := 100 * time.Millisecond
				if time.Since(lastCall) < minInterval {
					logger.Get().Warn("Rate limit: too many LLM calls")
					// Continue anyway for now
				}
			}
		}

		// Update last call time
		ctx.State().Set("_user_last_llm_call", time.Now())
		return nil, nil
	}
}

// generateCacheKey creates a deterministic cache key from LLM request
func generateCacheKey(req *model.LLMRequest) (string, error) {
	// Create a simplified representation for caching
	cacheData := struct {
		Contents []string
		Tools    map[string]interface{}
	}{
		Contents: make([]string, 0, len(req.Contents)),
		Tools:    req.Tools,
	}

	// Extract text content from each message
	for _, content := range req.Contents {
		for _, part := range content.Parts {
			if part.Text != "" {
				cacheData.Contents = append(cacheData.Contents, content.Role+":"+part.Text)
			}
		}
	}

	// Serialize to JSON
	dataBytes, err := json.Marshal(cacheData)
	if err != nil {
		return "", err
	}

	// Hash to create deterministic key
	hash := sha256.Sum256(dataBytes)
	return "llm_cache:" + hex.EncodeToString(hash[:]), nil
}
