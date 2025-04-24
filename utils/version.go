package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

// ProjectType represents the type of project detected.
// Exported type.
type ProjectType int

const (
	// Exported constants.
	UnknownProject ProjectType = iota
	ElixirProject
	NodeProject
)

// DetectProject checks for known project files (mix.exs, package.json).
// Exported function.
func DetectProject() (ProjectType, string) {
	if _, err := os.Stat("mix.exs"); err == nil {
		return ElixirProject, "mix.exs"
	}
	if _, err := os.Stat("package.json"); err == nil {
		return NodeProject, "package.json"
	}
	return UnknownProject, ""
}

// CalculateNewVersion determines the new version string without modifying the file.
// Exported function.
func CalculateNewVersion(project ProjectType, filePath, bumpType string) (newVersionStr, currentVersionStr string, err error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read %s: %w", filePath, err)
	}
	contentStr := string(content)

	var versionPattern *regexp.Regexp

	switch project {
	case ElixirProject:
		versionPattern = regexp.MustCompile(`(?:version:|@version)\s*"([^"]+)"`)
	case NodeProject:
		versionPattern = regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
	default:
		return "", "", fmt.Errorf("unsupported project type for calculation")
	}

	matches := versionPattern.FindStringSubmatch(contentStr)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("could not find version string pattern in %s", filePath)
	}
	currentVersionStr = matches[1]

	if bumpType == "date" {
		newVersionStr = time.Now().UTC().Format("2006.1.2-150405")
	} else {
		currentVersion, err := semver.NewVersion(currentVersionStr)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse existing version '%s': %w", currentVersionStr, err)
		}
		var newVersion semver.Version
		switch bumpType {
		case "Major":
			newVersion = currentVersion.IncMajor()
		case "Minor":
			newVersion = currentVersion.IncMinor()
		case "Patch":
			newVersion = currentVersion.IncPatch()
		default:
			return "", "", fmt.Errorf("invalid bump type: %s", bumpType)
		}
		newVersionStr = newVersion.String()
	}

	return newVersionStr, currentVersionStr, nil
}

// WriteVersion updates the file content with the new version string.
// Exported function.
func WriteVersion(project ProjectType, filePath, currentVersionStr, newVersionStr string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s for writing: %w", filePath, err)
	}
	contentStr := string(content)

	var oldVersionLine, newVersionLine string

	switch project {
	case ElixirProject:
		if strings.Contains(contentStr, "@version") {
			oldVersionLine = fmt.Sprintf(`@version "%s"`, currentVersionStr)
			newVersionLine = fmt.Sprintf(`@version "%s"`, newVersionStr)
		} else {
			oldVersionLine = fmt.Sprintf(`version: "%s"`, currentVersionStr)
			newVersionLine = fmt.Sprintf(`version: "%s"`, newVersionStr)
		}
	case NodeProject:
		oldVersionLine = fmt.Sprintf(`"version": "%s"`, currentVersionStr)
		newVersionLine = fmt.Sprintf(`"version": "%s"`, newVersionStr)
	default:
		return fmt.Errorf("unsupported project type for writing")
	}

	newContent := strings.Replace(contentStr, oldVersionLine, newVersionLine, 1)
	if newContent == contentStr {
		return fmt.Errorf("failed to replace version string in %s; old content matched new content", filePath)
	}

	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated %s: %w", filePath, err)
	}
	return nil
}
