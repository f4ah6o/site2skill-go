# site2skill

**Turn any documentation website into a Claude or Codex Agent Skill.**

`site2skill` is a tool that scrapes a documentation website, converts it to Markdown, and packages it as an Agent Skill (ZIP format) with proper entry points and search functionality.

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

### From Source

```bash
# Clone the repository
git clone https://github.com/laiso/site2skill
cd site2skill

# Build the binary
go build -o site2skill ./cmd/site2skill

# Optional: Install globally
go install ./cmd/site2skill
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/laiso/site2skill/releases).

## Usage

### Basic Usage

```bash
# Generate a Claude skill
site2skill https://docs.example.com myskill

# Generate a Codex skill
site2skill https://docs.example.com myskill --format codex
```

### Full Options

```bash
site2skill <URL> <SKILL_NAME> [options]

Options:
  -url string
        URL of the documentation site (required)
  -name string
        Name of the skill (required)
  -format string
        Output format: claude or codex (default "claude")
  -output string
        Base output directory for skill structure (default ".claude/skills")
  -skill-output string
        Output directory for .skill file (default ".")
  -temp-dir string
        Temporary directory for processing (default "build")
  -skip-fetch
        Skip the download step (use existing files in temp dir)
  -clean
        Clean up temporary directory after completion
```

### Examples

```bash
# Create a Claude skill for PAY.JP documentation
site2skill https://docs.pay.jp/v1/ payjp

# Create a Codex skill for Stripe API
site2skill https://stripe.com/docs/api stripe --format codex

# Custom output directory
site2skill https://docs.python.org/3/ python3 --output ./my-skills --clean

# Skip fetching (reuse downloaded files)
site2skill https://docs.example.com example --skip-fetch
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
GOOS=linux GOARCH=amd64 go build -o site2skill-linux-amd64 ./cmd/site2skill
GOOS=darwin GOARCH=amd64 go build -o site2skill-darwin-amd64 ./cmd/site2skill
GOOS=windows GOARCH=amd64 go build -o site2skill-windows-amd64.exe ./cmd/site2skill
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
