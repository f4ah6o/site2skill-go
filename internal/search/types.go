package search

// SearchResult represents a single search result from a documentation file
type SearchResult struct {
	File      string   `json:"file"`
	Matches   int      `json:"matches"`
	Contexts  []string `json:"contexts"`
	SourceURL string   `json:"source_url"`
	FetchedAt string   `json:"fetched_at"`
}

// Frontmatter represents YAML frontmatter from a Markdown file
type Frontmatter struct {
	Title     string
	SourceURL string
	FetchedAt string
}

// SearchOptions contains configuration for search operations
type SearchOptions struct {
	SkillDir   string
	Query      string
	MaxResults int
	JSONOutput bool
}
