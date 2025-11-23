package ui
import (
	"fmt"
	"os"
	"strings"
	"github.com/chann44/tidy/internal"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
type AppState int
const (
	StateMainMenu AppState = iota
	StateInstalling
	StateScanning
	StateRunningScript
	StateAddingPackages
	StateShowingInfo
	StateSelectingScript
	StateQuitting
)
type PackageProgress struct {
	name     string
	version  string
	progress int
	done     bool
	failed   bool
}
type Model struct {
	state          AppState
	choices        []string
	cursor         int
	selected       map[int]struct{}
	width          int
	height         int
	message        string
	scripts        []string
	scriptCursor   int
	installing     bool
	progress       string
	err            error
	packages       []PackageProgress
	installedCount int
}
func NewModel() Model {
	return Model{
		state: StateMainMenu,
		choices: []string{
			"📦 Install dependencies",
			"🔍 Scan & install from codebase",
			"▶️  Run a script",
			"➕ Add new packages",
			"ℹ️  Show project info",
			"❌ Exit",
		},
		cursor:   0,
		selected: make(map[int]struct{}),
		scripts:  []string{},
		packages: []PackageProgress{},
	}
}
func (m Model) Init() tea.Cmd {
	return nil
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case installCompleteMsg:
		m.installing = false
		m.message = msg.message
		m.state = StateMainMenu
		m.packages = []PackageProgress{}
		return m, nil
	case installProgressMsg:
		m.progress = msg.progress
		if msg.pkg != "" {
			found := false
			for i := range m.packages {
				if m.packages[i].name == msg.pkg {
					m.packages[i].progress = msg.pkgProgress
					m.packages[i].done = msg.pkgDone
					m.packages[i].failed = msg.pkgFailed
					found = true
					break
				}
			}
			if !found && msg.pkg != "" {
				m.packages = append(m.packages, PackageProgress{
					name:     msg.pkg,
					version:  msg.version,
					progress: msg.pkgProgress,
					done:     msg.pkgDone,
					failed:   msg.pkgFailed,
				})
			}
			if msg.pkgDone {
				m.installedCount++
			}
		}
		if m.state == StateInstalling {
			return m, installDependenciesCmd()
		} else if m.state == StateScanning {
			return m, scanAndInstallCmd()
		}
		return m, nil
	case errorMsg:
		m.err = msg.err
		m.state = StateMainMenu
		m.packages = []PackageProgress{}
		return m, nil
	case scriptsLoadedMsg:
		m.scripts = msg.scripts
		m.state = StateSelectingScript
		return m, nil
	case runScriptMsg:
		m.state = StateQuitting
		return m, tea.Sequence(
			tea.Quit,
			func() tea.Msg {
				wd, _ := os.Getwd()
				jsn, _ := internal.ReadJson(wd)
				runner := internal.NewScriptRunner(jsn)
				runner.Run(msg.scriptName)
				return nil
			},
		)
	}
	return m, nil
}
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateMainMenu:
		return m.handleMainMenuKeys(msg)
	case StateSelectingScript:
		return m.handleScriptSelectionKeys(msg)
	default:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.state = StateQuitting
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m Model) handleMainMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.state = StateQuitting
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "enter", " ":
		return m.handleMenuSelection()
	}
	return m, nil
}
func (m Model) handleScriptSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.state = StateMainMenu
		m.scriptCursor = 0
		return m, nil
	case "up", "k":
		if m.scriptCursor > 0 {
			m.scriptCursor--
		}
	case "down", "j":
		if m.scriptCursor < len(m.scripts)-1 {
			m.scriptCursor++
		}
	case "enter", " ":
		if len(m.scripts) > 0 {
			scriptName := m.scripts[m.scriptCursor]
			return m, runScriptCmd(scriptName)
		}
	}
	return m, nil
}
func (m Model) handleMenuSelection() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		m.state = StateInstalling
		m.installing = true
		m.message = ""
		m.packages = []PackageProgress{}
		m.installedCount = 0
		return m, installDependenciesCmd()
	case 1:
		m.state = StateScanning
		m.installing = true
		m.message = ""
		m.packages = []PackageProgress{}
		m.installedCount = 0
		return m, scanAndInstallCmd()
	case 2:
		return m, loadScriptsCmd()
	case 3:
		m.message = "Adding packages is not yet implemented in TUI mode.\nUse: tidy add <packages>"
		return m, nil
	case 4:
		return m, showProjectInfoCmd()
	case 5:
		m.state = StateQuitting
		return m, tea.Quit
	}
	return m, nil
}
func (m Model) View() string {
	if m.state == StateQuitting {
		return ""
	}
	switch m.state {
	case StateMainMenu:
		return m.renderMainMenu()
	case StateInstalling, StateScanning:
		return m.renderInstalling()
	case StateSelectingScript:
		return m.renderScriptSelection()
	case StateShowingInfo:
		return m.renderProjectInfo()
	default:
		return m.renderMainMenu()
	}
}
func (m Model) renderMainMenu() string {
	retroGreen := lipgloss.Color("#00FF00")
	retroCyan := lipgloss.Color("#00FFFF")
	retroYellow := lipgloss.Color("#FFFF00")
	titleStyle := lipgloss.NewStyle().
		Foreground(retroCyan).
		Bold(true).
		Padding(1, 0).
		Width(m.width).
		Align(lipgloss.Center)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(retroGreen).
		Foreground(retroGreen).
		Padding(2, 4).
		Width(m.width - 4).
		Height(m.height - 8)
	helpStyle := lipgloss.NewStyle().
		Foreground(retroYellow).
		Width(m.width).
		Align(lipgloss.Center)
	title := titleStyle.Render("╔═══════════════════════════════════════╗\n║  T I D Y  -  P A C K A G E  M G R    ║\n╚═══════════════════════════════════════╝")
	var menu string
	menu += "\n>>> SELECT AN OPTION <<<\n\n"
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "▶ "
			choice = lipgloss.NewStyle().
				Foreground(retroCyan).
				Bold(true).
				Render(choice)
		}
		menu += fmt.Sprintf("%s%s\n", cursor, choice)
	}
	if m.message != "" {
		messageStyle := lipgloss.NewStyle().
			Foreground(retroCyan).
			Padding(1, 0)
		menu += "\n" + messageStyle.Render(m.message)
	}
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Padding(1, 0)
		menu += "\n" + errorStyle.Render(fmt.Sprintf("ERROR: %v", m.err))
	}
	box := boxStyle.Render(menu)
	help := helpStyle.Render("[ ↑/↓: NAVIGATE ] [ ENTER: SELECT ] [ Q: QUIT ]")
	return lipgloss.JoinVertical(lipgloss.Left, title, box, help)
}
func renderProgressBar(progress int, width int) string {
	filled := int(float64(width) * float64(progress) / 100.0)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s] %d%%", bar, progress)
}
func (m Model) renderInstalling() string {
	retroGreen := lipgloss.Color("#00FF00")
	retroCyan := lipgloss.Color("#00FFFF")
	retroRed := lipgloss.Color("#FF0000")
	titleStyle := lipgloss.NewStyle().
		Foreground(retroCyan).
		Bold(true).
		Padding(1, 0).
		Width(m.width).
		Align(lipgloss.Center)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(retroGreen).
		Foreground(retroGreen).
		Padding(2, 4).
		Width(m.width - 4).
		Height(m.height - 6)
	var title string
	if m.state == StateScanning {
		title = titleStyle.Render("╔═══════════════════════════════════════╗\n║      S C A N N I N G  C O D E        ║\n╚═══════════════════════════════════════╝")
	} else {
		title = titleStyle.Render("╔═══════════════════════════════════════╗\n║    I N S T A L L I N G  P K G S      ║\n╚═══════════════════════════════════════╝")
	}
	var content string
	if m.state == StateScanning {
		content = ">>> SCANNING CODEBASE FOR PACKAGES...\n\n"
	} else {
		content = ">>> INSTALLING DEPENDENCIES...\n\n"
	}
	if len(m.packages) > 0 {
		maxVisible := m.height - 15
		start := 0
		if len(m.packages) > maxVisible {
			start = len(m.packages) - maxVisible
		}
		for i := start; i < len(m.packages); i++ {
			pkg := m.packages[i]
			status := ""
			color := retroGreen
			if pkg.failed {
				status = "✗ FAILED"
				color = retroRed
			} else if pkg.done {
				status = "✓ DONE"
				color = retroCyan
			} else {
				status = renderProgressBar(pkg.progress, 30)
			}
			pkgLine := fmt.Sprintf("%-25s %s", pkg.name+"@"+pkg.version, status)
			content += lipgloss.NewStyle().Foreground(color).Render(pkgLine) + "\n"
		}
		content += fmt.Sprintf("\n>>> INSTALLED: %d / %d <<<", m.installedCount, len(m.packages))
	} else {
		content += "[████████████████████████████████████████] INITIALIZING...\n"
	}
	box := boxStyle.Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}
func (m Model) renderScriptSelection() string {
	retroGreen := lipgloss.Color("#00FF00")
	retroCyan := lipgloss.Color("#00FFFF")
	retroYellow := lipgloss.Color("#FFFF00")
	titleStyle := lipgloss.NewStyle().
		Foreground(retroCyan).
		Bold(true).
		Padding(1, 0).
		Width(m.width).
		Align(lipgloss.Center)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(retroGreen).
		Foreground(retroGreen).
		Padding(2, 4).
		Width(m.width - 4).
		Height(m.height - 8)
	helpStyle := lipgloss.NewStyle().
		Foreground(retroYellow).
		Width(m.width).
		Align(lipgloss.Center)
	title := titleStyle.Render("╔═══════════════════════════════════════╗\n║      S E L E C T  S C R I P T        ║\n╚═══════════════════════════════════════╝")
	var menu string
	if len(m.scripts) == 0 {
		menu = ">>> NO SCRIPTS FOUND IN PACKAGE.JSON <<<"
	} else {
		menu = ">>> AVAILABLE SCRIPTS <<<\n\n"
		for i, script := range m.scripts {
			cursor := "  "
			if m.scriptCursor == i {
				cursor = "▶ "
				script = lipgloss.NewStyle().
					Foreground(retroCyan).
					Bold(true).
					Render(script)
			}
			menu += fmt.Sprintf("%s%s\n", cursor, script)
		}
	}
	box := boxStyle.Render(menu)
	help := helpStyle.Render("[ ↑/↓: NAVIGATE ] [ ENTER: RUN ] [ ESC: BACK ] [ Q: QUIT ]")
	return lipgloss.JoinVertical(lipgloss.Left, title, box, help)
}
func (m Model) renderProjectInfo() string {
	retroGreen := lipgloss.Color("#00FF00")
	retroCyan := lipgloss.Color("#00FFFF")
	titleStyle := lipgloss.NewStyle().
		Foreground(retroCyan).
		Bold(true).
		Padding(1, 0).
		Width(m.width).
		Align(lipgloss.Center)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(retroGreen).
		Foreground(retroGreen).
		Padding(2, 4).
		Width(m.width - 4).
		Height(m.height - 6)
	title := titleStyle.Render("╔═══════════════════════════════════════╗\n║    P R O J E C T  I N F O            ║\n╚═══════════════════════════════════════╝")
	content := ">>> PROJECT INFORMATION <<<\n\n"
	content += "PRESS ANY KEY TO RETURN TO MENU"
	box := boxStyle.Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}