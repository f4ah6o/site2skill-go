package fetcher

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractLocale_PathFormat(t *testing.T) {
	tests := []struct {
		name           string
		urlStr         string
		wantLocale     string
		wantCanonical  string
	}{
		{
			name:          "Japanese locale",
			urlStr:        "https://example.com/ja/docs/api",
			wantLocale:    "ja",
			wantCanonical: "/docs/api",
		},
		{
			name:          "English locale",
			urlStr:        "https://example.com/en/docs/api",
			wantLocale:    "en",
			wantCanonical: "/docs/api",
		},
		{
			name:          "Chinese Traditional",
			urlStr:        "https://example.com/zh-tw/docs/api",
			wantLocale:    "zh-tw",
			wantCanonical: "/docs/api",
		},
		{
			name:          "Root path with locale",
			urlStr:        "https://example.com/ja/",
			wantLocale:    "ja",
			wantCanonical: "/",
		},
		{
			name:          "No locale in path",
			urlStr:        "https://example.com/docs/api",
			wantLocale:    "",
			wantCanonical: "/docs/api",
		},
		{
			name:          "Unknown first segment (not a locale)",
			urlStr:        "https://example.com/api/v1/users",
			wantLocale:    "",
			wantCanonical: "/api/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.urlStr)
			locale, canonical := ExtractLocale(u, nil)
			if locale != tt.wantLocale {
				t.Errorf("ExtractLocale() locale = %q, want %q", locale, tt.wantLocale)
			}
			if canonical != tt.wantCanonical {
				t.Errorf("ExtractLocale() canonical = %q, want %q", canonical, tt.wantCanonical)
			}
		})
	}
}

func TestExtractLocale_QueryFormat(t *testing.T) {
	tests := []struct {
		name          string
		urlStr        string
		paramName     string
		wantLocale    string
		wantCanonical string
	}{
		{
			name:          "hl parameter - English",
			urlStr:        "https://ai.google.dev/docs?hl=en",
			paramName:     "hl",
			wantLocale:    "en",
			wantCanonical: "/docs",
		},
		{
			name:          "hl parameter - Japanese",
			urlStr:        "https://ai.google.dev/docs?hl=ja",
			paramName:     "hl",
			wantLocale:    "ja",
			wantCanonical: "/docs",
		},
		{
			name:          "lang parameter",
			urlStr:        "https://example.com/docs?lang=de&page=1",
			paramName:     "lang",
			wantLocale:    "de",
			wantCanonical: "/docs",
		},
		{
			name:          "No locale parameter",
			urlStr:        "https://example.com/docs?page=1",
			paramName:     "hl",
			wantLocale:    "",
			wantCanonical: "/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.urlStr)
			cfg := &LocaleConfig{ParamName: tt.paramName}
			locale, canonical := ExtractLocale(u, cfg)
			if locale != tt.wantLocale {
				t.Errorf("ExtractLocale() locale = %q, want %q", locale, tt.wantLocale)
			}
			if canonical != tt.wantCanonical {
				t.Errorf("ExtractLocale() canonical = %q, want %q", canonical, tt.wantCanonical)
			}
		})
	}
}

func TestBuildLocaleURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		locale    string
		canonical string
		cfg       *LocaleConfig
		want      string
	}{
		{
			name:      "Path format - Japanese",
			baseURL:   "https://example.com",
			locale:    "ja",
			canonical: "/docs/api",
			cfg:       nil,
			want:      "https://example.com/ja/docs/api",
		},
		{
			name:      "Path format - no locale",
			baseURL:   "https://example.com",
			locale:    "",
			canonical: "/docs/api",
			cfg:       nil,
			want:      "https://example.com/docs/api",
		},
		{
			name:      "Query format - Japanese",
			baseURL:   "https://ai.google.dev",
			locale:    "ja",
			canonical: "/docs",
			cfg:       &LocaleConfig{ParamName: "hl"},
			want:      "https://ai.google.dev/docs?hl=ja",
		},
		{
			name:      "Query format - English",
			baseURL:   "https://ai.google.dev",
			locale:    "en",
			canonical: "/gemini-api/docs",
			cfg:       &LocaleConfig{ParamName: "hl"},
			want:      "https://ai.google.dev/gemini-api/docs?hl=en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildLocaleURL(tt.baseURL, tt.locale, tt.canonical, tt.cfg)
			if got != tt.want {
				t.Errorf("BuildLocaleURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractHreflang(t *testing.T) {
	htmlStr := `
<!DOCTYPE html>
<html>
<head>
	<link rel="alternate" hreflang="en" href="https://example.com/en/docs">
	<link rel="alternate" hreflang="ja" href="https://example.com/ja/docs">
	<link rel="alternate" hreflang="zh-TW" href="https://example.com/zh-tw/docs">
	<link rel="canonical" href="https://example.com/docs">
</head>
<body></body>
</html>
`
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	result := ExtractHreflang(doc)

	expected := map[string]string{
		"en":    "https://example.com/en/docs",
		"ja":    "https://example.com/ja/docs",
		"zh-tw": "https://example.com/zh-tw/docs",
	}

	if len(result) != len(expected) {
		t.Errorf("ExtractHreflang() returned %d entries, want %d", len(result), len(expected))
	}

	for locale, wantURL := range expected {
		if gotURL, ok := result[locale]; !ok {
			t.Errorf("ExtractHreflang() missing locale %q", locale)
		} else if gotURL != wantURL {
			t.Errorf("ExtractHreflang()[%q] = %q, want %q", locale, gotURL, wantURL)
		}
	}
}

func TestSelectPreferredLocaleURL(t *testing.T) {
	hreflangMap := map[string]string{
		"en":    "https://example.com/en/docs",
		"ja":    "https://example.com/ja/docs",
		"zh-tw": "https://example.com/zh-tw/docs",
	}

	tests := []struct {
		name       string
		priority   []string
		wantLocale string
		wantURL    string
	}{
		{
			name:       "English first",
			priority:   []string{"en", "ja"},
			wantLocale: "en",
			wantURL:    "https://example.com/en/docs",
		},
		{
			name:       "Japanese first",
			priority:   []string{"ja", "en"},
			wantLocale: "ja",
			wantURL:    "https://example.com/ja/docs",
		},
		{
			name:       "Prefer unavailable locale, fallback to available",
			priority:   []string{"de", "fr", "en"},
			wantLocale: "en",
			wantURL:    "https://example.com/en/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocale, gotURL := SelectPreferredLocaleURL(hreflangMap, tt.priority)
			if gotLocale != tt.wantLocale {
				t.Errorf("SelectPreferredLocaleURL() locale = %q, want %q", gotLocale, tt.wantLocale)
			}
			if gotURL != tt.wantURL {
				t.Errorf("SelectPreferredLocaleURL() url = %q, want %q", gotURL, tt.wantURL)
			}
		})
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ja", "ja"},
		{"JA", "ja"},
		{"ja-JP", "ja"},
		{"en-US", "en"},
		{"en-GB", "en"},
		{"zh-Hans", "zh-cn"},
		{"zh-Hant", "zh-tw"},
		{"de", "de"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeLocale(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeLocale(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
