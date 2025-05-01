package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rossvz/bump/utils"
)

type viewState int

const (
	selectSchemeView viewState = iota
	selectSemverBumpView
	editBranchNameView
)

type model struct {
	state   viewState
	cursor  int
	choices []string

	scheme      string
	bump        string
	project     utils.ProjectType
	versionFile string

	branchInput textinput.Model
	newVersion  string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "release/"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return model{
		state:       selectSchemeView,
		choices:     []string{"semver", "date-based"},
		branchInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case editBranchNameView:
			switch msg.Type {
			case tea.KeyEnter:
				return m, tea.Quit
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}
			m.branchInput, cmd = m.branchInput.Update(msg)
			return m, cmd
		default:
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
							// Calculate version before branch name edit
							if m.project == utils.UnknownProject {
								fmt.Println("Warning: Could not detect project type. Version bumping might not work correctly.")
							} else {
								currentVersion, err := utils.GetCurrentVersion(m.project, m.versionFile)
								if err != nil {
									fmt.Printf("Error getting current version from %s: %v\n", m.versionFile, err)
								} else {
									fmt.Printf("Current version: %s\n", currentVersion)
								}
							}
							newVersion, _, err := utils.CalculateNewVersion(m.project, m.versionFile, "date")
							if err != nil {
								fmt.Printf("Error calculating version: %v\n", err)
								return m, tea.Quit
							}
							m.newVersion = newVersion
							m.branchInput.SetValue(fmt.Sprintf("release/%s", newVersion))
							m.state = editBranchNameView
							return m, nil
						}
					} else if m.state == selectSemverBumpView {
						m.bump = selectedChoice
						// Calculate version before branch name edit
						if m.project == utils.UnknownProject {
							fmt.Println("Error: Cannot calculate version - unknown project type.")
							return m, tea.Quit
						}
						newVersion, _, err := utils.CalculateNewVersion(m.project, m.versionFile, selectedChoice)
						if err != nil {
							fmt.Printf("Error calculating version: %v\n", err)
							return m, tea.Quit
						}
						m.newVersion = newVersion
						m.branchInput.SetValue(fmt.Sprintf("release/%s", newVersion))
						m.state = editBranchNameView
						return m, nil
					}
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
	case editBranchNameView:
		s = fmt.Sprintf("Enter branch name (version will be %s):\n\n", m.newVersion)
		s += m.branchInput.View()
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

	// 3. Detect project type and get current version
	project, versionFile := utils.DetectProject()
	if project == utils.UnknownProject {
		// If unknown, maybe default to git tag or inform user?
		// For now, we'll proceed, but version calculation might fail later
		fmt.Println("Warning: Could not detect project type. Version bumping might not work correctly.")
	} else {
		currentVersion, err := utils.GetCurrentVersion(project, versionFile)
		if err != nil {
			fmt.Printf("Error getting current version from %s: %v\n", versionFile, err)
			// Decide if we should exit or proceed without version info
			// os.Exit(1)
		} else {
			fmt.Printf("Current version: %s\n", currentVersion)
		}
		// Store project info in the initial model if needed later, or pass it
		// initial.project = project
		// initial.versionFile = versionFile
	}

	// 4. Run the TUI to get user input
	initial := initialModel()
	// Pass project info if needed, e.g.:
	initial.project = project
	initial.versionFile = versionFile

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

	// 5. Create new branch using the custom branch name
	branchName := finalModel.branchInput.Value()
	if branchName == "" {
		branchName = fmt.Sprintf("release/%s", finalModel.newVersion)
	}
	fmt.Printf("Creating branch: %s\n", branchName)
	if err := utils.CreateBranch(branchName); err != nil {
		fmt.Printf("Error creating branch: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 6. Write version update
	// Fetch current version again right before writing
	currentVersion, err := utils.GetCurrentVersion(finalModel.project, finalModel.versionFile)
	if err != nil {
		fmt.Printf("Error re-fetching current version for write: %v\n", err)
		// Handle error, maybe switch back and exit
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	fmt.Printf("Updating %s from %s to %s\n", finalModel.versionFile, currentVersion, finalModel.newVersion)
	if err := utils.WriteVersion(finalModel.project, finalModel.versionFile, currentVersion, finalModel.newVersion); err != nil {
		fmt.Printf("Error writing version update: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 7. Stage changes
	fmt.Printf("Staging %s\n", finalModel.versionFile)
	if err := utils.StageFile(finalModel.versionFile); err != nil {
		fmt.Printf("Error staging file: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	// 8. Commit changes
	commitMessage := fmt.Sprintf("version bump %s", finalModel.newVersion)
	fmt.Printf("Committing: %s\n", commitMessage)
	if err := utils.CommitChanges(commitMessage); err != nil {
		fmt.Printf("Error committing changes: %v\n", err)
		if switchErr := utils.CheckoutBranch(originalBranch); switchErr != nil {
			fmt.Printf("Warning: Failed to switch back to original branch '%s': %v\n", originalBranch, switchErr)
		}
		os.Exit(1)
	}

	fmt.Printf("Successfully created branch '%s', committed version bump. You are now on branch '%s'.\n", branchName, branchName)
}
