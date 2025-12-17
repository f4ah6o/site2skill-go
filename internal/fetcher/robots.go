// Package fetcher provides website crawling and downloading functionality.
// This file implements robots.txt parsing and URL filtering.

package fetcher

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RobotsChecker checks if URLs are allowed by robots.txt rules.
// It fetches, parses, and caches robots.txt files for each domain, applying
// user-agent-specific rules to determine crawl permissions. Thread-safe for
// concurrent use.
type RobotsChecker struct {
	// cache stores parsed robots.txt rules indexed by domain and basePath
	cache map[string]*robotsRules
	// mu protects concurrent access to the cache
	mu sync.RWMutex
	// userAgent identifies this crawler in robots.txt matching
	userAgent string
	// httpClient is used to fetch robots.txt files
	httpClient *http.Client
	// basePath is the base path for subdirectory deployments (e.g., "/site2skill-go")
	// Used to support GitHub Pages and similar hosting where robots.txt may be
	// located at a subdirectory rather than the root
	basePath string
}

// robotsRules holds parsed robots.txt directives for a specific domain and user agent.
// It stores both allow and disallow patterns as well as optional crawl delay settings.
type robotsRules struct {
	// disallowRules contains path patterns that should not be crawled
	disallowRules []string
	// allowRules contains path patterns that are explicitly allowed
	// (used to override broader disallow rules)
	allowRules []string
	// crawlDelay specifies the minimum time between requests (not currently enforced)
	crawlDelay time.Duration
	// fetchedAt records when these rules were retrieved
	fetchedAt time.Time
}

// NewRobotsChecker creates a new RobotsChecker configured with the specified user agent string.
// The user agent is used to match User-agent directives in robots.txt files.
//
// Parameters:
//   - userAgent: The user agent string to identify this crawler (e.g., "MyBot/1.0")
//
// Returns a new RobotsChecker instance ready for use. The checker is thread-safe
// and automatically caches robots.txt files to minimize network requests.
func NewRobotsChecker(userAgent string) *RobotsChecker {
	return &RobotsChecker{
		cache:     make(map[string]*robotsRules),
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetBasePath configures the base path for subdirectory deployments like GitHub Pages.
// This is necessary for sites hosted in subdirectories where robots.txt may be located
// at a path like /project-name/robots.txt instead of /robots.txt.
//
// When set, if /robots.txt returns 404, the checker will try basePath/robots.txt as a fallback.
// For example, if basePath is "/site2skill-go", the checker will try:
//   1. https://example.com/robots.txt
//   2. https://example.com/site2skill-go/robots.txt (if #1 fails)
//
// Parameters:
//   - basePath: The base path (e.g., "/site2skill-go"). Will be normalized to ensure
//     it starts with "/" and doesn't end with "/". Pass empty string to disable.
func (r *RobotsChecker) SetBasePath(basePath string) {
	// Normalize basePath: ensure it starts with / and doesn't end with /
	if basePath != "" {
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		basePath = strings.TrimSuffix(basePath, "/")
	}
	r.basePath = basePath
}

// IsAllowed checks if the given URL is allowed by robots.txt.
// It fetches and caches the robots.txt for the domain if not already cached.
// When multiple rules match, the longer (more specific) rule takes precedence.
func (r *RobotsChecker) IsAllowed(targetURL string) bool {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return true // Allow on parse error
	}

	rules := r.getRules(parsedURL.Scheme, parsedURL.Host)
	if rules == nil {
		return true // Allow if no rules found
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Find the longest matching rule (more specific rules take precedence)
	allowed := true
	matchedLen := 0

	for _, rule := range rules.allowRules {
		if r.pathMatches(path, rule) && len(rule) > matchedLen {
			allowed = true
			matchedLen = len(rule)
		}
	}

	for _, rule := range rules.disallowRules {
		if r.pathMatches(path, rule) && len(rule) > matchedLen {
			allowed = false
			matchedLen = len(rule)
		}
	}

	return allowed
}

// getRules fetches or retrieves cached robots.txt rules for a domain.
// It tries the root /robots.txt first, then falls back to basePath/robots.txt
// for subdirectory deployments like GitHub Pages.
func (r *RobotsChecker) getRules(scheme, host string) *robotsRules {
	cacheKey := scheme + "://" + host
	if r.basePath != "" {
		cacheKey += r.basePath
	}

	// Check cache first
	r.mu.RLock()
	rules, exists := r.cache[cacheKey]
	r.mu.RUnlock()

	if exists {
		return rules
	}

	// Try fetching robots.txt from root first
	robotsURL := scheme + "://" + host + "/robots.txt"
	rules = r.fetchRobotsTxt(robotsURL)

	// If root robots.txt not found and basePath is set, try basePath/robots.txt
	if rules == nil && r.basePath != "" {
		subDirRobotsURL := scheme + "://" + host + r.basePath + "/robots.txt"
		log.Printf("Root robots.txt not found, trying subdirectory: %s", subDirRobotsURL)
		rules = r.fetchRobotsTxt(subDirRobotsURL)
	}

	// Cache the result (even if nil)
	r.mu.Lock()
	r.cache[cacheKey] = rules
	r.mu.Unlock()

	return rules
}

// fetchRobotsTxt fetches and parses a robots.txt file from the specified URL.
// It performs an HTTP GET request and parses the response if successful.
//
// Parameters:
//   - robotsURL: The complete URL to the robots.txt file
//
// Returns the parsed robotsRules, or nil if the file doesn't exist (404) or cannot be fetched.
// A nil return value indicates all URLs should be allowed (permissive behavior).
func (r *RobotsChecker) fetchRobotsTxt(robotsURL string) *robotsRules {
	resp, err := r.httpClient.Get(robotsURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// If robots.txt doesn't exist or is inaccessible, allow all
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	log.Printf("Successfully fetched robots.txt from %s", robotsURL)
	return r.parseRobotsTxt(resp.Body)
}

// parseRobotsTxt parses robots.txt content from a reader and extracts rules for the configured user agent.
// It implements the robots.txt standard, supporting User-agent, Disallow, Allow, and Crawl-delay directives.
//
// The parser handles both user-agent-specific rules and wildcard (*) rules, preferring
// specific rules when available and falling back to wildcard rules otherwise.
//
// Parameters:
//   - reader: An io.Reader providing the robots.txt content
//
// Returns a robotsRules struct containing the parsed directives applicable to this user agent.
func (r *RobotsChecker) parseRobotsTxt(reader io.Reader) *robotsRules {
	rules := &robotsRules{
		fetchedAt: time.Now(),
	}

	scanner := bufio.NewScanner(reader)
	var currentUserAgent string
	matchesUs := false
	wildcardRules := &robotsRules{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key: value pairs
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(line[:colonIdx]))
		value := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "user-agent":
			currentUserAgent = strings.ToLower(value)
			// Check if this applies to us
			if currentUserAgent == "*" {
				matchesUs = false // Will use as fallback
			} else if strings.Contains(strings.ToLower(r.userAgent), currentUserAgent) ||
				currentUserAgent == strings.ToLower(r.userAgent) {
				matchesUs = true
			} else {
				matchesUs = false
			}

		case "disallow":
			if value == "" {
				continue // Empty disallow means allow all
			}
			if matchesUs {
				rules.disallowRules = append(rules.disallowRules, value)
			} else if currentUserAgent == "*" {
				wildcardRules.disallowRules = append(wildcardRules.disallowRules, value)
			}

		case "allow":
			if matchesUs {
				rules.allowRules = append(rules.allowRules, value)
			} else if currentUserAgent == "*" {
				wildcardRules.allowRules = append(wildcardRules.allowRules, value)
			}

		case "crawl-delay":
			// Parse crawl delay (optional)
			// Not implemented for now
		}
	}

	// If no specific rules for our user agent, use wildcard rules
	if len(rules.disallowRules) == 0 && len(rules.allowRules) == 0 {
		rules.disallowRules = wildcardRules.disallowRules
		rules.allowRules = wildcardRules.allowRules
	}

	return rules
}

// pathMatches checks if a URL path matches a robots.txt pattern according to the robots.txt standard.
// It supports:
//   - * wildcard: matches any sequence of characters
//   - $ anchor: matches end of path (must be at end of pattern)
//   - Prefix matching: patterns without wildcards match path prefixes
//
// Parameters:
//   - path: The URL path to test (e.g., "/docs/api/index.html")
//   - pattern: The robots.txt pattern (e.g., "/docs/*", "/admin$", "/api/")
//
// Returns true if the path matches the pattern according to robots.txt rules.
//
// Examples:
//   pathMatches("/docs/api", "/docs/") -> true (prefix match)
//   pathMatches("/docs/api", "/docs/*") -> true (wildcard match)
//   pathMatches("/docs", "/docs$") -> true (exact match with anchor)
//   pathMatches("/docs/api", "/docs$") -> false (anchor doesn't match)
func (r *RobotsChecker) pathMatches(path, pattern string) bool {
	if pattern == "" {
		return false
	}

	// Handle $ anchor at end
	mustMatchEnd := false
	if strings.HasSuffix(pattern, "$") {
		mustMatchEnd = true
		pattern = pattern[:len(pattern)-1]
	}

	// Handle * wildcards
	if strings.Contains(pattern, "*") {
		return r.wildcardMatch(path, pattern, mustMatchEnd)
	}

	// Simple prefix matching
	if mustMatchEnd {
		return path == pattern
	}
	return strings.HasPrefix(path, pattern)
}

// wildcardMatch matches a path against a pattern containing * wildcards.
// It splits the pattern by asterisks and ensures each non-wildcard part appears
// in order in the path.
//
// Parameters:
//   - path: The URL path to test
//   - pattern: The pattern containing one or more * wildcards
//   - mustMatchEnd: If true, the match must consume the entire path ($ anchor)
//
// Returns true if the path matches the wildcard pattern.
//
// Example:
//   wildcardMatch("/api/v1/users", "/api/*/users", false) -> true
//   wildcardMatch("/api/v1/users", "/api/*.html", false) -> false
func (r *RobotsChecker) wildcardMatch(path, pattern string, mustMatchEnd bool) bool {
	parts := strings.Split(pattern, "*")

	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}

		idx := strings.Index(path[pos:], part)
		if idx == -1 {
			return false
		}

		// First part must match at start if there's no leading *
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}

		pos += idx + len(part)
	}

	// Check end anchor
	if mustMatchEnd && pos != len(path) {
		return false
	}

	return true
}
