// Package fetcher provides website crawling and downloading functionality.
// It recursively crawls a website following same-domain links up to a maximum depth,
// storing HTML files locally while respecting rate limits and skipping non-HTML resources.
package fetcher

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// Fetcher crawls and downloads website content.
type Fetcher struct {
	outputDir        string
	domain           string
	visited          map[string]bool
	visitedCanonical map[string]bool // canonical path の重複管理（ロケール優先モード用）
	mu               sync.Mutex
	maxDepth         int
	downloadCount    int
	startTime        time.Time
	client           *http.Client
	localeConfig     *LocaleConfig  // ロケール優先設定（nil で無効）
	robotsChecker    *RobotsChecker // robots.txt チェッカー
}

// UserAgent is the user agent string used by the fetcher.
const UserAgent = "site2skillgo/1.0 (+https://github.com/f4ah6o/site2skill-go)"

// New creates a new Fetcher instance configured to save downloads to outputDir.
func New(outputDir string) *Fetcher {
	return &Fetcher{
		outputDir:        outputDir,
		visited:          make(map[string]bool),
		visitedCanonical: make(map[string]bool),
		maxDepth:         5,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		robotsChecker: NewRobotsChecker(UserAgent),
	}
}

// SetLocaleConfig configures the fetcher to use locale priority-based content negotiation.
// If cfg is nil, locale priority mode is disabled and the fetcher uses standard crawling.
// When enabled, the fetcher will attempt to fetch pages in the preferred languages from LocaleConfig.Priority.
func (f *Fetcher) SetLocaleConfig(cfg *LocaleConfig) {
	f.localeConfig = cfg
}

// Fetch downloads the website starting at targetURL, recursively following
// same-domain links up to maxDepth. It validates the URL scheme and saves
// all HTML files to the output directory in a structure preserving the original paths.
// targetURL must be a valid http or https URL with a domain.
// If the URL scheme is omitted, https:// is automatically prepended.
func (f *Fetcher) Fetch(targetURL string) error {
	// Auto-prepend https:// if no scheme is provided
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
		log.Printf("No scheme provided, using: %s", targetURL)
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s. Only http and https are supported", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("invalid URL: domain is missing")
	}

	f.domain = parsedURL.Host
	crawlDir := filepath.Join(f.outputDir, "crawl")

	// Set base path for robots.txt lookup (for subdirectory deployments like GitHub Pages)
	// Extract the first path segment as base path (e.g., "/site2skill-go" from "/site2skill-go/docs/")
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) > 0 && pathParts[0] != "" {
			basePath := "/" + pathParts[0]
			f.robotsChecker.SetBasePath(basePath)
			log.Printf("Set robots.txt base path: %s", basePath)
		}
	}

	// Clean/Create crawl directory
	if err := os.RemoveAll(crawlDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove crawl dir: %w", err)
	}
	if err := os.MkdirAll(crawlDir, 0755); err != nil {
		return fmt.Errorf("failed to create crawl dir: %w", err)
	}

	log.Printf("Fetching %s to %s...", targetURL, crawlDir)
	log.Printf("Domain restricted to: %s", f.domain)

	f.startTime = time.Now()
	f.downloadCount = 0

	// Start crawling
	if err := f.crawl(targetURL, crawlDir, 0); err != nil {
		return err
	}

	elapsed := time.Since(f.startTime)
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	log.Printf("Download complete. %d pages in %dm%02ds.", f.downloadCount, mins, secs)

	return nil
}

// crawl recursively downloads a page and follows links up to the maximum depth.
// It respects the domain restriction and uses locale priority mode if configured.
// crawl logs progress and silently skips errors to continue crawling other pages.
func (f *Fetcher) crawl(targetURL, crawlDir string, depth int) error {
	if depth > f.maxDepth {
		return nil
	}

	// Check robots.txt
	if !f.robotsChecker.IsAllowed(targetURL) {
		log.Printf("Blocked by robots.txt: %s", targetURL)
		return nil
	}

	// ロケール優先モードの場合、canonical path ベースで重複チェック
	if f.localeConfig != nil {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			return nil
		}

		_, canonical := ExtractLocale(parsedURL, f.localeConfig)

		f.mu.Lock()
		if f.visitedCanonical[canonical] {
			f.mu.Unlock()
			return nil
		}
		f.visitedCanonical[canonical] = true
		f.mu.Unlock()

		// ロケール優先クロール
		return f.crawlWithLocalePriority(targetURL, canonical, crawlDir, depth)
	}

	// Check if already visited (従来モード)
	f.mu.Lock()
	if f.visited[targetURL] {
		f.mu.Unlock()
		return nil
	}
	f.visited[targetURL] = true
	f.mu.Unlock()

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil // Skip invalid URLs
	}

	// Only crawl same domain
	if parsedURL.Host != f.domain {
		return nil
	}

	// Skip non-HTML resources
	if isNonHTMLResource(targetURL) {
		return nil
	}

	// Fetch the page
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)

	// Be polite: wait 1 second between requests
	time.Sleep(1 * time.Second)

	resp, err := f.client.Do(req)
	if err != nil {
		log.Printf("Warning: failed to fetch %s: %v", targetURL, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Warning: %s returned status %d", targetURL, resp.StatusCode)
		return nil
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		return nil // Skip non-HTML content
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Warning: failed to read body from %s: %v", targetURL, err)
		return nil
	}

	// Save to file
	filePath := f.getFilePath(crawlDir, parsedURL)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Printf("Warning: failed to create directory for %s: %v", filePath, err)
		return nil
	}

	if err := os.WriteFile(filePath, body, 0644); err != nil {
		log.Printf("Warning: failed to write file %s: %v", filePath, err)
		return nil
	}

	f.downloadCount++
	elapsed := time.Since(f.startTime)
	rate := float64(f.downloadCount) / elapsed.Seconds()
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	shortURL := targetURL
	if len(shortURL) > 60 {
		shortURL = shortURL[len(shortURL)-60:]
	}
	fmt.Printf("\r[%d pages | %dm%02ds | %.1f/s] %s", f.downloadCount, mins, secs, rate, shortURL)

	// Parse HTML and extract links
	htmlString := decodeHTML(body, resp.Header.Get("Content-Type"))
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return nil // Skip if we can't parse
	}

	links := f.extractLinks(doc, targetURL)

	// Crawl links (with depth limit)
	for _, link := range links {
		f.crawl(link, crawlDir, depth+1)
	}

	return nil
}

// getFilePath constructs a file path for saving a downloaded page.
// It creates a structure like crawl/domain.com/path/to/page.html, using index.html for root paths.
// If query parameters are present, they are encoded into the filename to avoid collisions.
func (f *Fetcher) getFilePath(crawlDir string, parsedURL *url.URL) string {
	// Create path like: crawl/domain.com/path/to/page.html
	path := parsedURL.Path
	if path == "" || path == "/" {
		path = "/index"
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Handle query parameters
	query := parsedURL.Query()
	if len(query) > 0 {
		// Encode query params into filename: page_key_val.html
		// Or simpler: page_QUERYHASH.html or page.html?key=val -> page_...
		// Using a simplified deterministic encoding for readability where possible

		// Sort keys for determinism
		encodedQuery := query.Encode() // e.g., "hl=ja&key=val"
		// Replace characters invalid in filenames
		safeQuery := strings.ReplaceAll(encodedQuery, "&", "_")
		safeQuery = strings.ReplaceAll(safeQuery, "=", "_")
		safeQuery = strings.ReplaceAll(safeQuery, "%", "")

		path += "_q_" + safeQuery
	}

	// Add .html if no extension
	if filepath.Ext(path) == "" {
		path += ".html"
	}

	return filepath.Join(crawlDir, parsedURL.Host, path)
}

// extractLinks recursively extracts all absolute URLs from href attributes in an HTML node tree.
// It resolves relative URLs using the provided base URL.
func (f *Fetcher) extractLinks(n *html.Node, baseURL string) []string {
	var links []string

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := attr.Val
					// Resolve relative URLs
					absoluteURL, err := url.Parse(link)
					if err != nil {
						continue
					}
					base, err := url.Parse(baseURL)
					if err != nil {
						continue
					}
					resolvedURL := base.ResolveReference(absoluteURL)
					links = append(links, resolvedURL.String())
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(n)
	return links
}

// isNonHTMLResource checks if a URL points to a non-HTML resource based on file extension.
// It returns true for assets like CSS, JavaScript, images, archives, and other non-HTML content.
func isNonHTMLResource(urlStr string) bool {
	nonHTMLExtensions := []string{
		".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".eot", ".zip", ".tar", ".gz", ".pdf",
		".xml", ".json", ".txt",
	}

	lower := strings.ToLower(urlStr)
	for _, ext := range nonHTMLExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// decodeHTML decodes HTML bytes to string using detected charset.
// It tries to extract charset from Content-Type header or HTML meta tag.
func decodeHTML(body []byte, contentType string) string {
	// Try to get charset from Content-Type header
	enc := getEncodingFromContentType(contentType)
	if enc != nil {
		decoded, err := decodeWithEncoding(body, enc)
		if err == nil {
			return decoded
		}
	}

	// Try to get charset from HTML meta tag
	enc = getEncodingFromMeta(body)
	if enc != nil {
		decoded, err := decodeWithEncoding(body, enc)
		if err == nil {
			return decoded
		}
	}

	// Fallback to UTF-8
	return string(body)
}

// getEncodingFromContentType extracts charset from Content-Type header.
func getEncodingFromContentType(contentType string) encoding.Encoding {
	if contentType == "" {
		return nil
	}

	// Parse charset from Content-Type header
	re := regexp.MustCompile(`charset=([^\s;]+)`)
	matches := re.FindStringSubmatch(contentType)
	if len(matches) > 1 {
		charset := strings.Trim(matches[1], `"'`)
		enc, err := htmlindex.Get(charset)
		if err == nil {
			return enc
		}
	}

	return nil
}

// getEncodingFromMeta extracts charset from HTML meta tag using regex on raw bytes.
// This avoids parsing the HTML with incorrect encoding which would corrupt the content.
func getEncodingFromMeta(body []byte) encoding.Encoding {
	// Quick check for BOM first
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		// UTF-8 BOM
		return nil // UTF-8 is default
	}

	// Use regex to find charset in raw bytes (works for ASCII-compatible encodings)
	// Pattern 1: <meta charset="...">
	charsetRe := regexp.MustCompile(`(?i)<meta[^>]+charset=["']?([^"'\s>]+)`)
	if matches := charsetRe.Find(body); matches != nil {
		submatches := charsetRe.FindSubmatch(body)
		if len(submatches) > 1 {
			charset := string(submatches[1])
			if enc, err := htmlindex.Get(charset); err == nil {
				return enc
			}
		}
	}

	// Pattern 2: <meta http-equiv="Content-Type" content="text/html; charset=...">
	contentTypeRe := regexp.MustCompile(`(?i)<meta[^>]+http-equiv=["']?Content-Type["']?[^>]+content=["']?[^"']*charset=([^"'\s;>]+)`)
	if matches := contentTypeRe.Find(body); matches != nil {
		submatches := contentTypeRe.FindSubmatch(body)
		if len(submatches) > 1 {
			charset := string(submatches[1])
			if enc, err := htmlindex.Get(charset); err == nil {
				return enc
			}
		}
	}

	// Pattern 3: content attribute before http-equiv (reverse order)
	contentTypeRe2 := regexp.MustCompile(`(?i)<meta[^>]+content=["']?[^"']*charset=([^"'\s;>]+)[^>]+http-equiv=["']?Content-Type["']?`)
	if matches := contentTypeRe2.Find(body); matches != nil {
		submatches := contentTypeRe2.FindSubmatch(body)
		if len(submatches) > 1 {
			charset := string(submatches[1])
			if enc, err := htmlindex.Get(charset); err == nil {
				return enc
			}
		}
	}

	return nil
}

// decodeWithEncoding decodes bytes using specified encoding.
func decodeWithEncoding(body []byte, enc encoding.Encoding) (string, error) {
	reader := transform.NewReader(bytes.NewReader(body), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// crawlWithLocalePriority downloads a page using locale priority-based content negotiation.
// It attempts to fetch the page in languages specified by the LocaleConfig.Priority order,
// using HEAD requests to check availability before fetching the full content.
// It falls back to the original URL if no preferred locale version is found.
func (f *Fetcher) crawlWithLocalePriority(originalURL, canonical, crawlDir string, depth int) error {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return nil
	}

	// Only crawl same domain
	if parsedURL.Host != f.domain {
		return nil
	}

	// Skip non-HTML resources
	if isNonHTMLResource(originalURL) {
		return nil
	}

	// Check robots.txt
	if !f.robotsChecker.IsAllowed(originalURL) {
		log.Printf("Blocked by robots.txt: %s", originalURL)
		return nil
	}

	baseURL := parsedURL.Scheme + "://" + parsedURL.Host

	// 優先順位に従ってロケールを試行
	priority := f.localeConfig.Priority
	if len(priority) == 0 {
		priority = DefaultLocalePriority
	}

	var fetchURL string
	var foundLocale string

	// まず各ロケールでHEADリクエストを試行
	for _, locale := range priority {
		testURL := BuildLocaleURL(baseURL, locale, canonical, f.localeConfig)
		exists, statusCode := f.checkURLExists(testURL)

		if exists {
			fetchURL = testURL
			foundLocale = locale
			break
		}

		// 404以外のエラーは異常系として中断
		if statusCode != http.StatusNotFound && statusCode != 0 {
			if statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests || statusCode >= 500 {
				log.Printf("Warning: %s returned status %d, skipping canonical %s", testURL, statusCode, canonical)
				return nil
			}
		}
	}

	// 優先ロケールで見つからない場合、元のURLを試す
	if fetchURL == "" {
		exists, _ := f.checkURLExists(originalURL)
		if exists {
			fetchURL = originalURL
		} else {
			return nil
		}
	}

	// Be polite: wait 1 second between requests
	time.Sleep(1 * time.Second)

	// 本文を取得
	req, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		log.Printf("Warning: failed to fetch %s: %v", fetchURL, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Warning: %s returned status %d", fetchURL, resp.StatusCode)
		return nil
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		return nil
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Warning: failed to read body from %s: %v", fetchURL, err)
		return nil
	}

	// Save to file
	filePath := f.getFilePath(crawlDir, parsedURL)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Printf("Warning: failed to create directory for %s: %v", filePath, err)
		return nil
	}

	if err := os.WriteFile(filePath, body, 0644); err != nil {
		log.Printf("Warning: failed to write file %s: %v", filePath, err)
		return nil
	}

	f.downloadCount++
	elapsed := time.Since(f.startTime)
	rate := float64(f.downloadCount) / elapsed.Seconds()
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	shortURL := fetchURL
	if len(shortURL) > 60 {
		shortURL = shortURL[len(shortURL)-60:]
	}
	localeInfo := ""
	if foundLocale != "" {
		localeInfo = fmt.Sprintf(" [%s]", foundLocale)
	}
	fmt.Printf("\r[%d pages | %dm%02ds | %.1f/s]%s %s", f.downloadCount, mins, secs, rate, localeInfo, shortURL)

	// Parse HTML and extract links
	htmlString := decodeHTML(body, resp.Header.Get("Content-Type"))
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return nil
	}

	links := f.extractLinks(doc, fetchURL)

	// Crawl links (with depth limit)
	for _, link := range links {
		f.crawl(link, crawlDir, depth+1)
	}

	return nil
}

// checkURLExists checks if a URL is accessible using a HEAD request.
// It returns both a boolean indicating success and the HTTP status code.
// If HEAD fails, it falls back to a GET request with a Range header.
func (f *Fetcher) checkURLExists(targetURL string) (bool, int) {
	req, err := http.NewRequest("HEAD", targetURL, nil)
	if err != nil {
		return false, 0
	}
	req.Header.Set("User-Agent", UserAgent)

	// HEAD リクエストは短いタイムアウトで
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// HEAD が失敗した場合、GET + Range を試す
		return f.checkURLExistsWithRange(targetURL)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, resp.StatusCode
}

// checkURLExistsWithRange checks if a URL is accessible using a GET request with a Range header.
// This is a fallback for servers that don't support HEAD requests.
// It requests only the first byte to minimize bandwidth usage.
func (f *Fetcher) checkURLExistsWithRange(targetURL string) (bool, int) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return false, 0
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Range", "bytes=0-0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()

	// 206 Partial Content または 200 OK を成功とみなす
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent, resp.StatusCode
}
