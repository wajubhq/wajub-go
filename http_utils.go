package wajub

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var bearerPrefix = regexp.MustCompile(`(?i)^bearer\s+`)

func normalizeAPIKey(raw string) string {
	key := strings.TrimSpace(raw)
	if bearerPrefix.MatchString(key) {
		key = strings.TrimSpace(bearerPrefix.ReplaceAllString(key, ""))
	}
	return key
}

func defaultAPIURL() string {
	return APIURL
}

func createIdempotencyKey(prefix string) string {
	if prefix == "" {
		prefix = "wajub"
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// escapeID makes id a single path segment, so that a "/", "?" or "#" in it,
// or an id of "." or "..", cannot send the request to another route.
func escapeID(id string) string {
	if id == "." || id == ".." {
		return strings.Repeat("%2E", len(id))
	}
	return url.PathEscape(id)
}

func pickResource(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := body[key]; ok && value != nil {
			if m, ok := value.(map[string]any); ok {
				return m
			}
		}
	}
	return body
}

func pickList(body map[string]any, keys ...string) ([]map[string]any, map[string]any) {
	for _, key := range append(append([]string{}, keys...), "items") {
		if value, ok := body[key]; ok && value != nil {
			return toMapSlice(value), toMeta(body["meta"])
		}
	}
	return toMapSlice(body["data"]), toMeta(body["meta"])
}

func toMapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func toMeta(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func decodeJSON(data []byte) map[string]any {
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return map[string]any{}
	}
	return body
}
