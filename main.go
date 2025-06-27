package main

import (
	"fmt"
	"io/ioutil"
	"log"
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
		m.loading = false // clear loading state
		
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
			
		case stateNetworkSelect:
			return m.updateNetworkSelect(msg)
			
		case stateGatewaySelect:
			return m.updateGatewaySelect(msg)
			
		case stateHelp:
			return m.updateHelp(msg)
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
	
	// Render command area
	commandArea := m.renderCommandArea()
	
	// Combine main content and command area
	result := strings.Join(mainContentLines, "\n") + "\n" + commandArea
	
	return result
}

func (m model) renderCommandArea() string {
	// Create dedicated command area at bottom
	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("65")) // Muted green
	
	commandStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")). // Black background
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
		commandContent = "Command: :" + m.commandInput
	case stateSearch:
		commandContent = "Search: /" + m.searchInput
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
			headerLines := 8 // Approximate header size
			commandLines := 3 // Approximate command prompt size
			
			if len(lines) > headerLines + commandLines {
				// Keep header and command prompt, trim table content
				tableTrimCount := len(lines) - m.height
				tableStartIdx := headerLines
				tableEndIdx := len(lines) - commandLines
				
				// Remove excess table lines
				if tableTrimCount > 0 && tableEndIdx > tableStartIdx {
					trimFromTable := min(tableTrimCount, tableEndIdx - tableStartIdx)
					newLines := make([]string, 0, len(lines) - trimFromTable)
					newLines = append(newLines, lines[:tableStartIdx]...)
					newLines = append(newLines, lines[tableStartIdx + trimFromTable:]...)
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
		Background(lipgloss.Color("0")). // Black background
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
		Width(m.width/3) // 33% for state

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")). // Soft grey-green
		Width(m.width*2/3 - 2) // 67% for commands

	// Column 1: App State
	appCount := len(m.applications)
	stateContent := fmt.Sprintf("🌐 Network: %s\n🧱 Gateway: %s\n📱 Applications: %d", 
		strings.ToUpper(m.currentNetwork), m.currentGateway, appCount)
	stateColumn := stateStyle.Render(stateContent)

	// Column 2: Commands (clean columns)
	commandContent := "Navigation:           Sort Columns:        Actions:\n"
	commandContent += "r: Refresh           :ss Status           :: Command\n" 
	commandContent += "n: Network           :sa Address          h: Help\n"
	commandContent += "g: Gateway           :sp Stake            q: Quit\n"
	commandContent += "/: Search            :sv Service          asc/desc: Direction"
	commandColumn := commandStyle.Render(commandContent)

	// Join 2 columns horizontally
	headerContent := lipgloss.JoinHorizontal(lipgloss.Top, stateColumn, commandColumn)

	return headerBoxStyle.Render(headerContent)
}

func (m model) renderTableContent() string {
	// Soft grey-green color scheme for table
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark grey background
		Foreground(lipgloss.Color("150")) // Light grey-green text

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
	stakeWidth := 12 // Reduced from 15 (fits "99999.99" comfortably)
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
				Foreground(lipgloss.Color("160"))   // Red text
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
		Background(lipgloss.Color("0")). // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("65")). // Muted green border
		Width(m.width).
		Padding(0, 1)

	cmdLine := cmdLineStyle.Render(":"+m.commandInput)
	
	return header + "\n" + tableContent + "\n" + cmdLine
}

func (m model) renderSearchMode() string {
	// Render table with reduced height to make room for search line
	header := m.renderHeader()
	tableContent := m.renderTableContent()
	
	// Create search line
	searchLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")). // Black background
		Foreground(lipgloss.Color("150")). // Light grey-green
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("108")). // Soft grey-green for search
		Width(m.width).
		Padding(0, 1)

	searchLine := searchLineStyle.Render("/"+m.searchInput)
	
	return header + "\n" + tableContent + "\n" + searchLine
}

func (m model) renderNetworkSelect() string {
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")). // Black background
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
		Background(lipgloss.Color("0")). // Black background
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
  
COMMANDS (prefix with :):
  q, quit         Quit application
  h, help         Show this help
  n, network      Switch network
  g, gateway      Switch gateway
  
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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
