# site2skill-go

**Turn any documentation website into a Claude or Codex Agent Skill.**

`site2skillgo` is a tool that scrapes a documentation website, converts it to Markdown, and packages it as an Agent Skill (ZIP format) with proper entry points and search functionality.

Agent Skills are dynamically loaded knowledge modules that AI assistants use on demand. This tool now supports both:
- **Claude Agent Skills** - For Claude Code, Claude apps, and the API
- **Codex Skills** - For OpenAI Codex and compatible systems

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
site2skillgo generate https://f4ah6o.github.io/site2skill-go/ myskill

# Generate a Codex skill
site2skillgo generate --format codex https://f4ah6o.github.io/site2skill-go/ myskill

# Generate both Claude and Codex skills
site2skillgo generate --format both https://f4ah6o.github.io/site2skill-go/ myskill
```

### Commands

#### Generate Command

Generate a skill package from a documentation website:

```bash
site2skillgo generate <URL> <SKILL_NAME> [options]
```

**Options:**
- `--format string`
  - Output format: `claude`, `codex`, or `both` (default "claude")
- `--global`
  - Install to global skills directory
  - Claude: `~/.claude/skills`
  - Codex: `$CODEX_HOME/skills` (or `~/.codex/skills` if `$CODEX_HOME` is not set)
  - Default: local installation (`./.claude/skills` or `./.codex/skills`)
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
# Create a Claude skill from documentation
site2skillgo generate https://f4ah6o.github.io/site2skill-go/ site2skill

# Create a Codex skill
site2skillgo generate --format codex https://f4ah6o.github.io/site2skill-go/ site2skill

# Create both Claude and Codex skills
site2skillgo generate --format both https://f4ah6o.github.io/site2skill-go/ site2skill

# Install to global skills directory
site2skillgo generate --global --clean https://f4ah6o.github.io/site2skill-go/ site2skill

# Skip fetching (reuse downloaded files)
site2skillgo generate --skip-fetch https://f4ah6o.github.io/site2skill-go/ site2skill

# Search in skill documentation
site2skillgo search "authentication" --skill-dir .claude/skills/site2skill

# Search with JSON output (limited results)
site2skillgo search "api endpoint" --json --max-results 5 --skill-dir .claude/skills/site2skill
```

## Environment Variables

- **`CODEX_HOME`**: Specifies the Codex home directory (default: `~/.codex`)
  - When set, Codex skills will be installed to `$CODEX_HOME/skills` instead of `~/.codex/skills`
  - Config file location: `$CODEX_HOME/config.toml`

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

> **Note**: To use Codex skills, you need to enable the skills feature in `$CODEX_HOME/config.toml` (or `~/.codex/config.toml` if `$CODEX_HOME` is not set):
> ```toml
> [features]
> skills = true
> ```
> When generating with `--format codex` or `--format both`, a reminder will be displayed if this setting is missing.

### Both Format
- Generates both Claude and Codex skill packages from the same documentation source
- Uses Claude format as the default SKILL.md for the skill directory
- Useful for maintaining compatibility across multiple AI platforms
- Both .skill files are created with their respective formats

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

## Acknowledgments

This project is a Go rewrite and fork of [laiso/site2skill](https://github.com/laiso/site2skill). Special thanks to [@laiso](https://github.com/laiso) for creating the original tool and concept.

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
