package fetcher

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

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

func (f *Fetcher) Fetch(targetURL string) error {
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
	req.Header.Set("User-Agent", "site2skill/1.0 (+https://github.com/laiso/site2skill)")

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
	doc, err := html.Parse(strings.NewReader(string(body)))
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
