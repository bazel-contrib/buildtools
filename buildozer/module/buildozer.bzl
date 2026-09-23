"""
Provides the buildozer label for use in repository rules and module extensions.
"""

visibility("public")

# The ".exe" suffix is present on all platforms to provide a stable label that
# also works on Windows.
BUILDOZER_LABEL = Label("@buildozer_binary//:buildozer.exe")
