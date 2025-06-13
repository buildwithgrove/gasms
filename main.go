package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateLoading state = iota
	stateTable
	stateCommand
	stateSearch
)

type model struct {
	state        state
	config       *Config
	applications []Application
	cursor       int
	commandInput string
	searchInput  string
	searchResults []int
	searchIndex  int
	err          error
	loading      bool
	width        int
	height       int
	splashArt    string
	logoLine     string
}

type applicationsLoadedMsg struct {
	apps []Application
	err  error
}

type configLoadedMsg struct {
	config *Config
	err    error
}

func loadSplashArt() string {
	content, err := ioutil.ReadFile("art/splash.txt")
	if err != nil {
		return "GASMS\nLoading..."
	}
	return string(content)
}

func loadLogoLine() string {
	content, err := ioutil.ReadFile("art/logo.txt")
	if err != nil {
		return "GASMS"
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		return strings.TrimSpace(lines[0])
	}
	return "GASMS"
}

func loadApplicationsCmd(rpcEndpoint, gateway string) tea.Cmd {
	return func() tea.Msg {
		apps, err := QueryApplications(rpcEndpoint, gateway)
		return applicationsLoadedMsg{apps: apps, err: err}
	}
}

func loadConfigCmd() tea.Cmd {
	return func() tea.Msg {
		config, err := LoadConfig("config.yaml")
		return configLoadedMsg{config: config, err: err}
	}
}

func initialModel() model {
	return model{
		state:     stateLoading,
		splashArt: loadSplashArt(),
		logoLine:  loadLogoLine(),
		loading:   true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadConfigCmd(),
		tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return "boot_complete"
		}),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case configLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.config = msg.config
		
		// Start loading applications from main network
		if mainNetwork, exists := m.config.Config.Networks["main"]; exists && len(mainNetwork.Gateways) > 0 {
			return m, loadApplicationsCmd(mainNetwork.RPCEndpoint, mainNetwork.Gateways[0])
		}
		m.err = fmt.Errorf("main network not found in config")
		return m, nil

	case applicationsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.applications = msg.apps
		
	case string:
		if msg == "boot_complete" && m.config != nil {
			m.state = stateTable
			m.loading = false
		}

	case tea.KeyMsg:
		switch m.state {
		case stateLoading:
			return m, nil
			
		case stateTable:
			return m.updateTable(msg)
			
		case stateCommand:
			return m.updateCommand(msg)
			
		case stateSearch:
			return m.updateSearch(msg)
		}
	}

	return m, nil
}

func (m model) updateTable(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
		
	case ":":
		m.state = stateCommand
		m.commandInput = ""
		
	case "/":
		m.state = stateSearch
		m.searchInput = ""
		
	case "r":
		if m.config != nil {
			if mainNetwork, exists := m.config.Config.Networks["main"]; exists && len(mainNetwork.Gateways) > 0 {
				m.loading = true
				return m, loadApplicationsCmd(mainNetwork.RPCEndpoint, mainNetwork.Gateways[0])
			}
		}
		
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		
	case "down", "j":
		if m.cursor < len(m.applications)-1 {
			m.cursor++
		}
		
	case "home", "g":
		m.cursor = 0
		
	case "end", "G":
		m.cursor = len(m.applications) - 1
	}
	
	return m, nil
}

func (m model) updateCommand(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := strings.TrimSpace(m.commandInput)
		m.state = stateTable
		
		switch cmd {
		case "q", "quit":
			return m, tea.Quit
		}
		
	case "esc":
		m.state = stateTable
		
	case "backspace":
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
		
	default:
		if msg.Type == tea.KeyRunes {
			m.commandInput += string(msg.Runes)
		}
	}
	
	return m, nil
}

func (m model) updateSearch(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.performSearch()
		m.state = stateTable
		
	case "esc":
		m.state = stateTable
		
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		
	default:
		if msg.Type == tea.KeyRunes {
			m.searchInput += string(msg.Runes)
		}
	}
	
	return m, nil
}

func (m *model) performSearch() {
	m.searchResults = []int{}
	searchTerm := strings.ToLower(m.searchInput)
	
	for i, app := range m.applications {
		if strings.Contains(strings.ToLower(app.Address), searchTerm) ||
		   strings.Contains(strings.ToLower(app.ServiceID), searchTerm) {
			m.searchResults = append(m.searchResults, i)
		}
	}
	
	if len(m.searchResults) > 0 {
		m.cursor = m.searchResults[0]
		m.searchIndex = 0
	}
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
	}

	switch m.state {
	case stateLoading:
		return m.renderLoading()
	case stateTable:
		return m.renderTable()
	case stateCommand:
		return m.renderTable() + m.renderCommandLine()
	case stateSearch:
		return m.renderTable() + m.renderSearchLine()
	}

	return ""
}

func (m model) renderLoading() string {
	// Create a simple centered layout without forcing width/height
	lines := strings.Split(m.splashArt, "\n")
	
	// Calculate padding for centering
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	
	// Center each line
	var centeredLines []string
	for _, line := range lines {
		padding := (maxWidth - len(line)) / 2
		centeredLine := strings.Repeat(" ", padding) + line
		centeredLines = append(centeredLines, centeredLine)
	}
	
	content := strings.Join(centeredLines, "\n")
	
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(m.width).
		Height(m.height)

	return style.Render(content)
}

func (m model) renderTable() string {
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("255"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Header
	header := headerStyle.Render("r:Refresh  /:Search  :Command") + 
		strings.Repeat(" ", max(0, m.width-50)) + 
		titleStyle.Render(m.logoLine)

	// Table header
	tableHeader := fmt.Sprintf("%-20s %-15s %-20s %-20s", 
		"App Address", "Stake (POKT)", "Service ID", "Gateway")

	var rows []string
	rows = append(rows, tableHeader)
	rows = append(rows, strings.Repeat("-", m.width-2))

	// Table rows
	gateway := ""
	if m.config != nil {
		if mainNetwork, exists := m.config.Config.Networks["main"]; exists && len(mainNetwork.Gateways) > 0 {
			gateway = TruncateAddress(mainNetwork.Gateways[0], 20)
		}
	}

	for i, app := range m.applications {
		row := fmt.Sprintf("%-20s %-15s %-20s %-20s",
			TruncateAddress(app.Address, 20),
			fmt.Sprintf("%.2f", app.StakePOKT),
			app.ServiceID,
			gateway)

		if i == m.cursor {
			row = selectedStyle.Render(row)
		} else {
			row = normalStyle.Render(row)
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	
	if m.loading {
		content += "\n\nRefreshing..."
	}

	return header + "\n" + content
}

func (m model) renderCommandLine() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255")).
		Width(m.width)

	return "\n" + style.Render(":"+m.commandInput)
}

func (m model) renderSearchLine() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255")).
		Width(m.width)

	return "\n" + style.Render("/"+m.searchInput)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
