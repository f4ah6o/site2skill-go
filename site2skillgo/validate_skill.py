import os
import re
import sys
import argparse
import logging
from typing import List, Tuple

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


def check_skill_size(skill_dir: str) -> None:
    """
    Checks the total size of the skill directory (specifically docs/).
    Warns if it exceeds 8MB.
    Lists top 10 largest files.
    """
    docs_dir = os.path.join(skill_dir, "docs")
    if not os.path.exists(docs_dir):
        return

    total_size = 0
    file_sizes: List[Tuple[int, str]] = []

    for root, _, files in os.walk(docs_dir):
        for f in files:
            fp = os.path.join(root, f)
            try:
                size = os.path.getsize(fp)
                total_size += size
                file_sizes.append((size, fp))
            except OSError:
                pass

    # Sort by size descending
    file_sizes.sort(key=lambda x: x[0], reverse=True)

    total_size_mb = total_size / (1024 * 1024)
    logger.info("\n--- Skill Size Analysis ---")
    logger.info(f"Total Uncompressed Size: {total_size_mb:.2f} MB")

    if total_size > 8 * 1024 * 1024:
        logger.warning("Skill uncompressed size exceeds Claude's 8MB limit.")
        logger.warning("The skill may fail to load in Claude.")
    else:
        logger.info("Size is within limits (< 8MB).")

    logger.info("\nTop 10 Largest Files:")
    for size, fp in file_sizes[:10]:
        rel_path = os.path.relpath(fp, skill_dir)
        logger.info(f"  {size / 1024:.1f} KB - {rel_path}")
    logger.info("---------------------------\n")


def validate_skill(skill_dir: str) -> bool:
    """
    Validates a skill directory structure and metadata.
    Checks for SKILL.md + docs/ structure.
    """
    logger.info(f"Validating skill in: {skill_dir}")

    errors = []
    warnings = []

    # 1. Check directory existence
    if not os.path.isdir(skill_dir):
        logger.error(f"Directory not found: {skill_dir}")
        return False

    # 2. Check SKILL.md
    skill_md_path = os.path.join(skill_dir, "SKILL.md")
    if not os.path.exists(skill_md_path):
        errors.append("SKILL.md not found.")
    else:
        logger.info("Found SKILL.md")
        # Validate frontmatter
        try:
            with open(skill_md_path, 'r', encoding='utf-8') as f:
                content = f.read()
                # Check for YAML frontmatter
                if content.startswith('---\n'):
                    frontmatter_match = re.match(r'^---\n(.*?)\n---', content, re.DOTALL)
                    if frontmatter_match:
                        frontmatter = frontmatter_match.group(1)
                        required_fields = ['name', 'description']
                        for field in required_fields:
                            if f'{field}:' not in frontmatter:
                                warnings.append(f"SKILL.md frontmatter missing '{field}' field")
                        logger.info("  YAML frontmatter present")
                    else:
                        warnings.append("SKILL.md has incomplete frontmatter")
                else:
                    warnings.append("SKILL.md missing YAML frontmatter")
        except Exception as e:
            warnings.append(f"Could not validate SKILL.md: {e}")

    # 3. Check docs/ directory
    docs_dir = os.path.join(skill_dir, "docs")
    if not os.path.isdir(docs_dir):
        errors.append("docs/ directory not found.")
    else:
        logger.info("Found docs/")

        # Count markdown files
        md_files = []
        for root, _, files in os.walk(docs_dir):
            for file in files:
                if file.endswith('.md'):
                    md_files.append(os.path.join(root, file))

        if len(md_files) == 0:
            warnings.append("docs/ directory is empty (no .md files)")
        else:
            logger.info(f"  {len(md_files)} markdown files")

    # 4. Check optional directories
    scripts_dir = os.path.join(skill_dir, "scripts")
    if os.path.isdir(scripts_dir):
        logger.info("Found scripts/ (optional)")

    # 5. Check skill size
    check_skill_size(skill_dir)

    # 6. Report results
    if errors:
        logger.error("VALIDATION FAILED:")
        for error in errors:
            logger.error(f"  - {error}")
        return False

    if warnings:
        logger.warning("Warnings:")
        for warning in warnings:
            logger.warning(f"  - {warning}")

    logger.info("Validation passed!")
    return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Validate a Skill directory.")
    parser.add_argument("skill_dir", help="Path to the skill directory")
    args = parser.parse_args()

    if not validate_skill(args.skill_dir):
        sys.exit(1)
