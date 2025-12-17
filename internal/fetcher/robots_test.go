package fetcher

import (
	"strings"
	"testing"
)

func TestRobotsChecker_PathMatches(t *testing.T) {
	r := NewRobotsChecker("test-bot")

	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		// Simple prefix matching
		{"exact match", "/docs/ng/", "/docs/ng/", true},
		{"prefix match", "/docs/ng/page.html", "/docs/ng/", true},
		{"no match", "/docs/ok/", "/docs/ng/", false},
		{"root match", "/", "/", true},

		// Wildcard matching
		{"wildcard middle", "/docs/test/page.html", "/docs/*/page.html", true},
		{"wildcard end", "/docs/anything", "/docs/*", true},
		{"wildcard start", "/anything/docs", "*/docs", true},

		// End anchor matching
		{"end anchor match", "/page.html", "/page.html$", true},
		{"end anchor no match", "/page.html/extra", "/page.html$", false},

		// Combined
		{"wildcard with anchor", "/docs/test.html", "/*.html$", true},
		{"wildcard with anchor no match", "/docs/test.html/", "/*.html$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.pathMatches(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("pathMatches(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestRobotsChecker_ParseRobotsTxt(t *testing.T) {
	r := NewRobotsChecker("site2skillgo")

	tests := []struct {
		name        string
		content     string
		testPath    string
		wantAllowed bool
	}{
		{
			name: "simple disallow",
			content: `User-agent: *
Disallow: /private/
`,
			testPath:    "/private/page.html",
			wantAllowed: false,
		},
		{
			name: "allow takes precedence",
			content: `User-agent: *
Disallow: /docs/
Allow: /docs/public/
`,
			testPath:    "/docs/public/page.html",
			wantAllowed: true,
		},
		{
			name: "empty disallow means allow all",
			content: `User-agent: *
Disallow:
`,
			testPath:    "/anything/page.html",
			wantAllowed: true,
		},
		{
			name: "specific user agent",
			content: `User-agent: Googlebot
Disallow: /

User-agent: *
Disallow: /private/
`,
			testPath:    "/public/page.html",
			wantAllowed: true,
		},
		{
			name: "our user agent blocked",
			content: `User-agent: site2skillgo
Disallow: /blocked/

User-agent: *
Disallow:
`,
			testPath:    "/blocked/page.html",
			wantAllowed: false,
		},
		{
			name: "github pages subdirectory",
			content: `User-agent: *
Disallow: /site2skill-go/docs/ng/
Disallow: /site2skill-go/docs/ja/forbidden/
`,
			testPath:    "/site2skill-go/docs/ng/index.html",
			wantAllowed: false,
		},
		{
			name: "github pages allowed path",
			content: `User-agent: *
Disallow: /site2skill-go/docs/ng/
`,
			testPath:    "/site2skill-go/docs/getting-started/",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := r.parseRobotsTxt(strings.NewReader(tt.content))
			if rules == nil {
				t.Fatal("parseRobotsTxt returned nil")
			}

			// Manually check the path using the same logic as IsAllowed
			// (longer matching rule takes precedence)
			allowed := true
			matchedLen := 0

			for _, rule := range rules.allowRules {
				if r.pathMatches(tt.testPath, rule) && len(rule) > matchedLen {
					allowed = true
					matchedLen = len(rule)
				}
			}
			for _, rule := range rules.disallowRules {
				if r.pathMatches(tt.testPath, rule) && len(rule) > matchedLen {
					allowed = false
					matchedLen = len(rule)
				}
			}

			if allowed != tt.wantAllowed {
				t.Errorf("path %q: got allowed=%v, want %v", tt.testPath, allowed, tt.wantAllowed)
				t.Logf("disallow rules: %v", rules.disallowRules)
				t.Logf("allow rules: %v", rules.allowRules)
			}
		})
	}
}

func TestNewRobotsChecker(t *testing.T) {
	r := NewRobotsChecker("test-bot")
	if r == nil {
		t.Error("NewRobotsChecker returned nil")
	}
	if r.userAgent != "test-bot" {
		t.Errorf("userAgent = %q, want %q", r.userAgent, "test-bot")
	}
}

func TestRobotsChecker_SetBasePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with leading slash", "/site2skill-go", "/site2skill-go"},
		{"without leading slash", "site2skill-go", "/site2skill-go"},
		{"with trailing slash", "/site2skill-go/", "/site2skill-go"},
		{"both slashes", "site2skill-go/", "/site2skill-go"},
		{"empty string", "", ""},
		{"just slash", "/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRobotsChecker("test-bot")
			r.SetBasePath(tt.input)
			if r.basePath != tt.expected {
				t.Errorf("SetBasePath(%q): got basePath=%q, want %q", tt.input, r.basePath, tt.expected)
			}
		})
	}
}
