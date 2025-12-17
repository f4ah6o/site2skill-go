# Test Coverage Analysis - site2skill-go

**Generated:** 2025-12-17
**Project:** site2skill-go (Go 1.24.7)
**Total Test Files:** 9
**Total Test Lines:** ~757 lines

---

## Executive Summary

The codebase has **uneven test coverage** with strong focus on complex business logic (robots.txt parsing, locale detection) but significant gaps in file I/O operations, error handling, and integration testing. The main CLI entry point has no tests at all.

**Coverage Tiers:**
- ✅ **Strong:** Fetcher locale handling (290 lines), Robots.txt parsing (189 lines)
- ⚠️ **Moderate:** Basic initialization tests across packages
- ❌ **Weak:** File operations (packager, converter), Search functionality, Validation logic
- ❌ **Missing:** CLI entry point, integration tests, error scenarios

---

## Detailed Analysis by Package

### 1. **internal/converter** (38 test lines / 232 source lines = 16% coverage)

#### Current Tests
- ✅ `TestNewConverter` - Constructor nil check
- ✅ `TestPostProcessMarkdown` - Two scenarios (simple, empty)

#### What's NOT Tested
- ❌ `ConvertFile()` - Main conversion pipeline (reads HTML, converts to MD, writes file)
  - No tests for HTML file reading/writing
  - No frontmatter generation validation
  - No error handling (missing files, invalid HTML)
- ❌ `cleanHTML()` - Element removal logic
  - No verification that unwanted elements are removed
  - No edge cases (nested elements, missing selectors)
- ❌ Character encoding detection (`decodeHTML()`, `getEncodingFromMeta()`)
  - No tests for charset detection (UTF-8, Shift-JIS, etc.)
  - No BOM handling validation
  - No regex pattern testing
- ❌ Title extraction
  - No tests for extracting from `<title>`, `<h1>`, or fallback to "Untitled"
- ❌ HTML-to-Markdown conversion quality
- ❌ Error scenarios (invalid HTML, missing main content)

#### Recommendations
1. Add integration tests for `ConvertFile()` with sample HTML files
2. Create test fixtures for different HTML structures (with main, article, div.content, body)
3. Add charset detection tests with multiple encodings
4. Test frontmatter YAML generation
5. Test error handling (missing files, read permissions)

---

### 2. **internal/packager** (12 test lines / 109 source lines = 11% coverage)

#### Current Tests
- ✅ `TestNewPackager` - Constructor nil check only

#### What's NOT Tested
- ❌ `Package()` - Core packaging logic
  - No ZIP file creation verification
  - No directory traversal validation
  - No file inclusion verification
  - No path separator handling
- ❌ Error conditions
  - Directory not found
  - Permission denied
  - Invalid output directory
- ❌ Cross-platform path handling (Windows vs Unix)
- ❌ File permissions preservation in ZIP
- ❌ Empty directories handling
- ❌ Large file handling

#### Recommendations
1. Add integration tests with temporary directories and files
2. Create test fixtures with various directory structures
3. Test ZIP structure by reading back the created file
4. Test error cases (missing directories, permission errors)
5. Verify file paths use forward slashes in ZIP (cross-platform compatibility)

---

### 3. **internal/search** (29 test lines / 330 source lines = 9% coverage)

#### Current Tests
- ✅ `TestSearchDocs` - Single empty search test (just error handling)

#### What's NOT Tested
- ❌ `SearchDocs()` - Main search functionality
  - No keyword matching tests
  - No multi-keyword OR logic verification
  - No context extraction around matches
- ❌ File discovery and parsing
- ❌ Frontmatter parsing (title extraction)
- ❌ Result formatting (JSON vs human-readable)
- ❌ Multiple result handling
- ❌ Edge cases
  - Empty skill directory
  - No matching results
  - Malformed markdown files
  - Non-ASCII characters in content

#### Recommendations
1. Create test fixtures with sample skill markdown files
2. Test various search scenarios (single keyword, multiple keywords, no matches)
3. Test JSON output format
4. Test context window extraction
5. Test error handling for malformed files
6. Benchmark search performance with large skill packages

---

### 4. **internal/normalizer** (46 test lines / 133 source lines = 35% coverage)

#### Current Tests
- ✅ `TestNewNormalizer` - Constructor nil check
- ✅ `TestNormalizeLinks` - Three basic scenarios with weak assertions

#### What's NOT Tested
- ❌ `parseFrontmatter()` - YAML parsing
  - No validation that frontmatter is correctly extracted
  - No YAML structure verification
- ❌ `normalizeLinks()` - Actual link resolution
  - Test only checks if output is non-empty
  - No verification of actual link transformation
  - No testing of relative-to-absolute path conversion
- ❌ Edge cases
  - Links with anchors (#section)
  - Absolute URLs in markdown
  - Protocol-relative URLs (//example.com)
  - Broken/malformed URLs
  - Links with query parameters
- ❌ File reading/writing (`NormalizeFile()` if it exists)

#### Recommendations
1. Add stronger assertions in `TestNormalizeLinks` - verify actual transformations
2. Add dedicated `TestParseFrontmatter` tests with various YAML formats
3. Test edge cases (anchors, protocols, query params)
4. Test frontmatter reconstruction in output
5. Test file I/O operations

---

### 5. **internal/validator** (34 test lines / 175 source lines = 19% coverage)

#### Current Tests
- ✅ `TestNewValidator` - Constructor nil check
- ✅ `TestValidateSkill` - Single test on current directory

#### What's NOT Tested
- ❌ `Validate()` - Core validation logic
  - No tests for required files (SKILL.md)
  - No directory structure validation (docs/)
  - Only tests with current directory, not actual skill structures
- ❌ `validateFiles()` - File validation
  - No markdown validation
  - No YAML frontmatter validation
- ❌ Size checks
  - Skill package size limits
  - Individual file size limits
- ❌ Error conditions
  - Missing SKILL.md
  - Missing docs/ directory
  - Invalid markdown
  - Invalid YAML frontmatter
- ❌ Invalid skill structures
  - Extra files/directories
  - Wrong file names

#### Recommendations
1. Create test fixtures with valid and invalid skill structures
2. Add tests for missing required files/directories
3. Test size validation logic
4. Test markdown and YAML validation
5. Test comprehensive validation report generation
6. Test error messages and logging

---

### 6. **internal/skillgen** (34 test lines / 240 source lines = 14% coverage)

#### Current Tests
- ✅ `TestNewGenerator` - Format initialization (claude, codex, both)

#### What's NOT Tested
- ❌ `Generate()` - Core generation logic
  - No directory structure creation verification
  - No SKILL.md file generation
  - No content verification
- ❌ Format-specific behavior
  - Claude format SKILL.md structure
  - Codex format SKILL.md structure
  - Both format (both files created)
- ❌ `copySearchScript()` - Script file handling
- ❌ File operations
  - Directory creation
  - File writing
  - Existing file handling (overwrite vs skip)
- ❌ Error cases
  - Permission denied
  - Disk full
  - Invalid paths

#### Recommendations
1. Add integration tests with temporary directories
2. Test directory structure creation for each format
3. Verify SKILL.md content structure
4. Test script file copying
5. Test overwrite behavior
6. Test error handling
7. Mock file system for failure scenarios

---

### 7. **internal/fetcher** (564 test lines / 648 source lines = 87% coverage)

#### Current Tests
✅ **Well-covered areas:**
- ✅ Locale detection (PathFormat, QueryFormat) - 7+ scenarios each
- ✅ Locale URL building - 4 scenarios
- ✅ Hreflang extraction
- ✅ Locale preference selection - 3 scenarios
- ✅ Locale normalization - 8 scenarios
- ✅ Robots.txt parsing - 30+ test cases (exact, prefix, wildcard, root matches)
- ✅ File path generation from URLs - 8 scenarios

#### What's NOT Tested
- ❌ `Fetch()` - Main crawling logic
  - No actual HTTP requests tested
  - No recursive depth handling
  - No visited URL tracking
  - No domain boundary enforcement
- ❌ Network error handling
  - HTTP errors (404, 500)
  - Timeouts
  - Connection refused
- ❌ URL canonicalization
  - Query parameter ordering
  - Fragment handling
- ❌ Character encoding in crawled content
- ❌ Rate limiting enforcement
- ❌ robots.txt fetching from remote servers

#### Recommendations
1. Add mock HTTP client tests for `Fetch()` method
2. Test recursive crawling depth limiting
3. Test domain boundary enforcement
4. Test HTTP error handling
5. Add concurrency tests
6. Test rate limiting

---

### 8. **cmd/site2skillgo** (0 test lines / 615 source lines = 0% coverage)

#### What's NOT Tested
- ❌ `main()` - Entry point
  - No command parsing tests
  - No subcommand routing
- ❌ `runGenerate()` - Generate command pipeline
  - No full pipeline tests
  - No flag parsing
  - No output verification
- ❌ `runSearch()` - Search command
- ❌ `getCodexHome()` - Environment and home directory handling
- ❌ `checkCodexSkillsConfig()` - Config file parsing
- ❌ Flag parsing
  - `--locale-priority`
  - `--no-locale-priority`
  - `--format`
  - `--output`
  - `--skill-dir`
- ❌ Error scenarios
  - Invalid URLs
  - Missing flags
  - Network errors
  - File system errors

#### Recommendations
1. **HIGH PRIORITY** - Add CLI integration tests
2. Create test commands with mock URLs
3. Test flag parsing with various combinations
4. Test help text output
5. Test subcommand routing
6. Mock external systems (HTTP, file I/O)

---

## Cross-Package Issues

### Error Handling Coverage
- ❌ Most tests don't verify error cases
- ❌ No tests for error messages
- ❌ No error recovery testing

### File I/O Testing
- ❌ Minimal use of temporary directories in tests
- ❌ No permission/access error testing
- ❌ No disk space simulation
- ❌ No cleanup verification

### Concurrency
- ❌ Fetcher uses goroutines - no concurrent access tests
- ❌ No race condition testing
- ❌ No deadlock scenarios

### Integration Testing
- ❌ Only GitHub Actions E2E tests exist
- ❌ No end-to-end tests in the test suite
- ❌ No pipeline integration verification

### Test Quality
- ❌ Many tests only check for nil or non-empty results
- ❌ Weak assertions that don't verify correctness
- ❌ No output validation
- ❌ No state verification

---

## Recommended Improvements (Prioritized)

### Tier 1: Critical (Start with these)

1. **CLI Integration Tests** (`cmd/site2skillgo/main_test.go`)
   - Command routing tests
   - Flag parsing validation
   - End-to-end pipeline verification
   - **Impact:** Ensures main tool works correctly
   - **Effort:** 150-200 lines

2. **Converter File I/O Tests** (`internal/converter/converter_test.go`)
   - Add `TestConvertFile` with sample HTML
   - Character encoding tests
   - Frontmatter validation
   - **Impact:** Critical content conversion quality
   - **Effort:** 200-300 lines

3. **Packager ZIP Creation Tests** (`internal/packager/packager_test.go`)
   - Add `TestPackage` with directory fixtures
   - Verify ZIP contents
   - Error handling
   - **Impact:** Ensures .skill files are valid
   - **Effort:** 150-200 lines

### Tier 2: Important (High value-to-effort ratio)

4. **Validator Logic Tests** (`internal/validator/validator_test.go`)
   - Add comprehensive `TestValidate` with fixtures
   - Test each validation rule
   - **Impact:** Quality checks before packaging
   - **Effort:** 150-200 lines

5. **Normalizer Link Resolution** (`internal/normalizer/normalizer_test.go`)
   - Improve `TestNormalizeLinks` assertions
   - Add `TestParseFrontmatter`
   - Test link edge cases
   - **Impact:** Correct link transformation
   - **Effort:** 150-200 lines

6. **Search Functionality** (`internal/search/search_test.go`)
   - Add `TestSearch` with keyword matching
   - Test result formatting
   - **Impact:** Usable search feature
   - **Effort:** 100-150 lines

### Tier 3: Good to Have (Polish)

7. **Skillgen Generation** (`internal/skillgen/skillgen_test.go`)
   - Test directory creation
   - Format-specific manifests
   - **Impact:** Skill structure correctness
   - **Effort:** 150-200 lines

8. **Fetcher Network Tests** (`internal/fetcher/fetcher_test.go`)
   - Mock HTTP client for `Fetch()`
   - Recursive crawling tests
   - Error handling
   - **Impact:** Reliable web crawling
   - **Effort:** 200-300 lines

---

## Testing Best Practices to Adopt

1. **Use Table-Driven Tests** (already doing well in fetcher)
   - Continue this pattern across all tests

2. **Create Test Fixtures**
   ```go
   // Use testdata/ directory structure:
   testdata/
   ├── html/
   │   ├── simple.html
   │   ├── complex.html
   │   └── utf8.html
   ├── markdown/
   │   └── sample.md
   └── skills/
       └── valid-skill/
   ```

3. **Use Subtests** (already doing)
   - Keep using `t.Run()` for grouped assertions

4. **Test Fixtures and Helpers**
   - Create `*_test_helpers.go` files for common test setup
   - Use `testing.TB` for helper functions

5. **Mock External Dependencies**
   - HTTP client mocking for network tests
   - File system interfaces for I/O testing

6. **Coverage Goals by Category**
   - Business logic: 80%+
   - I/O operations: 70%+
   - Error handling: 80%+
   - CLI: 60%+ (harder to test)

---

## Summary Statistics

| Package | Source | Tests | Coverage | Priority |
|---------|--------|-------|----------|----------|
| converter | 232 | 38 | 16% | 🔴 Tier 1 |
| fetcher | 648 | 564 | 87% | ✅ Good |
| normalizer | 133 | 46 | 35% | 🟡 Tier 2 |
| packager | 109 | 12 | 11% | 🔴 Tier 1 |
| search | 330 | 29 | 9% | 🟡 Tier 2 |
| skillgen | 240 | 34 | 14% | 🟡 Tier 3 |
| validator | 175 | 34 | 19% | 🟡 Tier 2 |
| cmd/main | 615 | 0 | 0% | 🔴 Tier 1 |
| **TOTAL** | **2,482** | **757** | **30%** | — |

---

## Next Steps

1. **Week 1:** Implement Tier 1 tests (CLI, Converter, Packager)
2. **Week 2:** Implement Tier 2 tests (Validator, Normalizer, Search)
3. **Week 3:** Implement Tier 3 tests (Skillgen, Fetcher network)
4. **Ongoing:** Maintain test coverage as features are added
