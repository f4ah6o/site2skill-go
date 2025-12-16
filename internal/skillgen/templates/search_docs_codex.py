#!/usr/bin/env python3
"""
search_docs.py - Documentation search tool for Codex Skills

Searches through Markdown documentation files and returns relevant results.
Optimized for Codex with clear, structured output.
"""

import os
import sys
import argparse
import re
import json
from pathlib import Path
from typing import List, Dict, Tuple

def extract_frontmatter(content: str) -> Tuple[Dict, str]:
    """Parse YAML frontmatter from Markdown content."""
    frontmatter = {}
    body = content

    match = re.match(r'^---\s*\n(.*?)\n---\s*\n(.*)', content, re.DOTALL)

    if match:
        frontmatter_str = match.group(1)
        body = match.group(2)

        for line in frontmatter_str.split('\n'):
            if ':' in line:
                key, value = line.split(':', 1)
                frontmatter[key.strip()] = value.strip().strip('"')

    return frontmatter, body

def get_context(text: str, query: str, context_lines: int = 3) -> List[str]:
    """Extract context around matching lines."""
    lines = text.split('\n')
    keywords = query.lower().split()
    contexts = []

    match_indices = [i for i, line in enumerate(lines)
                     if any(kw in line.lower() for kw in keywords)]

    if not match_indices:
        return []

    # Group nearby matches
    groups = []
    if match_indices:
        current_group = [match_indices[0]]
        for i in range(1, len(match_indices)):
            if match_indices[i] - match_indices[i-1] <= (context_lines * 2 + 1):
                current_group.append(match_indices[i])
            else:
                groups.append(current_group)
                current_group = [match_indices[i]]
        groups.append(current_group)

    # Extract context for each group
    for group in groups:
        start_idx = max(0, group[0] - context_lines)
        end_idx = min(len(lines), group[-1] + context_lines + 1)

        snippet = '\n'.join(lines[start_idx:end_idx])
        contexts.append(snippet)

    return contexts

def search_docs(docs_dir: Path, query: str, max_results: int = 10) -> List[Dict]:
    """Search documentation files for query."""
    if not docs_dir.exists():
        return []

    keywords = query.lower().split()
    results = []

    for file_path in docs_dir.glob("**/*.md"):
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()

            frontmatter, body = extract_frontmatter(content)
            body_lower = body.lower()

            # Count keyword matches
            matches_count = sum(body_lower.count(kw) for kw in keywords)

            if matches_count > 0:
                contexts = get_context(body, query)

                results.append({
                    "file": str(file_path.name),
                    "title": frontmatter.get("title", file_path.stem),
                    "matches": matches_count,
                    "contexts": contexts[:3],  # Top 3 contexts
                    "source_url": frontmatter.get("source_url", ""),
                    "fetched_at": frontmatter.get("fetched_at", "")
                })
        except Exception as e:
            print(f"Warning: Error reading {file_path}: {e}", file=sys.stderr)

    # Sort by match count
    results.sort(key=lambda x: x["matches"], reverse=True)
    return results[:max_results]

def print_results(results: List[Dict], query: str):
    """Print results in human-readable format."""
    if not results:
        print(f"No results found for query: '{query}'")
        return

    print(f"\n=== Search Results for: '{query}' ===\n")
    print(f"Found {len(results)} matching document(s)\n")

    for i, result in enumerate(results, 1):
        print(f"{i}. {result['title']}")
        print(f"   File: {result['file']}")
        print(f"   Matches: {result['matches']}")
        if result['source_url']:
            print(f"   URL: {result['source_url']}")
        print(f"   ---")

        # Show first context snippet
        if result['contexts']:
            print(f"   Preview:")
            preview = result['contexts'][0][:200]
            print(f"   {preview}...")
        print()

def main():
    parser = argparse.ArgumentParser(
        description="Search documentation files",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("query", help="Search query (keywords)")
    parser.add_argument("--max-results", "-n", type=int, default=10,
                       help="Maximum results to return (default: 10)")
    parser.add_argument("--json", action="store_true",
                       help="Output as JSON")
    parser.add_argument("--docs-dir", type=str, default=None,
                       help="Documentation directory (default: auto-detect)")

    args = parser.parse_args()

    # Auto-detect docs directory
    if args.docs_dir:
        docs_dir = Path(args.docs_dir)
    else:
        # Assume script is in skills/scripts/ directory
        script_dir = Path(__file__).resolve().parent
        docs_dir = script_dir.parent / "docs"

    results = search_docs(docs_dir, args.query, args.max_results)

    if args.json:
        print(json.dumps(results, indent=2))
    else:
        print_results(results, args.query)

if __name__ == "__main__":
    main()
