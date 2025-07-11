package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	stateNetworkSelect
	stateGatewaySelect
	stateHelp
	stateApplicationDetails
)

type model struct {
	state          state
	config         *Config
	applications   []Application
	cursor         int
	commandInput   string
	searchInput    string
	searchResults  []int
	searchIndex    int
	err            error
	loading        bool
	width          int
	height         int
	splashArt      string
	logoLine       string
	currentNetwork string
	currentGateway string
	networkList    []string
	networkCursor  int
	sortBy         string // Current sort field
	sortDesc       bool   // Sort direction (true = descending, false = ascending)
	gatewayList    []string
	gatewayCursor  int
	txHash         string    // Current upstake transaction hash to display
	txTimestamp    time.Time // When the upstake transaction was submitted
	fundTxHash     string    // Current fund transaction hash to display
	fundTimestamp  time.Time // When the fund transaction was submitted
	// Application details view
	selectedAppAddress string // Address of currently viewed application
	applicationDetails string // Raw output from show-application command
	bankBalances       string // Raw output from bank balances command
	detailsLoading     bool   // Loading state for details view
}

type applicationsLoadedMsg struct {
	apps []Application
	err  error
}

type configLoadedMsg struct {
	config *Config
	err    error
}

type upstakeCompletedMsg struct {
	txHash string
}

type applicationDetailsLoadedMsg struct {
	address     string
	appDetails  string
	bankBalance string
	err         error
}

type fundCompletedMsg struct {
	txHash string
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
		sortBy:    "service", // Default sort by service
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

		// Build network list and set defaults
		m.networkList = []string{}
		for name := range m.config.Config.Networks {
			m.networkList = append(m.networkList, name)
		}

		// Default to first network found
		if len(m.networkList) == 0 {
			m.err = fmt.Errorf("no networks found in config")
			return m, nil
		}

		m.currentNetwork = m.networkList[0]
		if firstNetwork, exists := m.config.Config.Networks[m.currentNetwork]; exists && len(firstNetwork.Gateways) > 0 {
			m.currentGateway = firstNetwork.Gateways[0]
			return m, loadApplicationsCmd(firstNetwork.RPCEndpoint, firstNetwork.Gateways[0])
		}
		m.err = fmt.Errorf("first network %s has no gateways configured", m.currentNetwork)
		return m, nil

	case applicationsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.applications = msg.apps
		m.sortApplications() // Sort applications after loading
		m.loading = false    // clear loading state

	case string:
		if msg == "boot_complete" && m.config != nil {
			m.state = stateTable
			m.loading = false
		} else if msg == "clear_tx_hash" {
			m.txHash = ""
		} else if msg == "clear_fund_hash" {
			m.fundTxHash = ""
		} else if strings.HasPrefix(msg, "Upstake failed:") {
			m.err = fmt.Errorf("%s", msg)
		} else if strings.HasPrefix(msg, "Fund failed:") {
			m.err = fmt.Errorf("%s", msg)
		}

	case upstakeCompletedMsg:
		// Set transaction hash and timestamp for display
		m.txHash = msg.txHash
		m.txTimestamp = time.Now()

		// Refresh application data after successful upstake
		if m.config != nil {
			if network, exists := m.config.Config.Networks[m.currentNetwork]; exists && len(network.Gateways) > 0 {
				m.loading = true
				return m, tea.Batch(
					loadApplicationsCmd(network.RPCEndpoint, m.currentGateway),
					tea.Tick(time.Second*10, func(t time.Time) tea.Msg {
						return "clear_tx_hash"
					}),
				)
			}
		}

	case fundCompletedMsg:
		// Set fund transaction hash and timestamp for display
		m.fundTxHash = msg.txHash
		m.fundTimestamp = time.Now()

		// Set timer to clear fund hash after 10 seconds
		return m, tea.Tick(time.Second*10, func(t time.Time) tea.Msg {
			return "clear_fund_hash"
		})

	case applicationDetailsLoadedMsg:
		m.detailsLoading = false
		if msg.err != nil {
			m.err = msg.err
			m.state = stateTable // Return to table on error
		} else {
			m.selectedAppAddress = msg.address
			m.applicationDetails = msg.appDetails
			m.bankBalances = msg.bankBalance
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

		case stateNetworkSelect:
			return m.updateNetworkSelect(msg)

		case stateGatewaySelect:
			return m.updateGatewaySelect(msg)

		case stateHelp:
			return m.updateHelp(msg)

		case stateApplicationDetails:
			return m.updateApplicationDetails(msg)
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

	case "n":
		m.state = stateNetworkSelect
		m.networkCursor = 0

	case "r":
		if m.config != nil {
			if network, exists := m.config.Config.Networks[m.currentNetwork]; exists && len(network.Gateways) > 0 {
				m.loading = true
				return m, loadApplicationsCmd(network.RPCEndpoint, network.Gateways[0])
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

	case "u":
		if len(m.applications) > 0 && m.cursor < len(m.applications) {
			currentApp := m.applications[m.cursor]
			m.state = stateCommand
			m.commandInput = "u " + currentApp.Address + " "
		}

	case "enter":
		if len(m.applications) > 0 && m.cursor < len(m.applications) {
			currentApp := m.applications[m.cursor]
			return m.showApplicationDetails(currentApp.Address)
		}

	case "f":
		if len(m.applications) > 0 && m.cursor < len(m.applications) {
			currentApp := m.applications[m.cursor]
			m.state = stateCommand
			m.commandInput = "f " + currentApp.Address + " "
		}
	}

	return m, nil
}

func (m model) updateCommand(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := strings.TrimSpace(m.commandInput)
		m.commandInput = "" // Clear command input
		m.state = stateTable

		switch cmd {
		case "q", "quit":
			return m, tea.Quit
		case "n", "network":
			m.state = stateNetworkSelect
			m.networkCursor = 0
		case "g", "gateway":
			m.state = stateGatewaySelect
			m.gatewayCursor = 0
			// Build gateway list from current network
			if m.config != nil {
				if network, exists := m.config.Config.Networks[m.currentNetwork]; exists {
					m.gatewayList = network.Gateways
				}
			}
		// Sorting commands
		case "ss", "sort status":
			m.setSortBy("status")
		case "sg", "sort gateway":
			m.setSortBy("gateway")
		case "sa", "sort address":
			m.setSortBy("address")
		case "sp", "sort stake":
			m.setSortBy("stake")
		case "sv", "sort service":
			m.setSortBy("service")
		// Sort direction commands
		case "asc":
			m.sortDesc = false
			m.sortApplications()
		case "desc":
			m.sortDesc = true
			m.sortApplications()
		case "h", "help":
			m.state = stateHelp
		default:
			// Handle upstake command: "u <address> <amount>"
			if strings.HasPrefix(cmd, "u ") {
				return m.handleUpstakeCommand(cmd)
			}
			// Handle show command: "show <address>"
			if strings.HasPrefix(cmd, "show ") {
				return m.handleShowCommand(cmd)
			}
			// Handle fund command: "f <address> <amount>" or "fund <address> <amount>"
			if strings.HasPrefix(cmd, "f ") || strings.HasPrefix(cmd, "fund ") {
				return m.handleFundCommand(cmd)
			}
		}

	case "esc":
		m.state = stateTable

	case "backspace":
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}

	case " ":
		m.commandInput += " "

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

	case " ":
		m.searchInput += " "

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

func (m model) updateNetworkSelect(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.networkCursor < len(m.networkList) {
			selectedNetwork := m.networkList[m.networkCursor]
			if network, exists := m.config.Config.Networks[selectedNetwork]; exists && len(network.Gateways) > 0 {
				m.currentNetwork = selectedNetwork
				m.currentGateway = network.Gateways[0]
				m.state = stateTable
				m.loading = true
				return m, loadApplicationsCmd(network.RPCEndpoint, network.Gateways[0])
			}
		}
		m.state = stateTable

	case "esc", "q":
		m.state = stateTable

	case "up", "k":
		if m.networkCursor > 0 {
			m.networkCursor--
		}

	case "down", "j":
		if m.networkCursor < len(m.networkList)-1 {
			m.networkCursor++
		}
	}

	return m, nil
}

func (m model) updateGatewaySelect(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.gatewayCursor < len(m.gatewayList) {
			selectedGateway := m.gatewayList[m.gatewayCursor]
			if m.config != nil {
				if network, exists := m.config.Config.Networks[m.currentNetwork]; exists {
					m.currentGateway = selectedGateway
					m.state = stateTable
					m.loading = true
					return m, loadApplicationsCmd(network.RPCEndpoint, selectedGateway)
				}
			}
		}
		m.state = stateTable

	case "esc", "q":
		m.state = stateTable

	case "up", "k":
		if m.gatewayCursor > 0 {
			m.gatewayCursor--
		}

	case "down", "j":
		if m.gatewayCursor < len(m.gatewayList)-1 {
			m.gatewayCursor++
		}
	}

	return m, nil
}

func (m model) updateHelp(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "enter":
		m.state = stateTable
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
	}

	// Reserve space for command prompt at bottom (3 lines)
	commandAreaHeight := 3
	mainContentHeight := m.height - commandAreaHeight

	// Ensure mainContentHeight is never negative
	if mainContentHeight < 1 {
		mainContentHeight = 1
	}

	// Render main content based on state
	var mainContent string
	switch m.state {
	case stateLoading:
		mainContent = m.renderLoading()
	case stateTable, stateCommand, stateSearch:
		mainContent = m.renderTable()
	case stateNetworkSelect:
		mainContent = m.renderNetworkSelect()
	case stateGatewaySelect:
		mainContent = m.renderGatewaySelect()
	case stateHelp:
		mainContent = m.renderHelp()
	case stateApplicationDetails:
		mainContent = m.renderApplicationDetails()
	default:
		mainContent = ""
	}

	// Trim main content to reserved height
	mainContentLines := strings.Split(mainContent, "\n")
	if len(mainContentLines) > mainContentHeight {
		mainContentLines = mainContentLines[:mainContentHeight]
	}

	// Pad main content to exact height
	for len(mainContentLines) < mainContentHeight {
		mainContentLines = append(mainContentLines, "")
	}

	// Render command area (skip for application details view)
	var result string
	if m.state == stateApplicationDetails {
		// No command area for details view
		result = strings.Join(mainContentLines, "\n")
	} else {
		commandArea := m.renderCommandArea()
		result = strings.Join(mainContentLines, "\n") + "\n" + commandArea
	}

	return result
}

func (m model) renderCommandArea() string {
	// Create dedicated command area at bottom
	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("65")) // Muted green

	commandStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).   // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Padding(0, 1)

	// Calculate border width accounting for terminal width
	borderWidth := m.width
	if borderWidth < 1 {
		borderWidth = 80 // Fallback width
	}

	// Top border for command area
	border := borderStyle.Render(strings.Repeat("─", borderWidth))

	var commandContent string
	switch m.state {
	case stateCommand:
		commandContent = ":" + m.commandInput
	case stateSearch:
		commandContent = "/" + m.searchInput
	default:
		commandContent = "Press : for commands, / for search, h for help"
	}

	commandLine := commandStyle.Width(borderWidth).Render(commandContent)

	// Return 3-line command area: border + command + empty
	return border + "\n" + commandLine + "\n"
}

func (m model) ensureFixedHeight(content string) string {
	lines := strings.Split(content, "\n")

	// For command and search modes, preserve the last few lines (command prompt)
	// and trim from the middle (table content) instead
	if len(lines) > m.height {
		if m.state == stateCommand || m.state == stateSearch {
			// Keep first few lines (header) and last few lines (command prompt)
			// Trim from the table content in the middle
			headerLines := 8  // Approximate header size
			commandLines := 3 // Approximate command prompt size

			if len(lines) > headerLines+commandLines {
				// Keep header and command prompt, trim table content
				tableTrimCount := len(lines) - m.height
				tableStartIdx := headerLines
				tableEndIdx := len(lines) - commandLines

				// Remove excess table lines
				if tableTrimCount > 0 && tableEndIdx > tableStartIdx {
					trimFromTable := min(tableTrimCount, tableEndIdx-tableStartIdx)
					newLines := make([]string, 0, len(lines)-trimFromTable)
					newLines = append(newLines, lines[:tableStartIdx]...)
					newLines = append(newLines, lines[tableStartIdx+trimFromTable:]...)
					lines = newLines
				}
			}
		} else {
			// For other states, trim from the end as before
			if m.height > 0 && len(lines) > m.height {
				lines = lines[:m.height]
			}
		}
	}

	// Pad to exact terminal height
	for len(lines) < m.height {
		// Insert padding before the last line (command prompt) if it exists
		if (m.state == stateCommand || m.state == stateSearch) && len(lines) > 0 {
			// Insert empty line before the last line
			lastLine := lines[len(lines)-1]
			lines = lines[:len(lines)-1]
			lines = append(lines, "", lastLine)
		} else {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
		Background(lipgloss.Color("0")).   // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Align(lipgloss.Center, lipgloss.Center).
		Width(m.width).
		Height(m.height)

	return style.Render(content)
}

func (m model) renderTable() string {
	return m.renderWithHeader(m.renderTableContent())
}

func (m model) renderWithHeader(content string) string {
	header := m.renderHeader()
	return header + "\n" + content
}

func (m model) renderHeader() string {
	// Clean header without background highlighting
	headerBoxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("65")). // Muted green for border
		Padding(0, 1).
		Width(m.width)

	// 2-column layout: state and commands
	stateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true).
		Width(m.width / 3) // 33% for state

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")). // Soft grey-green
		Width(m.width*2/3 - 2)             // 67% for commands

	// Column 1: App State
	appCount := len(m.applications)
	stateContent := fmt.Sprintf("🌐 Network: %s\n🧱 Gateway: %s\n📱 Applications: %d",
		strings.ToUpper(m.currentNetwork), m.currentGateway, appCount)
	stateColumn := stateStyle.Render(stateContent)

	// Column 2: Commands (clean columns)
	commandContent := "Navigation:           Sort Columns:        Actions:\n"
	commandContent += "r: Refresh           :ss Status           :: Command\n"
	commandContent += "n: Network           :sa Address          h: Help\n"
	commandContent += "g: Gateway           :sp Stake            u: Upstake\n"
	commandContent += "/: Search            :sv Service          q: Quit"
	commandColumn := commandStyle.Render(commandContent)

	// Join 2 columns horizontally
	headerContent := lipgloss.JoinHorizontal(lipgloss.Top, stateColumn, commandColumn)

	return headerBoxStyle.Render(headerContent)
}

func (m model) renderTableContent() string {
	// Soft grey-green color scheme for table
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark grey background
		Foreground(lipgloss.Color("150"))  // Light grey-green text

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")) // Soft grey-green

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true)

	// Calculate available height for table content
	// Account for command area (3 lines) and header (8-10 lines typically)
	reservedLines := 13 // Conservative estimate
	availableHeight := m.height - reservedLines
	if availableHeight < 10 {
		availableHeight = 10 // Minimum usable table height
	}

	// Optimized column widths - prioritize service_id readability
	statusWidth := 10
	stakeWidth := 12   // Reduced from 15 (fits "99999.99" comfortably)
	serviceWidth := 25 // Increased from 20 (never truncate service IDs)
	gatewayWidth := 20
	// Remaining width for address column (account for borders and padding)
	addressWidth := m.width - statusWidth - stakeWidth - serviceWidth - gatewayWidth - 8
	if addressWidth < 20 {
		addressWidth = 20 // Minimum width for readability
	}

	tableHeader := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		statusWidth, m.getColumnHeader("ℹ️  STATUS", "status"),
		addressWidth, m.getColumnHeader("📫 App Address", "address"),
		stakeWidth, m.getColumnHeader("🪙 Stake (POKT)", "stake"),
		serviceWidth, m.getColumnHeader("⚡ Service ID", "service"),
		gatewayWidth, m.getColumnHeader("🧱 Gateway", "gateway"))

	var rows []string
	rows = append(rows, headerStyle.Render(tableHeader))
	// Create separator with GASMS branding
	gasmsText := " 🌿 G A S M S 🌿 "
	availableWidth := m.width - 4 - len(gasmsText) // Account for border padding
	if availableWidth < 0 {
		availableWidth = 0
	}
	leftPadding := availableWidth / 2
	rightPadding := availableWidth - leftPadding
	separatorText := strings.Repeat("═", leftPadding) + gasmsText + strings.Repeat("═", rightPadding)
	rows = append(rows, headerStyle.Render(separatorText))

	// Table rows (limit to available height)
	displayRows := availableHeight - 2 // Reserve space for header and separator
	if displayRows < 1 {
		displayRows = 1 // Always show at least one row
	}

	startRow := 0
	if m.cursor >= displayRows {
		startRow = m.cursor - displayRows + 1
	}

	for i := startRow; i < len(m.applications) && i < startRow+displayRows; i++ {
		app := m.applications[i]

		// Determine stake status and colors
		status, rowStyle := m.getStakeStatus(app, selectedStyle, normalStyle, i == m.cursor)

		// Use dynamic widths for consistent formatting
		row := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			statusWidth, status,
			addressWidth, TruncateAddress(app.Address, addressWidth-2),
			stakeWidth, fmt.Sprintf("%.2f", app.StakePOKT),
			serviceWidth, app.ServiceID, // Never truncate service ID
			gatewayWidth, TruncateAddress(m.currentGateway, gatewayWidth-2))

		row = rowStyle.Render(row)
		rows = append(rows, row)
	}

	tableContent := strings.Join(rows, "\n")

	// Add loading notification at bottom if loading
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")). // Bold yellow
			Bold(true).
			Align(lipgloss.Center).
			Width(m.width)
		loadingMsg := loadingStyle.Render("🔄 REFRESHING DATA...")
		tableContent += "\n" + loadingMsg
	}

	// Add transaction hash display if available
	if m.txHash != "" {
		txStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")). // Bright green
			Bold(true).
			Align(lipgloss.Center).
			Width(m.width)
		txMsg := txStyle.Render("💸 UPSTAKE TXHASH: " + m.txHash)
		tableContent += "\n" + txMsg
	}

	// Add fund transaction hash display if available
	if m.fundTxHash != "" {
		fundStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")). // Bright green
			Bold(true).
			Align(lipgloss.Center).
			Width(m.width)
		fundMsg := fundStyle.Render("💸 FUND TXHASH: " + m.fundTxHash)
		tableContent += "\n" + fundMsg
	}

	return tableContent
}

func (m model) getStakeStatus(app Application, selectedStyle, normalStyle lipgloss.Style, isSelected bool) (string, lipgloss.Style) {
	// Convert stake amount to uPOKT for comparison (StakeAmount is in uPOKT string format)
	stakeAmountInt, err := strconv.ParseInt(app.StakeAmount, 10, 64)
	if err != nil {
		stakeAmountInt = 0
	}

	// Default thresholds if config is not available
	warningThreshold := int64(2000000000) // 2000 POKT
	dangerThreshold := int64(1000000000)  // 1000 POKT

	// Use config thresholds if available
	if m.config != nil {
		warningThreshold = m.config.Config.Thresholds.WarningThreshold
		dangerThreshold = m.config.Config.Thresholds.DangerThreshold
	}

	var status string
	var style lipgloss.Style

	if stakeAmountInt >= warningThreshold {
		// Green circle for good stakes
		status = "🟢"
		if isSelected {
			style = selectedStyle
		} else {
			style = normalStyle
		}
	} else if stakeAmountInt >= dangerThreshold {
		// Yellow circle for warning stakes
		status = "🟡"
		if isSelected {
			style = selectedStyle
		} else {
			style = normalStyle
		}
	} else {
		// Red circle and red text for danger stakes
		status = "🔴"
		dangerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("160")) // Red text
		if isSelected {
			// Combine red text with selected background
			style = lipgloss.NewStyle().
				Background(lipgloss.Color("236")). // Dark grey background
				Foreground(lipgloss.Color("160"))  // Red text
		} else {
			style = dangerStyle
		}
	}

	return status, style
}

func (m *model) sortApplications() {
	sort.Slice(m.applications, func(i, j int) bool {
		var result bool
		switch m.sortBy {
		case "status":
			// Sort by stake amount
			stakeI, _ := strconv.ParseInt(m.applications[i].StakeAmount, 10, 64)
			stakeJ, _ := strconv.ParseInt(m.applications[j].StakeAmount, 10, 64)
			result = stakeI > stakeJ // Default: highest stakes first
		case "address":
			result = m.applications[i].Address < m.applications[j].Address
		case "stake":
			// Sort by stake amount
			stakeI, _ := strconv.ParseInt(m.applications[i].StakeAmount, 10, 64)
			stakeJ, _ := strconv.ParseInt(m.applications[j].StakeAmount, 10, 64)
			result = stakeI > stakeJ // Default: highest stakes first
		case "service":
			result = m.applications[i].ServiceID < m.applications[j].ServiceID
		case "gateway":
			result = m.currentGateway < m.currentGateway // All same gateway, so no change
		default:
			result = m.applications[i].ServiceID < m.applications[j].ServiceID
		}

		// Reverse result if descending sort
		if m.sortDesc {
			return !result
		}
		return result
	})
}

func (m *model) setSortBy(field string) {
	// Toggle direction if same field, otherwise reset to ascending
	if m.sortBy == field {
		m.sortDesc = !m.sortDesc
	} else {
		m.sortBy = field
		m.sortDesc = false // Default to ascending for new field
	}
	m.sortApplications()
}

func (m model) getColumnHeader(baseText, fieldName string) string {
	if m.sortBy == fieldName {
		if m.sortDesc {
			return baseText + " 🔽"
		} else {
			return baseText + " 🔼"
		}
	}
	return baseText
}

func (m model) renderCommandMode() string {
	// Render table with reduced height to make room for command line
	header := m.renderHeader()
	tableContent := m.renderTableContent()

	// Create command line
	cmdLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).   // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("65")). // Muted green border
		Width(m.width).
		Padding(0, 1)

	cmdLine := cmdLineStyle.Render(":" + m.commandInput)

	return header + "\n" + tableContent + "\n" + cmdLine
}

func (m model) renderSearchMode() string {
	// Render table with reduced height to make room for search line
	header := m.renderHeader()
	tableContent := m.renderTableContent()

	// Create search line
	searchLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).   // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("108")). // Soft grey-green for search
		Width(m.width).
		Padding(0, 1)

	searchLine := searchLineStyle.Render("/" + m.searchInput)

	return header + "\n" + tableContent + "\n" + searchLine
}

func (m model) renderNetworkSelect() string {
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).   // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true).
		Align(lipgloss.Center)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark grey background
		Foreground(lipgloss.Color("150")). // Light grey-green text
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")) // Soft grey-green

	// Header
	header := headerStyle.Render("Select Network (Enter to switch, Esc to cancel)")

	// Title
	title := titleStyle.Width(m.width).Render("Available Networks")

	var rows []string
	rows = append(rows, "")
	rows = append(rows, title)
	rows = append(rows, "")

	// Network list
	for i, network := range m.networkList {
		indicator := "  "
		if network == m.currentNetwork {
			indicator = "* "
		}

		row := indicator + strings.ToUpper(network)

		if m.config != nil {
			if net, exists := m.config.Config.Networks[network]; exists {
				row += fmt.Sprintf(" (%s)", TruncateAddress(net.RPCEndpoint, 30))
			}
		}

		if i == m.networkCursor {
			row = selectedStyle.Render(row)
		} else {
			row = normalStyle.Render(row)
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	return header + "\n" + content
}

func (m model) renderGatewaySelect() string {
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).   // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true).
		Align(lipgloss.Center)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark grey background
		Foreground(lipgloss.Color("150")). // Light grey-green text
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")) // Soft grey-green

	// Header
	header := headerStyle.Render("Select Gateway (Enter to switch, Esc to cancel)")

	// Title
	title := titleStyle.Width(m.width).Render("Available Gateways")

	var rows []string
	rows = append(rows, "")
	rows = append(rows, title)
	rows = append(rows, "")

	// Gateway list
	for i, gateway := range m.gatewayList {
		indicator := "  "
		if gateway == m.currentGateway {
			indicator = "* "
		}

		row := indicator + TruncateAddress(gateway, 50)

		if i == m.gatewayCursor {
			row = selectedStyle.Render(row)
		} else {
			row = normalStyle.Render(row)
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	return header + "\n" + content
}

func (m model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Padding(1, 2).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("65")).
		Width(m.width - 4)

	helpContent := `GASMS - Grove AppStakes Management System

NAVIGATION:
  ↑/k, ↓/j        Navigate up/down
  g, G            Go to top/bottom
  u               Upstake selected application (add to current stake)
  f               Fund selected application
  enter           Show application details
  
COMMANDS (prefix with :):
  q, quit         Quit application
  h, help         Show this help
  n, network      Switch network
  g, gateway      Switch gateway
  u <addr> <amt>  Upstake application (add amount to current stake)
  f <addr> <amt>  Fund application (send tokens)
  show <addr>     Show application details
  
SORTING:
  ss, sort status    Sort by stake status (high to low)
  sa, sort address   Sort by address (A-Z)
  sp, sort stake     Sort by stake amount (high to low)
  sv, sort service   Sort by service ID (A-Z)
  sg, sort gateway   Sort by gateway
  
SEARCH:
  /               Search applications (by address or service ID)
  
REFRESH:
  r               Refresh application data

STAKE STATUS INDICATORS:
  🟢              Healthy stake (≥ warning threshold)
  🟡              Warning stake (between thresholds)  
  🔴              Danger stake (< danger threshold)

Press ESC, Enter, or q to return to main view.`

	return helpStyle.Render(helpContent)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) handleUpstakeCommand(cmd string) (model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) < 3 {
		m.err = fmt.Errorf("usage: u <address> <amount>")
		return m, nil
	}

	address := parts[1]
	amountStr := parts[2]

	// Validate amount is numeric
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		m.err = fmt.Errorf("amount must be a positive integer: %s", amountStr)
		return m, nil
	}

	// Find the application to get its service ID
	var serviceID string
	for _, app := range m.applications {
		if app.Address == address {
			serviceID = app.ServiceID
			break
		}
	}

	if serviceID == "" {
		m.err = fmt.Errorf("application not found: %s", address)
		return m, nil
	}

	// Execute upstake in background
	return m, m.executeUpstake(address, serviceID, amount)
}

func (m model) executeUpstake(address, serviceID string, amount int64) tea.Cmd {
	return func() tea.Msg {
		txHash, err := upstakeApplication(address, serviceID, amount, m.config, m.currentNetwork)
		if err != nil {
			return fmt.Sprintf("Upstake failed: %v", err)
		}
		return upstakeCompletedMsg{txHash: txHash}
	}
}

func upstakeApplication(address, serviceID string, amount int64, config *Config, networkName string) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config not loaded")
	}

	network, exists := config.Config.Networks[networkName]
	if !exists {
		return "", fmt.Errorf("network not found: %s", networkName)
	}

	// Note: Bank address field is available in config but not currently used for --from
	// The --from parameter uses the application address instead

	// Get current stake amount
	currentStake, err := getCurrentStake(address, network.RPCEndpoint, networkName)
	if err != nil {
		return "", fmt.Errorf("failed to get current stake: %v", err)
	}

	var newStake int64
	if currentStake == -1 {
		// New application
		newStake = amount
	} else {
		// Existing application, increment
		newStake = currentStake + amount
	}

	// Create temporary config file
	tempDir := "/tmp"
	configFile := filepath.Join(tempDir, fmt.Sprintf("gasms_upstake_%s_%d.yaml", address, time.Now().Unix()))

	configContent := fmt.Sprintf(`stake_amount: %dupokt
service_ids:
  - "%s"
address: %s
`, newStake, serviceID, address)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create config file: %v", err)
	}

	// Clean up temp file when done
	defer os.Remove(configFile)

	// Determine chain ID and node based on network
	var chainID, node string
	switch networkName {
	case "pocket":
		chainID = "pocket"
		node = "https://shannon-grove-rpc.mainnet.poktroll.com"
	case "pocket-beta":
		chainID = "pocket-beta"
		node = "https://shannon-testnet-grove-rpc.beta.poktroll.com"
	default:
		return "", fmt.Errorf("unsupported network: %s", networkName)
	}

	// Execute pocketd command using application address for --from
	cmd := exec.Command("pocketd", "tx", "application", "stake-application",
		"--config="+configFile,
		"--from="+address,
		"--node="+node,
		"--chain-id="+chainID,
		"--fees=20000upokt",
		"--home="+os.Getenv("HOME")+"/.pocket",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pocketd command failed: %v, output: %s", err, string(output))
	}

	// Parse transaction hash from output
	outputStr := string(output)
	txHash := ""

	// Look for txhash in the output (pocketd outputs in various formats)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Handle formats: "txhash: ABC123", "txhash:ABC123", or just "ABC123" on its own
		if strings.HasPrefix(strings.ToLower(line), "txhash:") {
			txHash = strings.TrimSpace(strings.TrimPrefix(line, "txhash:"))
			txHash = strings.TrimSpace(strings.TrimPrefix(txHash, " "))
			break
		} else if len(line) == 64 && isHexString(line) {
			// Likely a 64-character hex hash
			txHash = line
			break
		}
	}

	return txHash, nil
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func getCurrentStake(address, rpcEndpoint, networkName string) (int64, error) {
	var chainID string
	switch networkName {
	case "pocket":
		chainID = "pocket"
	case "pocket-beta":
		chainID = "pocket-beta"
	default:
		return 0, fmt.Errorf("unsupported network: %s", networkName)
	}

	cmd := exec.Command("pocketd", "query", "application", "show-application", address,
		"--node="+rpcEndpoint,
		"--chain-id="+chainID,
		"--home="+os.Getenv("HOME")+"/.pocket",
		"--output=json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if application not found
		if strings.Contains(string(output), "application not found") || strings.Contains(string(output), "key not found") {
			return -1, nil // Indicates new application
		}
		return 0, fmt.Errorf("query failed: %v, output: %s", err, string(output))
	}

	// Parse JSON to extract stake amount
	var appData map[string]interface{}
	if err := json.Unmarshal(output, &appData); err != nil {
		return 0, fmt.Errorf("failed to parse JSON output: %v", err)
	}

	// Navigate to application.stake.amount
	app, ok := appData["application"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("application field not found in response")
	}

	stake, ok := app["stake"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("stake field not found in application")
	}

	amountStr, ok := stake["amount"].(string)
	if !ok {
		return 0, fmt.Errorf("amount field not found in stake or not a string")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid stake amount: %v", err)
	}

	return amount, nil
}

func (m model) showApplicationDetails(address string) (model, tea.Cmd) {
	m.selectedAppAddress = address
	m.state = stateApplicationDetails
	m.detailsLoading = true
	m.applicationDetails = ""
	m.bankBalances = ""
	return m, m.loadApplicationDetailsCmd(address)
}

func (m model) handleShowCommand(cmd string) (model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		m.err = fmt.Errorf("usage: show <address>")
		return m, nil
	}

	address := parts[1]
	return m.showApplicationDetails(address)
}

func (m model) loadApplicationDetailsCmd(address string) tea.Cmd {
	return func() tea.Msg {
		if m.config == nil {
			return applicationDetailsLoadedMsg{
				address: address,
				err:     fmt.Errorf("config not loaded"),
			}
		}

		network, exists := m.config.Config.Networks[m.currentNetwork]
		if !exists {
			return applicationDetailsLoadedMsg{
				address: address,
				err:     fmt.Errorf("network not found: %s", m.currentNetwork),
			}
		}

		// Query application details
		appDetails, err := queryApplicationDetails(address, network.RPCEndpoint, m.currentNetwork)
		if err != nil {
			return applicationDetailsLoadedMsg{
				address: address,
				err:     fmt.Errorf("failed to query application details: %v", err),
			}
		}

		// Query bank balances
		bankBalance, err := queryBankBalances(address, network.RPCEndpoint, m.currentNetwork)
		if err != nil {
			return applicationDetailsLoadedMsg{
				address: address,
				err:     fmt.Errorf("failed to query bank balances: %v", err),
			}
		}

		return applicationDetailsLoadedMsg{
			address:     address,
			appDetails:  appDetails,
			bankBalance: bankBalance,
		}
	}
}

func queryApplicationDetails(address, rpcEndpoint, networkName string) (string, error) {
	var chainID string
	switch networkName {
	case "pocket":
		chainID = "pocket"
	case "pocket-beta":
		chainID = "pocket-beta"
	default:
		return "", fmt.Errorf("unsupported network: %s", networkName)
	}

	cmd := exec.Command("pocketd", "query", "application", "show-application", address,
		"--node="+rpcEndpoint,
		"--chain-id="+chainID,
		"--home="+os.Getenv("HOME")+"/.pocket",
		"--output=json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

func queryBankBalances(address, rpcEndpoint, networkName string) (string, error) {
	var chainID string
	switch networkName {
	case "pocket":
		chainID = "pocket"
	case "pocket-beta":
		chainID = "pocket-beta"
	default:
		return "", fmt.Errorf("unsupported network: %s", networkName)
	}

	cmd := exec.Command("pocketd", "query", "bank", "balances", address,
		"--node="+rpcEndpoint,
		"--chain-id="+chainID,
		"--home="+os.Getenv("HOME")+"/.pocket",
		"--output=json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

func (m model) updateApplicationDetails(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = stateTable
	}
	return m, nil
}

func (m model) renderApplicationDetails() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")). // Light grey-green
		Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("65")). // Muted green for border
		Padding(0, 1).
		Width(m.width - 4)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")). // Soft grey-green
		Padding(1, 2).
		Width(m.width - 4)

	if m.detailsLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")). // Bold yellow
			Bold(true).
			Align(lipgloss.Center).
			Width(m.width)
		return loadingStyle.Render("🔄 Loading application details...")
	}

	// Header with address
	headerText := fmt.Sprintf("📮 APPLICATION DETAILS - %s", m.selectedAppAddress)
	header := headerStyle.Render(headerText)

	// Application details section
	appDetailsHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")).
		Bold(true).
		Render("ℹ️ Application Information:")

	// Pretty print the JSON for application details
	prettyAppDetails := m.prettyPrintJSON(m.applicationDetails)
	appDetailsContent := contentStyle.Render(prettyAppDetails)

	// Bank balances section
	bankHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("150")).
		Bold(true).
		Render("💰 BANK BALANCES")

	bankContent := contentStyle.Render(m.bankBalances)

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")).
		Italic(true).
		Align(lipgloss.Center).
		Width(m.width).
		Render("Press ESC to return to main view")

	content := header + "\n\n" +
		appDetailsHeader + "\n" + appDetailsContent + "\n\n" +
		bankHeader + "\n" + bankContent + "\n\n" +
		instructions

	return content
}

func (m model) prettyPrintJSON(jsonStr string) string {
	if jsonStr == "" {
		return "No data available"
	}

	// Try to parse and reformat the JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		// If parsing fails, return the original string
		return jsonStr
	}

	// Marshal with indentation for pretty printing
	prettyBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		// If pretty printing fails, return the original string
		return jsonStr
	}

	return string(prettyBytes)
}

func (m model) handleFundCommand(cmd string) (model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) < 3 {
		m.err = fmt.Errorf("usage: f <address> <amount> or fund <address> <amount>")
		return m, nil
	}

	address := parts[1]
	amountStr := parts[2]

	// Validate amount is numeric
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		m.err = fmt.Errorf("amount must be a positive integer: %s", amountStr)
		return m, nil
	}

	// Execute fund in background
	return m, m.executeFund(address, amount)
}

func (m model) executeFund(address string, amount int64) tea.Cmd {
	return func() tea.Msg {
		txHash, err := fundApplication(address, amount, m.config, m.currentNetwork)
		if err != nil {
			return fmt.Sprintf("Fund failed: %v", err)
		}
		return fundCompletedMsg{txHash: txHash}
	}
}

func fundApplication(address string, amount int64, config *Config, networkName string) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config not loaded")
	}

	network, exists := config.Config.Networks[networkName]
	if !exists {
		return "", fmt.Errorf("network not found: %s", networkName)
	}

	// Validate bank address is configured
	if network.Bank == "" {
		return "", fmt.Errorf("bank address not configured for network: %s", networkName)
	}

	// Determine chain ID and node based on network
	var chainID, node string
	switch networkName {
	case "pocket":
		chainID = "pocket"
		node = "https://shannon-grove-rpc.mainnet.poktroll.com"
	case "pocket-beta":
		chainID = "pocket-beta"
		node = "https://shannon-testnet-grove-rpc.beta.poktroll.com"
	default:
		return "", fmt.Errorf("unsupported network: %s", networkName)
	}

	// Execute pocketd bank send command
	amountWithDenom := fmt.Sprintf("%dupokt", amount)
	cmd := exec.Command("pocketd", "tx", "bank", "send",
		network.Bank,
		address,
		amountWithDenom,
		"--node="+node,
		"--chain-id="+chainID,
		"--fees=20000upokt",
		"--home="+os.Getenv("HOME")+"/.pocket",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pocketd command failed: %v, output: %s", err, string(output))
	}

	// Parse transaction hash from output
	outputStr := string(output)
	txHash := ""

	// Look for txhash in the output (pocketd outputs in various formats)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Handle formats: "txhash: ABC123", "txhash:ABC123", or just "ABC123" on its own
		if strings.HasPrefix(strings.ToLower(line), "txhash:") {
			txHash = strings.TrimSpace(strings.TrimPrefix(line, "txhash:"))
			txHash = strings.TrimSpace(strings.TrimPrefix(txHash, " "))
			break
		} else if len(line) == 64 && isHexString(line) {
			// Likely a 64-character hex hash
			txHash = line
			break
		}
	}

	return txHash, nil
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
