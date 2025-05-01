# bump

a tiny utility to make it easier to cut a release branch and bump the version number in Elixir and JS repos.

## Installation

Pre-compiled binaries for Linux, macOS (Darwin), and Windows are available on the [GitHub Releases page](https://github.com/rossvz/bump/releases). Download the appropriate archive for your system, extract the `bump` binary, and place it somewhere in your PATH.

Alternatively, if you have Go installed, you can build from source:

```bash
go install github.com/rossvz/bump@latest
```

## Usage

Run `bump` in the project directory:
1. Detects Elixir vs JS based on mix.exs or package.json
2. Prompts for semver or date-based versioning scheme
3. Checks out a new release/<version> branch
4. Adjusts the version and creates a commit

```bash
bump
Current branch: main
Select SemVer bump type:

  Major
> Minor
  Patch

Press Ctrl+C or q to quit early.
Creating branch: release/0.2.0
Updating mix.exs from 0.1.0 to 0.2.0
Staging mix.exs
Committing: version bump 0.2.0
Successfully created branch 'release/0.2.0', committed version bump. You are now on branch 'release/0.2.0'.
```

