"""
The buildozer_binary extension that provides the host-compatible buildozer binary.
"""

load("//private:buildozer_binary.bzl", _buildozer_binary = "buildozer_binary")

visibility("public")

buildozer_binary = _buildozer_binary
