// Package fetcher provides website crawling and downloading functionality.
// This file implements robots.txt parsing and URL filtering.

package fetcher

import (
	"bufio"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RobotsChecker checks if a URL is allowed by robots.txt rules.
type RobotsChecker struct {
	cache      map[string]*robotsRules
	mu         sync.RWMutex
	userAgent  string
	httpClient *http.Client
}

// robotsRules holds parsed robots.txt rules for a domain.
type robotsRules struct {
	disallowRules []string
	allowRules    []string
	crawlDelay    time.Duration
	fetchedAt     time.Time
}

// NewRobotsChecker creates a new RobotsChecker with the specified user agent.
func NewRobotsChecker(userAgent string) *RobotsChecker {
	return &RobotsChecker{
		cache:     make(map[string]*robotsRules),
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
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
func (r *RobotsChecker) getRules(scheme, host string) *robotsRules {
	cacheKey := scheme + "://" + host

	// Check cache first
	r.mu.RLock()
	rules, exists := r.cache[cacheKey]
	r.mu.RUnlock()

	if exists {
		return rules
	}

	// Fetch robots.txt
	robotsURL := scheme + "://" + host + "/robots.txt"
	rules = r.fetchRobotsTxt(robotsURL)

	// Cache the result (even if nil)
	r.mu.Lock()
	r.cache[cacheKey] = rules
	r.mu.Unlock()

	return rules
}

// fetchRobotsTxt fetches and parses a robots.txt file.
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

	return r.parseRobotsTxt(resp.Body)
}

// parseRobotsTxt parses robots.txt content and extracts rules for our user agent.
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

// pathMatches checks if a path matches a robots.txt pattern.
// Supports * wildcard and $ end-of-string anchor.
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

// wildcardMatch handles patterns with * wildcards.
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
