import shutil
import os
import argparse
import sys

def package_skill(skill_dir, output_dir=None):
    """
    Packages a skill directory into a .skill file (zip).
    """
    if not os.path.isdir(skill_dir):
        print(f"Error: Directory not found: {skill_dir}")
        return False

    skill_name = os.path.basename(os.path.normpath(skill_dir))
    if output_dir is None:
        output_dir = os.path.dirname(os.path.normpath(skill_dir))
    
    output_filename = os.path.join(output_dir, f"{skill_name}") # shutil.make_archive adds extension
    
    print(f"Packaging {skill_dir} to {output_filename}.zip...")
    
    try:
        # Create zip
        archive_path = shutil.make_archive(output_filename, 'zip', skill_dir)
        
        # Rename .zip to .skill
        final_path = output_filename + ".skill"
        if os.path.exists(final_path):
            os.remove(final_path)
        
        os.rename(archive_path, final_path)
        print(f"Successfully created: {final_path}")
        return final_path
    except Exception as e:
        print(f"Error packaging skill: {e}")
        return None

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Package a Skill directory into a .skill file.")
    parser.add_argument("skill_dir", help="Path to the skill directory")
    parser.add_argument("--output", "-o", help="Output directory", default=".")
    args = parser.parse_args()

    if not package_skill(args.skill_dir, args.output):
        sys.exit(1)
