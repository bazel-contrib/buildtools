#!/usr/bin/env python3

from __future__ import print_function, unicode_literals
from subprocess import Popen, PIPE
from sys import exit


def run(*cmd):
    process = Popen(cmd, stdout=PIPE)
    output, err = process.communicate()
    if process.wait():
        exit(1)
    return output.strip().decode()


def main():
    tags = run("git", "describe", "--tags")
    version = tags.lstrip('v')
    print("STABLE_buildVersion", version)

    # rules_nodejs expects to read from volatile-status.txt
    print("BUILD_SCM_VERSION", version)

    revision = run("git", "rev-parse", "HEAD")
    print("STABLE_buildScmRevision", revision)


if __name__ == "__main__":
    main()
