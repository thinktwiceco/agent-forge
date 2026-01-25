"""Version information for ThinkTwice SDK."""

import os
from pathlib import Path


def _get_version() -> str:
    """Read version from VERSION file."""
    # Get the directory where this file is located
    current_dir = Path(__file__).parent.parent
    version_file = current_dir / "VERSION"
    
    if version_file.exists():
        with open(version_file, "r", encoding="utf-8") as f:
            version = f.read().strip()
            return version
    
    # Fallback to environment variable or default
    return os.getenv("THINKTWICE_SDK_VERSION", "0.1.0")


__version__ = _get_version()

