# site2skill-go

**Turn any documentation website into a Claude or Codex Agent Skill.**

`site2skillgo` is a tool that scrapes a documentation website, converts it to Markdown, and packages it as an Agent Skill (ZIP format) with proper entry points and search functionality.

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
go install github.com/f4ah6o/site2skill-go/cmd/site2skillgo@latest
```

This will download and install the latest version globally. The binary will be placed in `$GOPATH/bin` (usually `~/go/bin`).

### From Source

```bash
# Clone the repository
git clone https://github.com/f4ah6o/site2skill-go
cd site2skill-go

# Build the binary
go build -o site2skillgo ./cmd/site2skillgo

# Optional: Install globally
go install ./cmd/site2skillgo
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/f4ah6o/site2skill-go/releases).

## Usage

### Basic Usage

```bash
# Generate a Claude skill
site2skillgo generate https://docs.example.com myskill

# Generate a Codex skill
site2skillgo generate https://docs.example.com myskill --format codex
```

### Commands

#### Generate Command

Generate a skill package from a documentation website:

```bash
site2skillgo generate <URL> <SKILL_NAME> [options]
```

**Options:**
- `--format string`
  - Output format: `claude` or `codex` (default "claude")
- `--output string`
  - Base output directory for skill structure (default ".claude/skills")
- `--skill-output string`
  - Output directory for .skill file (default ".")
- `--temp-dir string`
  - Temporary directory for processing (default "build")
- `--skip-fetch`
  - Skip the download step (use existing files in temp dir)
- `--clean`
  - Clean up temporary directory after completion

#### Search Command

Search through skill documentation:

```bash
site2skillgo search <QUERY> [options]
```

**Options:**
- `--skill-dir string`
  - Path to the skill directory (default ".")
- `--max-results int`
  - Maximum number of results to display (default 10)
- `--json`
  - Output results as JSON

### Examples

```bash
# Create a Claude skill for PAY.JP documentation
site2skillgo generate https://docs.pay.jp/v1/ payjp

# Create a Codex skill for Stripe API
site2skillgo generate https://stripe.com/docs/api stripe --format codex

# Custom output directory
site2skillgo generate https://docs.python.org/3/ python3 --output ./my-skills --clean

# Skip fetching (reuse downloaded files)
site2skillgo generate https://docs.example.com example --skip-fetch

# Search in skill documentation
site2skillgo search "authentication" --skill-dir .claude/skills/myskill

# Search with JSON output (limited results)
site2skillgo search "api endpoint" --json --max-results 5 --skill-dir .claude/skills/myskill
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
└── docs/              # Markdown documentation files
```

Additionally, a `<skill_name>.skill` file (ZIP archive) is created.

Use the built-in `site2skillgo search` command to search through documentation files.

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

## Search Functionality

The `site2skillgo search` command is automatically embedded in each generated skill and can also be used via the command line to search through skill documentation. See the [Search Command](#search-command) section above for usage details.

## Development

```bash
# Run tests
go test ./...

# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o site2skillgo-linux-amd64 ./cmd/site2skillgo
GOOS=darwin GOARCH=amd64 go build -o site2skillgo-darwin-amd64 ./cmd/site2skillgo
GOOS=windows GOARCH=amd64 go build -o site2skillgo-windows-amd64.exe ./cmd/site2skillgo
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
