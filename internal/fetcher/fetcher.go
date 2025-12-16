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
	outputDir     string
	domain        string
	visited       map[string]bool
	mu            sync.Mutex
	maxDepth      int
	downloadCount int
	startTime     time.Time
	client        *http.Client
}

// New creates a new Fetcher instance configured to save downloads to outputDir.
func New(outputDir string) *Fetcher {
	return &Fetcher{
		outputDir: outputDir,
		visited:   make(map[string]bool),
		maxDepth:  5,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

func (f *Fetcher) crawl(targetURL, crawlDir string, depth int) error {
	if depth > f.maxDepth {
		return nil
	}

	// Check if already visited
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
	req.Header.Set("User-Agent", "s2s-go/1.0 (+https://github.com/f4ah6o/site2skill-go)")

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

func (f *Fetcher) getFilePath(crawlDir string, parsedURL *url.URL) string {
	// Create path like: crawl/domain.com/path/to/page.html
	path := parsedURL.Path
	if path == "" || path == "/" {
		path = "/index"
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Add .html if no extension
	if filepath.Ext(path) == "" {
		path += ".html"
	}

	return filepath.Join(crawlDir, parsedURL.Host, path)
}

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
