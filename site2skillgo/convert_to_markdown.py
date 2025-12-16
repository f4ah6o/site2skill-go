import os
import re
import argparse
import logging
from typing import Optional
from bs4 import BeautifulSoup
from markdownify import MarkdownConverter

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def clean_html(soup: BeautifulSoup) -> BeautifulSoup:
    """Remove unwanted tags and noise from the HTML."""
    # Remove non-content tags
    for tag in soup(["script", "style", "meta", "link", "noscript", "iframe", "svg"]):
        tag.decompose()
    
    # Remove navigation, header, footer, sidebar if they exist
    # Common selectors for documentation sites
    selectors = [
        ".sidebar", "header", "footer", ".nav", ".menu", "#sidebar", 
        ".navigation", ".toc", "#toc", ".footer", "#footer"
    ]
    
    for selector in selectors:
        for tag in soup.select(selector):
            tag.decompose()
            
    return soup

def post_process_markdown(md_content: str) -> str:
    """Clean up the generated Markdown."""
    # Remove multiple consecutive blank lines
    md_content = re.sub(r'\n{3,}', '\n\n', md_content)
    
    # Remove trailing whitespace
    md_content = "\n".join([line.rstrip() for line in md_content.splitlines()])
    
    return md_content

def convert_html_to_md(html_path: str, output_path: Optional[str] = None, source_url: Optional[str] = None, fetched_at: Optional[str] = None) -> None:
    """Convert a single HTML file to Markdown with Frontmatter."""
    try:
        with open(html_path, 'r', encoding='utf-8') as f:
            html_content = f.read()
        
        soup = BeautifulSoup(html_content, 'html.parser')
        
        # Extract title
        title = "Untitled"
        if soup.title:
            title = soup.title.string.strip()
        elif soup.h1:
            title = soup.h1.get_text().strip()
            
        # Extract main content
        # Try to find the most relevant content container
        main_content = soup.find('main')
        if not main_content:
            main_content = soup.find('article')
        if not main_content:
            main_content = soup.find('div', class_='content')
        if not main_content:
            main_content = soup.body
            
        if not main_content:
            logger.warning(f"No main content found in {html_path}")
            return

        clean_html(main_content)
        
        # Convert to Markdown
        # heading_style="atx" uses # for headers
        md_body = MarkdownConverter(heading_style="atx").convert_soup(main_content)
        md_body = post_process_markdown(md_body)
        
        # Create Frontmatter
        # Escape quotes in title for YAML
        escaped_title = title.replace('"', '\\"')
        
        frontmatter = "---\n"
        frontmatter += f'title: "{escaped_title}"\n'
        if source_url:
            frontmatter += f'source_url: "{source_url}"\n'
        if fetched_at:
            frontmatter += f'fetched_at: "{fetched_at}"\n'
        frontmatter += "---\n\n"
        
        final_md = frontmatter + md_body
        
        if output_path:
            # Ensure output directory exists
            os.makedirs(os.path.dirname(output_path), exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                f.write(final_md)
            logger.info(f"Converted: {html_path} -> {output_path}")
        else:
            print(final_md)
            
    except Exception as e:
        logger.error(f"Error converting {html_path}: {e}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Convert HTML to Markdown with Metadata.")
    parser.add_argument("input_file", help="Path to input HTML file")
    parser.add_argument("--output", "-o", help="Path to output Markdown file")
    parser.add_argument("--url", help="Source URL of the page")
    parser.add_argument("--fetched-at", help="Timestamp of fetch (ISO8601)")
    
    args = parser.parse_args()
    
    convert_html_to_md(args.input_file, args.output, args.url, args.fetched_at)
