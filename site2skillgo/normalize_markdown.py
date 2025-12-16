import re
import argparse
import os
import yaml
from urllib.parse import urljoin

def extract_frontmatter(content):
    """Extract YAML frontmatter from markdown content."""
    match = re.match(r'^---\n(.*?)\n---\n', content, re.DOTALL)
    if match:
        try:
            return yaml.safe_load(match.group(1))
        except yaml.YAMLError:
            return None
    return None

def normalize_links(md_content, source_url=None):
    """
    Convert relative links to absolute URLs based on source_url.
    Matches: [text](path)
    """
    if not source_url:
        return md_content

    # Regex to capture links
    # \[([^\]]*)\]\(([^)]+)\)
    pattern = re.compile(r'\[([^\]]*)\]\(([^)]+)\)')
    
    def callback(match):
        text = match.group(1)
        url = match.group(2)
        
        # Skip if already absolute
        if url.startswith("http:") or url.startswith("https:") or url.startswith("mailto:"):
            return match.group(0)
            
        # Skip anchors only
        if url.startswith("#"):
            return match.group(0)
            
        # Resolve absolute URL
        # urljoin handles relative paths correctly
        absolute_url = urljoin(source_url, url)
        
        return f"[{text}]({absolute_url})"

    return pattern.sub(callback, md_content)

def normalize_markdown(input_path, output_path=None):
    try:
        with open(input_path, 'r', encoding='utf-8') as f:
            content = f.read()
            
        # Extract source_url from frontmatter
        frontmatter = extract_frontmatter(content)
        source_url = frontmatter.get('source_url') if frontmatter else None
        
        if source_url:
            normalized = normalize_links(content, source_url)
        else:
            print(f"Warning: No source_url found in {input_path}, skipping link normalization.")
            normalized = content
        
        if output_path:
            with open(output_path, 'w', encoding='utf-8') as f:
                f.write(normalized)
            print(f"Normalized: {input_path}")
        else:
            print(normalized)
            
    except Exception as e:
        print(f"Error normalizing {input_path}: {e}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Normalize Markdown links to absolute URLs.")
    parser.add_argument("input_file", help="Path to input Markdown file")
    parser.add_argument("--output", "-o", help="Path to output Markdown file")
    
    args = parser.parse_args()
    
    normalize_markdown(args.input_file, args.output)
