// Package fetcher provides website crawling and downloading functionality.
package fetcher

import (
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// LocaleConfig はロケール優先設定
type LocaleConfig struct {
	Priority  []string // 優先順位 ["en","ja"]
	ParamName string   // クエリパラメータ名（例: "hl"）。空ならパス形式を自動検出
}

// DefaultLocalePriority はデフォルトのロケール優先順位
var DefaultLocalePriority = []string{"en", "ja"}

// KnownLocales は既知のロケールコード（パス形式での検出用）
var KnownLocales = map[string]bool{
	"en": true, "en-us": true, "en-gb": true,
	"ja": true, "ja-jp": true,
	"zh": true, "zh-cn": true, "zh-tw": true, "zh-hk": true,
	"ko": true, "ko-kr": true,
	"de": true, "de-de": true,
	"fr": true, "fr-fr": true,
	"es": true, "es-es": true,
	"it": true, "it-it": true,
	"pt": true, "pt-br": true,
	"ru": true, "ru-ru": true,
	"ar": true, "nl": true, "pl": true, "tr": true,
	"vi": true, "th": true, "id": true, "ms": true,
}

// localePathPattern はパス形式のロケールを検出する正規表現
var localePathPattern = regexp.MustCompile(`^/([a-z]{2}(?:-[a-zA-Z]{2,4})?)/`)

// ExtractLocale はURLからロケールとcanonical pathを抽出する
// パス形式: /ja/docs/xxx → locale="ja", canonical="/docs/xxx"
// クエリ形式: /docs?hl=ja → locale="ja", canonical="/docs"
func ExtractLocale(u *url.URL, cfg *LocaleConfig) (locale, canonical string) {
	if u == nil {
		return "", ""
	}

	// クエリパラメータ形式
	if cfg != nil && cfg.ParamName != "" {
		locale = u.Query().Get(cfg.ParamName)
		// canonical はクエリパラメータを除いたパス
		canonical = u.Path
		return locale, canonical
	}

	// パス形式の自動検出
	path := u.Path
	matches := localePathPattern.FindStringSubmatch(path)
	if len(matches) >= 2 {
		potentialLocale := strings.ToLower(matches[1])
		if KnownLocales[potentialLocale] {
			locale = potentialLocale
			canonical = strings.TrimPrefix(path, "/"+matches[1])
			if canonical == "" {
				canonical = "/"
			}
			return locale, canonical
		}
	}

	// ロケールが見つからない場合はパス全体がcanonical
	return "", path
}

// BuildLocaleURL は canonical path と locale から URL を構築する
func BuildLocaleURL(baseURL, locale, canonical string, cfg *LocaleConfig) string {
	if locale == "" {
		return baseURL + canonical
	}

	// クエリパラメータ形式
	if cfg != nil && cfg.ParamName != "" {
		u, err := url.Parse(baseURL + canonical)
		if err != nil {
			return baseURL + canonical
		}
		q := u.Query()
		q.Set(cfg.ParamName, locale)
		u.RawQuery = q.Encode()
		return u.String()
	}

	// パス形式
	return baseURL + "/" + locale + canonical
}

// ExtractHreflang は HTML から hreflang リンクを抽出する
// 戻り値: map[locale]url (例: {"en": "https://...", "ja": "https://..."})
func ExtractHreflang(doc *html.Node) map[string]string {
	result := make(map[string]string)
	if doc == nil {
		return result
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, hreflang, href string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "hreflang":
					hreflang = attr.Val
				case "href":
					href = attr.Val
				}
			}
			if rel == "alternate" && hreflang != "" && href != "" {
				// ロケールを正規化（小文字、ja-JP -> ja-jp）
				normalizedLocale := strings.ToLower(hreflang)
				result[normalizedLocale] = href
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return result
}

// SelectPreferredLocaleURL は hreflang マップから優先ロケールのURLを選択する
func SelectPreferredLocaleURL(hreflangMap map[string]string, priority []string) (locale, url string) {
	if len(hreflangMap) == 0 {
		return "", ""
	}

	// 優先順位に従って選択
	for _, loc := range priority {
		if u, ok := hreflangMap[loc]; ok {
			return loc, u
		}
		// ja -> ja-jp のような変換も試す
		if u, ok := hreflangMap[loc+"-"+loc]; ok {
			return loc + "-" + loc, u
		}
	}

	// 優先順位に一致するものがなければ最初のものを返す
	for loc, u := range hreflangMap {
		return loc, u
	}

	return "", ""
}

// NormalizeLocale はロケールコードを正規化する（小文字化、エイリアス解決）
func NormalizeLocale(locale string) string {
	locale = strings.ToLower(locale)
	// 主要なエイリアス
	switch locale {
	case "ja-jp":
		return "ja"
	case "en-us", "en-gb":
		return "en"
	case "zh-hans", "zh-cn":
		return "zh-cn"
	case "zh-hant", "zh-tw":
		return "zh-tw"
	}
	return locale
}
