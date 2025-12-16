# site2skill-go

**Turn any documentation website into a Claude or Codex Agent Skill.**

`s2s-go` is a tool that scrapes a documentation website, converts it to Markdown, and packages it as an Agent Skill (ZIP format) with proper entry points and search functionality.

Agent Skills are dynamically loaded knowledge modules that AI assistants use on demand. This tool now supports both:
- **Claude Agent Skills** - For Claude Code, Claude apps, and the API
- **Codex Skills** - For OpenAI Codex and compatible systems

## Features

- 🚀 **Rewritten in Go** - Fast, single binary, no dependencies
- 🔄 **Dual Format Support** - Generate skills for both Claude and Codex
- 🌐 **Built-in Web Crawler** - No need for wget
- 📝 **Smart HTML to Markdown Conversion** - Clean, readable documentation
- 🔍 **Full-text Search** - Embedded search script in each skill
- ✅ **Validation** - Automatic size and structure checks

## Installation

### Using `go install` (Recommended)

```bash
go install github.com/f4ah6o/site2skill-go/cmd/s2s-go@latest
```

This will download and install the latest version globally. The binary will be placed in `$GOPATH/bin` (usually `~/go/bin`).

### From Source

```bash
# Clone the repository
git clone https://github.com/f4ah6o/site2skill-go
cd site2skill-go

# Build the binary
go build -o s2s-go ./cmd/s2s-go

# Optional: Install globally
go install ./cmd/s2s-go
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/f4ah6o/site2skill-go/releases).

## Usage

### Basic Usage

```bash
# Generate a Claude skill
s2s-go https://docs.example.com myskill

# Generate a Codex skill
s2s-go https://docs.example.com myskill --format codex
```

### Full Options

```bash
s2s-go <URL> <SKILL_NAME> [options]

Options:
  --url string
        URL of the documentation site (required)
  --name string
        Name of the skill (required)
  --format string
        Output format: claude or codex (default "claude")
  --output string
        Base output directory for skill structure (default ".claude/skills")
  --skill-output string
        Output directory for .skill file (default ".")
  --temp-dir string
        Temporary directory for processing (default "build")
  --skip-fetch
        Skip the download step (use existing files in temp dir)
  --clean
        Clean up temporary directory after completion
```

### Examples

```bash
# Create a Claude skill for PAY.JP documentation
s2s-go https://docs.pay.jp/v1/ payjp

# Create a Codex skill for Stripe API
s2s-go https://stripe.com/docs/api stripe --format codex

# Custom output directory
s2s-go https://docs.python.org/3/ python3 --output ./my-skills --clean

# Skip fetching (reuse downloaded files)
s2s-go https://docs.example.com example --skip-fetch
```

## How it works

1. **Fetch**: Downloads the documentation site recursively using built-in HTTP crawler
2. **Convert**: Converts HTML pages to Markdown using smart content extraction
3. **Normalize**: Cleans up links and formatting, converts relative URLs to absolute
4. **Validate**: Checks the skill structure and size limits (8MB for Claude)
5. **Package**: Generates SKILL.md and zips everything into a `.skill` file

## Output Structure

The tool generates a skill directory with the following structure:

```
<skill_name>/
├── SKILL.md           # Entry point with usage instructions
├── docs/              # Markdown documentation files
└── scripts/
    └── search_docs.py # Search tool for documentation
```

Additionally, a `<skill_name>.skill` file (ZIP archive) is created.

## Format Differences

### Claude Format
- Optimized for Claude Agent SDK
- YAML frontmatter in SKILL.md with name and description
- Colored terminal output in search script
- Context-aware search results

### Codex Format
- Optimized for OpenAI Codex
- Plain markdown structure
- JSON output support
- Simplified search interface

## Search Tool

Each generated skill includes a Python search script:

```bash
# Search documentation
python scripts/search_docs.py "query"

# JSON output with limited results
python scripts/search_docs.py "query" --json --max-results 5
```

## Python Version (Legacy)

The original Python version is available in the `python-legacy` branch. The Go version offers:
- ✅ Single binary distribution
- ✅ No external dependencies (no wget required)
- ✅ Faster execution
- ✅ Better cross-platform support
- ✅ Native Claude and Codex format support

## Development

```bash
# Run tests
go test ./...

# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o s2s-go-linux-amd64 ./cmd/s2s-go
GOOS=darwin GOARCH=amd64 go build -o s2s-go-darwin-amd64 ./cmd/s2s-go
GOOS=windows GOARCH=amd64 go build -o s2s-go-windows-amd64.exe ./cmd/s2s-go
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
