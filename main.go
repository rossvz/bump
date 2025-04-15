package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rossvz/bump/utils"
)

type viewState int

const (
	selectSchemeView viewState = iota
	selectSemverBumpView
)

type model struct {
	state   viewState
	cursor  int
	choices []string

	scheme      string
	bump        string
	project     utils.ProjectType
	versionFile string
}

func initialModel() model {
	return model{
		state:   selectSchemeView,
		choices: []string{"semver", "date-based"},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		switch m.state {
		case selectSchemeView, selectSemverBumpView:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter", " ":
				selectedChoice := m.choices[m.cursor]
				if m.state == selectSchemeView {
					m.scheme = selectedChoice
					if m.scheme == "semver" {
						m.state = selectSemverBumpView
						m.choices = []string{"Major", "Minor", "Patch"}
						m.cursor = 0
						return m, nil
					} else {
						m.bump = "date"
						return m, tea.Quit
					}
				} else {
					m.bump = selectedChoice
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var s string

	switch m.state {
	case selectSchemeView:
		s = "Version scheme?\n\n"
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
	case selectSemverBumpView:
		s = "Select SemVer bump type:\n\n"
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
	}
	s += "\nPress Ctrl+C or q to quit early.\n"
	return s
}

func main() {
	// 1. Check Git status
	clean, err := utils.IsGitClean()
	if err != nil {
		fmt.Printf("Error checking git status: %v\n", err)
		os.Exit(1)
	}
	if !clean {
		fmt.Println("Error: Git working directory is not clean. Please commit or stash changes.")
		os.Exit(1)
	}

	// 2. Get current branch
	originalBranch, err := utils.GetCurrentBranch()
	if err != nil {
		fmt.Printf("Error getting current branch: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current branch: %s\n", originalBranch)

	// 3. Run the TUI to get user input
	initial := initialModel()
	p := tea.NewProgram(initial)
	finalModelInterface, err := p.Run()
	if err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	finalModel, ok := finalModelInterface.(model)
	if !ok {
		fmt.Println("Error: Could not get final model state from TUI.")
		os.Exit(1)
	}

	if finalModel.scheme == "" || (finalModel.scheme == "semver" && finalModel.bump == "") {
		fmt.Println("Operation cancelled by user.")
		os.Exit(0)
	}

	// 4. Detect project and calculate new version (after TUI)
	project, versionFile := utils.DetectProject()
	if project == utils.UnknownProject {
		fmt.Println("Error: Could not detect project type (mix.exs or package.json not found).")
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	bumpType := finalModel.bump
	if finalModel.scheme == "date-based" {
		bumpType = "date"
	}

	newVersion, currentVersion, err := utils.CalculateNewVersion(project, versionFile, bumpType)
	if err != nil {
		fmt.Printf("Error calculating new version: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 5. Create new branch
	releaseBranch := fmt.Sprintf("release/%s", newVersion)
	fmt.Printf("Creating branch: %s\n", releaseBranch)
	if err := utils.CreateBranch(releaseBranch); err != nil {
		fmt.Printf("Error creating branch: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 6. Write version update
	fmt.Printf("Updating %s from %s to %s\n", versionFile, currentVersion, newVersion)
	if err := utils.WriteVersion(project, versionFile, currentVersion, newVersion); err != nil {
		fmt.Printf("Error writing version update: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 7. Stage changes
	fmt.Printf("Staging %s\n", versionFile)
	if err := utils.StageFile(versionFile); err != nil {
		fmt.Printf("Error staging file: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 8. Commit changes
	commitMessage := fmt.Sprintf("version bump %s", newVersion)
	fmt.Printf("Committing: %s\n", commitMessage)
	if err := utils.CommitChanges(commitMessage); err != nil {
		fmt.Printf("Error committing changes: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	fmt.Printf("Successfully created branch '%s', committed version bump. You are now on branch '%s'.\n", releaseBranch, releaseBranch)
}
