package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pangolin brand colors
const (
	primaryOrange = "#F97317"
	orangeLight   = "#FFA500"
	orangeDark    = "#FF4500"
	orangeGold    = "#FFD700"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(primaryOrange)).
			Bold(true).
			Align(lipgloss.Center).
			Margin(1, 0)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(orangeLight)).
			Italic(true).
			Align(lipgloss.Center).
			Margin(0, 0, 1, 0)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(primaryOrange)).
			Padding(1, 2).
			Margin(1, 0)

	focusedButtonStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(primaryOrange)).
				Foreground(lipgloss.Color("#FFFFFF")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(primaryOrange)).
				Padding(0, 3).
				Margin(0, 1).
				Bold(true)

	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Foreground(lipgloss.Color("#666666")).
			Padding(0, 3).
			Margin(0, 1)

	inputStyle = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color(primaryOrange))

	focusedInputStyle = lipgloss.NewStyle().
				BorderForeground(lipgloss.Color(orangeGold)).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			Margin(1, 0)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00")).
			Bold(true).
			Margin(1, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Margin(1, 0)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(orangeLight)).
			Margin(1, 0)
)

// Key bindings
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev field"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Tab, k.ShiftTab, k.Enter, k.Back},
		{k.Help, k.Quit},
	}
}

// Screen types with proper branching
type screenID string

const (
	welcomeScreen           screenID = "welcome"
	hybridModeScreen        screenID = "hybrid_mode"
	hybridCredentialsScreen screenID = "hybrid_credentials"
	domainConfigScreen      screenID = "domain_config"
	emailConfigScreen       screenID = "email_config"
	emailInputScreen        screenID = "email_input"
	advancedConfigScreen    screenID = "advanced_config"
	containerScreen         screenID = "container"
	installContainersScreen screenID = "install_containers"
	installScreen           screenID = "install"
	crowdsecScreen          screenID = "crowdsec"
	crowdsecManageScreen    screenID = "crowdsec_manage"
	crowdsecInstallScreen   screenID = "crowdsec_install"
	setupTokenScreen        screenID = "setup_token"
	completeScreen          screenID = "complete"
)

// Form field types
type fieldType int

const (
	fieldButton fieldType = iota
	fieldInput
)

type field struct {
	fieldType fieldType
	label     string
	input     textinput.Model
	required  bool
}

// Screen definition for better navigation
type screenDef struct {
	id         screenID
	title      string
	question   string
	fieldTypes []fieldType
	labels     []string
	required   []bool
	nextScreen func(m model) screenID
	prevScreen screenID
}

// Navigation logic
func getNextScreen(currentScreen screenID, choice int, config Config) screenID {
	switch currentScreen {
	case welcomeScreen:
		return hybridModeScreen
	case hybridModeScreen:
		if choice == 0 { // Yes - hybrid mode
			return hybridCredentialsScreen
		} else { // No - standard mode
			return domainConfigScreen
		}
	case hybridCredentialsScreen:
		return domainConfigScreen
	case domainConfigScreen:
		if config.HybridMode {
			return advancedConfigScreen // Skip email for hybrid
		} else {
			return emailConfigScreen
		}
	case emailConfigScreen:
		if choice == 0 { // Yes - enable email
			return emailInputScreen
		} else { // No - skip email
			return advancedConfigScreen
		}
	case emailInputScreen:
		return advancedConfigScreen
	case advancedConfigScreen:
		return containerScreen
	case containerScreen:
		return installContainersScreen
	case installContainersScreen:
		// This screen should not be used in getNextScreen
		// The user stays on this screen until they make a choice
		return installScreen
	case installScreen:
		// After installation, check crowdsec
		if !config.HybridMode {
			return crowdsecScreen
		}
		return setupTokenScreen
	case crowdsecScreen:
		if choice == 0 { // Yes to crowdsec
			return crowdsecManageScreen
		}
		// No to crowdsec
		return setupTokenScreen
	case crowdsecManageScreen:
		if config.DoCrowdsecInstall {
			return crowdsecInstallScreen // Install crowdsec
		}
		return setupTokenScreen // Skip to setup token if CrowdSec disabled
	case crowdsecInstallScreen:
		return setupTokenScreen // After CrowdSec installation, go to setup token
	case setupTokenScreen:
		return completeScreen
	case completeScreen:
		return completeScreen // Stay here
	default:
		return welcomeScreen
	}
}

func getPrevScreen(currentScreen screenID, config Config) screenID {
	switch currentScreen {
	case welcomeScreen:
		return welcomeScreen // Can't go back
	case hybridModeScreen:
		return welcomeScreen
	case hybridCredentialsScreen:
		return hybridModeScreen
	case domainConfigScreen:
		if config.HybridMode {
			return hybridCredentialsScreen
		} else {
			return hybridModeScreen
		}
	case emailConfigScreen:
		return domainConfigScreen
	case emailInputScreen:
		return emailConfigScreen
	case advancedConfigScreen:
		if config.HybridMode {
			return domainConfigScreen
		} else if config.EnableEmail {
			return emailInputScreen
		} else {
			return emailConfigScreen
		}
	case containerScreen:
		return advancedConfigScreen
	case installContainersScreen:
		return containerScreen
	case installScreen:
		return installContainersScreen
	case crowdsecScreen:
		return installScreen
	case crowdsecManageScreen:
		return crowdsecScreen
	case setupTokenScreen:
		if config.DoCrowdsecInstall {
			return crowdsecManageScreen
		}
		return installScreen
	case completeScreen:
		return completeScreen // Can't go back
	default:
		return welcomeScreen
	}
}

// Main model
type model struct {
	currentScreen screenID
	config        Config
	fields        []field
	focusIndex    int
	err           error
	spinner       spinner.Model
	help          help.Model
	keys          keyMap
	showHelp      bool
	width         int
	height        int
	installing    bool
	lastChoice    int // Store the last button choice for navigation

	// Installation logs
	installLogs  []string
	logsViewport viewport.Model
	installStep  string

	// Port validation
	portWarnings []string
}

// Messages
type installCompleteMsg struct{}
type installErrorMsg struct{ err error }
type installStepMsg struct{ step string }
type installLogMsg struct{ log string }
type installBatchLogsMsg struct{ logs []string }

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(primaryOrange))

	vp := viewport.New(70, 12)
	vp.SetContent("")

	// Initialize empty logs - they will be populated during installation
	var installLogs []string

	// Try to load existing config values
	config := Config{}
	if existingConfig, err := loadExistingConfig(); err == nil {
		config = *existingConfig
	}

	// Check ports if running as root (like original installer)
	var portWarnings []string
	if os.Geteuid() == 0 {
		for _, p := range []int{80, 443} {
			if err := checkPortsAvailable(p); err != nil {
				portWarnings = append(portWarnings, fmt.Sprintf("Port %d: %v", p, err))
			}
		}
	}

	return model{
		currentScreen: welcomeScreen,
		config:        config,
		fields:        []field{},
		spinner:       s,
		help:          help.New(),
		keys:          keys,
		logsViewport:  vp,
		installLogs:   installLogs,
		installStep:   "",
		portWarnings:  portWarnings,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case tea.KeyMsg:
		// Check if we're currently focused on an input field
		focusedOnInput := false
		if m.focusIndex < len(m.fields) && m.fields[m.focusIndex].fieldType == fieldInput {
			focusedOnInput = m.fields[m.focusIndex].input.Focused()
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			// Only toggle help if not typing in an input field
			if !focusedOnInput {
				m.showHelp = !m.showHelp
				return m, nil
			}

		case key.Matches(msg, m.keys.Back):
			// Don't allow going back from install screen - installation is one-way
			if m.currentScreen != installScreen && !focusedOnInput {
				return m.handleBack()
			}

		case key.Matches(msg, m.keys.Enter):
			// Don't allow Enter navigation on install screen - installation is one-way
			if m.currentScreen != installScreen {
				return m.handleEnter()
			}

		case key.Matches(msg, m.keys.Tab):
			// Don't allow navigation on install screen - installation is one-way
			if m.currentScreen != installScreen {
				return m.handleNavigation(1)
			}

		case key.Matches(msg, m.keys.ShiftTab):
			// Don't allow navigation on install screen - installation is one-way
			if m.currentScreen != installScreen {
				return m.handleNavigation(-1)
			}

		case key.Matches(msg, m.keys.Up):
			// Up/down arrows should not handle form navigation
			// Let viewport handle scrolling on install screen
			// Users should use Tab/Shift+Tab for form navigation

		case key.Matches(msg, m.keys.Down):
			// Up/down arrows should not handle form navigation
			// Let viewport handle scrolling on install screen
			// Users should use Tab/Shift+Tab for form navigation

		case key.Matches(msg, m.keys.Left):
			// Allow left/right navigation on button screens, but not install screen
			if m.currentScreen != installScreen && !focusedOnInput && m.hasButtonFields() {
				return m.handleNavigation(-1)
			}

		case key.Matches(msg, m.keys.Right):
			// Allow left/right navigation on button screens, but not install screen
			if m.currentScreen != installScreen && !focusedOnInput && m.hasButtonFields() {
				return m.handleNavigation(1)
			}

		}

	case installStepMsg:
		m.installStep = msg.step
		return m, nil

	case installLogMsg:
		m.installLogs = append(m.installLogs, msg.log)
		// Keep only last 200 log lines
		if len(m.installLogs) > 200 {
			m.installLogs = m.installLogs[1:]
		}
		// Update viewport content
		m.logsViewport.SetContent(strings.Join(m.installLogs, "\n"))
		m.logsViewport.GotoBottom()
		return m, nil

	case installBatchLogsMsg:
		m.installLogs = append(m.installLogs, msg.logs...)
		// Keep only last 200 log lines
		if len(m.installLogs) > 200 {
			start := len(m.installLogs) - 200
			m.installLogs = m.installLogs[start:]
		}
		// Update viewport content
		m.logsViewport.SetContent(strings.Join(m.installLogs, "\n"))
		m.logsViewport.GotoBottom()

		// Check if installation completed
		for _, log := range msg.logs {
			if strings.Contains(log, "🎉 Installation completed successfully!") {
				return m, func() tea.Msg { return installCompleteMsg{} }
			}
			if strings.Contains(log, "Error") {
				return m, func() tea.Msg { return installErrorMsg{err: fmt.Errorf("installation failed")} }
			}
		}
		return m, nil

	case installCompleteMsg:
		m.installing = false
		// Stay on install screen so users can review logs
		// They can press Ctrl+C to exit
		return m, nil

	case installErrorMsg:
		m.installing = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if m.installing {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update focused input
	if m.focusIndex < len(m.fields) && m.fields[m.focusIndex].fieldType == fieldInput {
		m.fields[m.focusIndex].input, cmd = m.fields[m.focusIndex].input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport for installation logs when on install screen
	if m.currentScreen == installScreen {
		m.logsViewport, cmd = m.logsViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) handleNavigation(direction int) (model, tea.Cmd) {
	if len(m.fields) == 0 {
		return m, nil
	}

	// Clear any previous errors
	m.err = nil

	if direction > 0 {
		m.focusIndex = (m.focusIndex + 1) % len(m.fields)
	} else {
		m.focusIndex = (m.focusIndex - 1 + len(m.fields)) % len(m.fields)
	}

	return m.updateFocus(), nil
}

func (m model) updateFocus() model {
	// Update focus for all inputs
	for i := range m.fields {
		if m.fields[i].fieldType == fieldInput {
			if i == m.focusIndex {
				m.fields[i].input.Focus()
			} else {
				m.fields[i].input.Blur()
			}
		}
	}
	return m
}

func (m model) hasButtonFields() bool {
	for _, field := range m.fields {
		if field.fieldType == fieldButton {
			return true
		}
	}
	return false
}

func (m model) handleBack() (model, tea.Cmd) {
	if m.currentScreen == welcomeScreen {
		return m, tea.Quit
	}

	prevScreen := getPrevScreen(m.currentScreen, m.config)
	if prevScreen == m.currentScreen {
		return m, tea.Quit // Can't go back further
	}

	m.currentScreen = prevScreen
	m = m.initScreen()
	return m, nil
}

func (m model) handleEnter() (model, tea.Cmd) {
	if len(m.fields) == 0 {
		return m.nextScreen()
	}

	currentField := m.fields[m.focusIndex]

	switch currentField.fieldType {
	case fieldButton:
		return m.handleButtonPress()
	case fieldInput:
		// Move to next field, or if last input field and there's a button, move to button
		if m.focusIndex < len(m.fields)-1 {
			return m.handleNavigation(1)
		} else {
			// We're on the last field - if it's an input and there's no button, submit
			// Check if there's a button after this input
			hasButton := false
			for i := m.focusIndex + 1; i < len(m.fields); i++ {
				if m.fields[i].fieldType == fieldButton {
					hasButton = true
					break
				}
			}
			if hasButton {
				return m.handleNavigation(1) // Move to the button
			} else {
				return m.handleSubmit() // Submit directly
			}
		}
	}

	return m, nil
}

func (m model) handleButtonPress() (model, tea.Cmd) {
	// Store the choice for navigation
	m.lastChoice = m.focusIndex

	switch m.currentScreen {
	case welcomeScreen:
		return m.nextScreen()

	case hybridModeScreen:
		m.config.HybridMode = (m.focusIndex == 0) // Yes = 0, No = 1
		return m.nextScreen()

	case hybridCredentialsScreen:
		// Store whether user already has credentials
		// For now, we'll handle this in the installation phase
		// In a full implementation, we'd show input fields for ID and secret
		return m.nextScreen()

	case domainConfigScreen:
		// Continue button pressed - validate and proceed
		return m.validateAndSaveDomainConfig()

	case emailConfigScreen:
		m.config.EnableEmail = (m.focusIndex == 0) // Yes = 0, No = 1
		return m.nextScreen()

	case emailInputScreen:
		// Continue button pressed - save email config and proceed
		return m.handleSubmit()

	case advancedConfigScreen:
		m.config.EnableIPv6 = (m.focusIndex == 0) // Yes = 0, No = 1
		return m.nextScreen()

	case containerScreen:
		if m.focusIndex == 0 {
			m.config.InstallationContainerType = Docker
		} else {
			m.config.InstallationContainerType = Podman
		}
		return m.nextScreen()

	case installContainersScreen:
		// Store the choice and create config files
		// This happens regardless of container choice
		return m.createConfigFilesAndProceed()

	case crowdsecScreen:
		m.config.DoCrowdsecInstall = (m.focusIndex == 0) // Yes = 0, No = 1
		return m.nextScreen()

	case crowdsecManageScreen:
		// If they choose No to manage CrowdSec, disable it
		if m.focusIndex == 1 { // No = 1
			m.config.DoCrowdsecInstall = false
		} else {
			// Yes = 0, they want to manage CrowdSec
			m.config.DoCrowdsecInstall = true
		}
		return m.nextScreen()

	case setupTokenScreen:
		return m.nextScreen()
	}

	return m, nil
}

func (m model) handleSubmit() (model, tea.Cmd) {
	switch m.currentScreen {
	case domainConfigScreen:
		return m.validateAndSaveDomainConfig()
	case emailInputScreen:
		// Save email config and proceed
		if len(m.fields) >= 5 {
			m.config.EmailSMTPHost = m.fields[0].input.Value()
			if port, err := strconv.Atoi(m.fields[1].input.Value()); err == nil {
				m.config.EmailSMTPPort = port
			} else {
				m.config.EmailSMTPPort = 587
			}
			m.config.EmailSMTPUser = m.fields[2].input.Value()
			m.config.EmailSMTPPass = m.fields[3].input.Value()
			m.config.EmailNoReply = m.fields[4].input.Value()
		}
		return m.nextScreen()
	default:
		// For other screens, just proceed to next screen
		return m.nextScreen()
	}
}

func (m model) validateAndSaveDomainConfig() (model, tea.Cmd) {
	// Validate required fields
	for _, field := range m.fields {
		if field.required && field.input.Value() == "" {
			m.err = fmt.Errorf("%s is required", field.label)
			return m, nil
		}
	}

	if m.config.HybridMode {
		// For hybrid mode: only dashboard domain (public IP/domain)
		dashboardDomain := m.fields[0].input.Value()
		m.config.DashboardDomain = dashboardDomain
		m.config.InstallGerbil = true
	} else {
		// For standard mode: base domain + subdomain + email
		baseDomain := m.fields[0].input.Value()
		subdomain := m.fields[1].input.Value()
		email := m.fields[2].input.Value()

		// Default subdomain if empty
		if subdomain == "" {
			subdomain = "pangolin"
		}

		// Combine subdomain + base domain
		dashboardDomain := subdomain + "." + baseDomain

		m.config.BaseDomain = baseDomain
		m.config.DashboardDomain = dashboardDomain
		m.config.LetsEncryptEmail = email
		m.config.InstallGerbil = true
	}

	return m.nextScreen()
}

func (m model) nextScreen() (model, tea.Cmd) {
	nextScreen := getNextScreen(m.currentScreen, m.lastChoice, m.config)

	// Handle special cases that need to happen before screen transition
	if m.currentScreen == emailInputScreen {
		// Save email config if we have input fields
		if len(m.fields) >= 5 {
			m.config.EmailSMTPHost = m.fields[0].input.Value()
			if port, err := strconv.Atoi(m.fields[1].input.Value()); err == nil {
				m.config.EmailSMTPPort = port
			} else {
				m.config.EmailSMTPPort = 587
			}
			m.config.EmailSMTPUser = m.fields[2].input.Value()
			m.config.EmailSMTPPass = m.fields[3].input.Value()
			m.config.EmailNoReply = m.fields[4].input.Value()
		}
	}

	m.currentScreen = nextScreen

	// Handle special screen transitions
	if nextScreen == installScreen {
		return m.startInstallation()
	}
	if nextScreen == crowdsecInstallScreen {
		return m.startCrowdsecInstallation()
	}
	if nextScreen == completeScreen && m.currentScreen == completeScreen {
		return m, tea.Quit
	}

	// Handle transition TO installContainersScreen
	if nextScreen == installContainersScreen {
		// Just show the screen, don't start config creation yet
		m.currentScreen = nextScreen
		return m.initScreen(), nil
	}

	// Handle config creation when transitioning from installContainersScreen
	// This should only happen when the user has made a choice, not when first showing the screen

	// Handle config creation completion
	if m.currentScreen == installContainersScreen && m.installing == false {
		// Config files have been created, now route based on container choice
		if m.lastChoice == 0 { // Yes to containers
			return m.startInstallation()
		} else { // No to containers - go to CrowdSec
			if !m.config.HybridMode {
				m.currentScreen = crowdsecScreen
				return m.initScreen(), nil
			}
			m.currentScreen = setupTokenScreen
			return m.initScreen(), nil
		}
	}

	m = m.initScreen()
	return m, nil
}

func (m model) initScreen() model {
	m.fields = []field{}
	m.focusIndex = 0
	m.err = nil

	switch m.currentScreen {
	case welcomeScreen:
		// No fields needed for welcome screen

	case hybridModeScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}
		// Set focus based on existing config
		if m.config.HybridMode {
			m.focusIndex = 0 // Yes
		} else {
			m.focusIndex = 1 // No
		}

	case hybridCredentialsScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}

	case domainConfigScreen:
		if m.config.HybridMode {
			// For hybrid mode: only dashboard domain (public IP/domain) - no base domain or email
			m.fields = []field{
				m.createInputField("Public IP or Domain", true),
				m.createButtonField("Continue"),
			}

			// Set placeholder and prefill with existing value or public IP
			m.fields[0].input.Placeholder = "203.0.113.1 or myserver.example.com"
			if m.config.DashboardDomain != "" {
				m.fields[0].input.SetValue(m.config.DashboardDomain)
			} else if publicIP := getPublicIP(); publicIP != "" {
				m.fields[0].input.SetValue(publicIP)
			}
		} else {
			// For standard mode: base domain + dashboard subdomain + email
			m.fields = []field{
				m.createInputField("Base Domain", true),
				m.createInputField("Dashboard Subdomain", false),
				m.createInputField("Let's Encrypt Email", true),
				m.createButtonField("Continue"),
			}

			// Set placeholders
			m.fields[0].input.Placeholder = "example.com"
			m.fields[1].input.Placeholder = "pangolin"
			m.fields[2].input.Placeholder = "admin@example.com"

			// Prefill with existing values if available
			if m.config.BaseDomain != "" {
				m.fields[0].input.SetValue(m.config.BaseDomain)
			}

			// Extract subdomain from dashboard domain if possible
			if m.config.DashboardDomain != "" && m.config.BaseDomain != "" {
				subdomain := strings.TrimSuffix(m.config.DashboardDomain, "."+m.config.BaseDomain)
				if subdomain != m.config.DashboardDomain { // Only set if it's actually a subdomain
					m.fields[1].input.SetValue(subdomain)
				}
			}

			if m.config.LetsEncryptEmail != "" {
				m.fields[2].input.SetValue(m.config.LetsEncryptEmail)
			}
		}

	case emailConfigScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}
		// Set focus based on existing config
		if m.config.EnableEmail {
			m.focusIndex = 0 // Yes
		} else {
			m.focusIndex = 1 // No
		}

	case emailInputScreen:
		m.fields = []field{
			m.createInputField("SMTP Host", false),
			m.createInputField("SMTP Port", false),
			m.createInputField("SMTP Username", false),
			m.createInputField("SMTP Password", false),
			m.createInputField("No-Reply Email", false),
			m.createButtonField("Continue"),
		}

		// Set password field mode
		m.fields[3].input.EchoMode = textinput.EchoPassword
		m.fields[3].input.EchoCharacter = '•'

		// Prefill with existing values or defaults
		if m.config.EmailSMTPHost != "" {
			m.fields[0].input.SetValue(m.config.EmailSMTPHost)
		}

		if m.config.EmailSMTPPort > 0 {
			m.fields[1].input.SetValue(fmt.Sprintf("%d", m.config.EmailSMTPPort))
		} else {
			m.fields[1].input.SetValue("587")
		}

		if m.config.EmailSMTPUser != "" {
			m.fields[2].input.SetValue(m.config.EmailSMTPUser)
		}

		if m.config.EmailSMTPPass != "" {
			m.fields[3].input.SetValue(m.config.EmailSMTPPass)
		}

		if m.config.EmailNoReply != "" {
			m.fields[4].input.SetValue(m.config.EmailNoReply)
		}

	case advancedConfigScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}

	case containerScreen:
		m.fields = []field{
			m.createButtonField("Docker"),
			m.createButtonField("Podman"),
		}
		// Set focus based on existing config
		if m.config.InstallationContainerType == Podman {
			m.focusIndex = 1 // Podman
		} else {
			m.focusIndex = 0 // Docker (default)
		}

	case installContainersScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}

	case crowdsecScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}

	case crowdsecManageScreen:
		m.fields = []field{
			m.createButtonField("Yes"),
			m.createButtonField("No"),
		}

	case crowdsecInstallScreen:
		// No fields needed - this will be automatic installation
		// The installation will be started in the Update function when this screen is reached

	case setupTokenScreen:
		// No fields needed - this will be automatic
	}

	return m.updateFocus()
}

func (m model) createInputField(label string, required bool) field {
	input := textinput.New()
	input.Width = 40
	input.CharLimit = 100

	return field{
		fieldType: fieldInput,
		label:     label,
		input:     input,
		required:  required,
	}
}

func (m model) createButtonField(label string) field {
	return field{
		fieldType: fieldButton,
		label:     label,
	}
}

func (m model) startInstallation() (model, tea.Cmd) {
	m.installing = true

	return m, tea.Batch(
		m.spinner.Tick,
		m.performInstallationAsync(),
	)
}

func (m model) startCrowdsecInstallation() (model, tea.Cmd) {
	m.installing = true

	return m, tea.Batch(
		m.spinner.Tick,
		m.performCrowdsecInstallationAsync(),
	)
}

func (m model) createConfigFilesOnly() (model, tea.Cmd) {
	m.installing = true

	return m, tea.Batch(
		m.spinner.Tick,
		m.performConfigCreationAsync(),
	)
}

func (m model) createConfigFilesAndProceed() (model, tea.Cmd) {
	m.installing = true

	return m, tea.Batch(
		m.spinner.Tick,
		m.performConfigCreationAndProceedAsync(),
	)
}

func (m model) performInstallationAsync() tea.Cmd {
	return tea.Sequence(
		// Step 1: Load configuration
		func() tea.Msg {
			loadVersions(&m.config)
			m.config.Secret = generateRandomSecretKey()
			return installLogMsg{log: "✓ Configuration loaded and secret generated"}
		},

		// Step 3: Create config files
		func() tea.Msg {
			if err := createConfigFiles(m.config); err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Config error: %v", err)}
			}
			moveFile("config/docker-compose.yml", "docker-compose.yml")
			return installLogMsg{log: "✓ Configuration files created"}
		},

		// Step 4: Test container runtime and install Docker if needed
		func() tea.Msg {
			if m.config.InstallationContainerType == Podman {
				cmd := exec.Command("podman-compose", "--version")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return installLogMsg{log: fmt.Sprintf("❌ podman-compose error: %v", err)}
				}
				return installLogMsg{log: fmt.Sprintf("✓ podman-compose available: %s", strings.TrimSpace(string(output)))}
			} else {
				// Test Docker compose
				cmd := exec.Command("docker", "compose", "version")
				output, err := cmd.CombinedOutput()
				if err != nil {
					// Try legacy docker-compose
					cmd = exec.Command("docker-compose", "--version")
					output, err = cmd.CombinedOutput()
					if err != nil {
						return installLogMsg{log: fmt.Sprintf("❌ docker compose error: %v", err)}
					}
					return installLogMsg{log: fmt.Sprintf("✓ docker-compose available: %s", strings.TrimSpace(string(output)))}
				}
				return installLogMsg{log: fmt.Sprintf("✓ docker compose available: %s", strings.TrimSpace(string(output)))}
			}
		},
		// Step 4.5: Install Docker if needed (Linux only)
		func() tea.Msg {
			if m.config.InstallationContainerType == Docker && !isDockerInstalled() && runtime.GOOS == "linux" {
				return installLogMsg{log: "🐳 Docker not installed, installing Docker..."}
			}
			return installLogMsg{log: "✓ Container runtime ready"}
		},
		func() tea.Msg {
			if m.config.InstallationContainerType == Docker && !isDockerInstalled() && runtime.GOOS == "linux" {
				installDocker()
				// Try to start docker service
				if err := startDockerService(); err != nil {
					return installLogMsg{log: fmt.Sprintf("⚠️ Docker service start error: %v", err)}
				}
				return installLogMsg{log: "✓ Docker service started"}
			}
			return installLogMsg{log: "✓ Container runtime ready"}
		},
		func() tea.Msg {
			if m.config.InstallationContainerType == Docker && !isDockerInstalled() && runtime.GOOS == "linux" {
				// Wait for Docker to start
				return installLogMsg{log: "⏳ Waiting for Docker to start..."}
			}
			return installLogMsg{log: "✓ Container runtime ready"}
		},
		func() tea.Msg {
			if m.config.InstallationContainerType == Docker && !isDockerInstalled() && runtime.GOOS == "linux" {
				// Check if Docker is running
				for i := 0; i < 5; i++ {
					if isDockerRunning() {
						return installLogMsg{log: "✓ Docker is running!"}
					}
					time.Sleep(2 * time.Second)
				}
				return installLogMsg{log: "⚠️ Docker may not be running yet"}
			}
			return installLogMsg{log: "✓ Container runtime ready"}
		},

		// Step 5: Check docker-compose.yml
		func() tea.Msg {
			if _, err := exec.Command("ls", "-la", "docker-compose.yml").CombinedOutput(); err != nil {
				return installLogMsg{log: "❌ docker-compose.yml not found"}
			}
			return installLogMsg{log: "✓ docker-compose.yml found"}
		},

		// Step 6: Pull container images
		func() tea.Msg {
			if m.config.InstallationContainerType == Podman {
				return installLogMsg{log: "🚢 Pulling container images with podman-compose..."}
			} else {
				return installLogMsg{log: "🚢 Pulling container images with docker compose..."}
			}
		},
		func() tea.Msg {
			var cmd *exec.Cmd

			if m.config.InstallationContainerType == Podman {
				cmd = exec.Command("podman-compose", "-f", "docker-compose.yml", "pull")
			} else {
				// Try docker compose first, then fallback to docker-compose
				cmd = exec.Command("docker", "compose", "-f", "docker-compose.yml", "pull")
				if err := exec.Command("docker", "compose", "version").Run(); err != nil {
					cmd = exec.Command("docker-compose", "-f", "docker-compose.yml", "pull")
				}
			}

			output, err := cmd.CombinedOutput()

			result := strings.TrimSpace(string(output))
			if result == "" {
				result = "(no output)"
			}

			if err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Image pull failed: %v\nOutput: %s", err, result)}
			} else {
				return installLogMsg{log: fmt.Sprintf("✅ Images pulled successfully\nOutput: %s", result)}
			}
		},

		// Step 7: Start containers
		func() tea.Msg {
			if m.config.InstallationContainerType == Podman {
				return installLogMsg{log: "🚀 Starting containers with podman-compose..."}
			} else {
				return installLogMsg{log: "🚀 Starting containers with docker compose..."}
			}
		},
		func() tea.Msg {
			var cmd *exec.Cmd

			if m.config.InstallationContainerType == Podman {
				cmd = exec.Command("podman-compose", "-f", "docker-compose.yml", "up", "-d")
			} else {
				// Try docker compose first, then fallback to docker-compose
				cmd = exec.Command("docker", "compose", "-f", "docker-compose.yml", "up", "-d")
				if err := exec.Command("docker", "compose", "version").Run(); err != nil {
					cmd = exec.Command("docker-compose", "-f", "docker-compose.yml", "up", "-d")
				}
			}

			output, err := cmd.CombinedOutput()

			result := strings.TrimSpace(string(output))
			if result == "" {
				result = "(no output)"
			}

			if err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Container start failed: %v\nOutput: %s", err, result)}
			} else {
				return installLogMsg{log: fmt.Sprintf("✅ Containers started successfully\nOutput: %s", result)}
			}
		},

		// Step 8: Complete initial installation
		func() tea.Msg {
			return installLogMsg{log: "🎉 Installation process completed!"}
		},
		func() tea.Msg {
			return installLogMsg{log: ""}
		},
		func() tea.Msg {
			return installLogMsg{log: "📋 Review the logs above for any errors or warnings."}
		},
		func() tea.Msg {
			return installLogMsg{log: "🔍 Use ↑↓ arrows to scroll through the installation output."}
		},
		func() tea.Msg {
			return installLogMsg{log: "✨ Press Ctrl+C to exit when you're ready."}
		},
	)
}

func (m model) performConfigCreationAsync() tea.Cmd {
	return tea.Sequence(
		// Step 1: Load configuration
		func() tea.Msg {
			loadVersions(&m.config)
			m.config.Secret = generateRandomSecretKey()
			return installLogMsg{log: "✓ Configuration loaded and secret generated"}
		},

		// Step 2: Create config files
		func() tea.Msg {
			if err := createConfigFiles(m.config); err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Config error: %v", err)}
			}
			moveFile("config/docker-compose.yml", "docker-compose.yml")
			return installLogMsg{log: "✓ Configuration files created"}
		},

		// Step 3: Complete
		func() tea.Msg {
			return installLogMsg{log: "🎉 Configuration files created successfully!"}
		},
		func() tea.Msg {
			return installLogMsg{log: ""}
		},
		func() tea.Msg {
			return installLogMsg{log: "📋 You can now install containers manually or proceed to CrowdSec setup."}
		},
		func() tea.Msg {
			return installLogMsg{log: "✨ Press Ctrl+C to exit when you're ready."}
		},
		func() tea.Msg {
			return installCompleteMsg{}
		},
	)
}

func (m model) performConfigCreationAndProceedAsync() tea.Cmd {
	return tea.Sequence(
		// Step 1: Load configuration
		func() tea.Msg {
			loadVersions(&m.config)
			m.config.Secret = generateRandomSecretKey()
			return installLogMsg{log: "✓ Configuration loaded and secret generated"}
		},

		// Step 2: Create config files
		func() tea.Msg {
			if err := createConfigFiles(m.config); err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Config error: %v", err)}
			}
			moveFile("config/docker-compose.yml", "docker-compose.yml")
			return installLogMsg{log: "✓ Configuration files created"}
		},

		// Step 3: Check if containers should be installed
		func() tea.Msg {
			if m.lastChoice == 0 { // Yes to containers
				return installLogMsg{log: "🚀 Proceeding with container installation..."}
			} else {
				return installLogMsg{log: "⏭️ Skipping container installation, proceeding to CrowdSec setup..."}
			}
		},
		func() tea.Msg {
			return installCompleteMsg{}
		},
	)
}

func (m model) performCrowdsecInstallationAsync() tea.Cmd {
	return tea.Sequence(
		// Step 1: Stop existing containers
		func() tea.Msg { return installLogMsg{log: "🛑 Stopping existing containers..."} },
		func() tea.Msg {
			if err := stopContainers(m.config.InstallationContainerType); err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Failed to stop containers: %v", err)}
			}
			return installLogMsg{log: "✓ Containers stopped successfully"}
		},

		// Step 2: Backup config
		func() tea.Msg { return installLogMsg{log: "💾 Backing up configuration..."} },
		func() tea.Msg {
			if err := backupConfig(); err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ Backup failed: %v", err)}
			}
			return installLogMsg{log: "✓ Configuration backed up"}
		},

		// Step 3: Install CrowdSec
		func() tea.Msg { return installLogMsg{log: "🛡️ Installing CrowdSec security solution..."} },
		func() tea.Msg {
			if err := installCrowdsec(m.config); err != nil {
				return installLogMsg{log: fmt.Sprintf("❌ CrowdSec installation failed: %v", err)}
			}
			return installLogMsg{log: "✅ CrowdSec installed successfully!"}
		},

		// Step 4: Complete
		func() tea.Msg { return installLogMsg{log: "🎉 CrowdSec installation completed!"} },
		func() tea.Msg { return installLogMsg{log: ""} },
		func() tea.Msg { return installLogMsg{log: "📋 Review the logs above for any errors or warnings."} },
		func() tea.Msg {
			return installLogMsg{log: "🔍 Use ↑↓ arrows to scroll through the installation output."}
		},
		func() tea.Msg { return installLogMsg{log: "✨ Press Ctrl+C to exit when you're ready."} },
		func() tea.Msg { return installCompleteMsg{} },
	)
}

func (m model) performSetupToken() tea.Msg {
	// Only generate setup token for non-hybrid mode
	if !m.config.HybridMode {
		// Check if containers were started during this installation
		containersStarted := false
		if (isDockerInstalled() && m.config.InstallationContainerType == Docker) ||
			(isPodmanInstalled() && m.config.InstallationContainerType == Podman) {
			containersStarted = true
		}

		if containersStarted {
			// Wait for services to be ready
			time.Sleep(5 * time.Second)

			// Generate setup token (implementation would call the setup token functions)
			// This is where you'd call the setup token generation logic from main.go
		}
	}

	return installCompleteMsg{}
}

func (m model) View() string {
	if m.showHelp {
		return m.help.View(m.keys)
	}

	var content string

	switch m.currentScreen {
	case welcomeScreen:
		content = m.welcomeView()
	case hybridModeScreen:
		content = m.hybridModeView()
	case hybridCredentialsScreen:
		content = m.hybridCredentialsView()
	case domainConfigScreen:
		content = m.domainConfigView()
	case emailConfigScreen:
		content = m.emailConfigView()
	case emailInputScreen:
		content = m.emailInputView()
	case advancedConfigScreen:
		content = m.advancedConfigView()
	case containerScreen:
		content = m.containerView()
	case installContainersScreen:
		content = m.installContainersView()
	case installScreen:
		content = m.installView()
	case crowdsecScreen:
		content = m.crowdsecView()
	case crowdsecManageScreen:
		content = m.crowdsecManageView()
	case crowdsecInstallScreen:
		content = m.crowdsecInstallView()
	case setupTokenScreen:
		content = m.setupTokenView()
	case completeScreen:
		content = m.completeView()
	}

	// Add error message if present
	if m.err != nil {
		content += "\n" + errorStyle.Render("Error: "+m.err.Error())
	}

	// Add help info
	content += "\n" + helpStyle.Render(m.help.View(m.keys))

	return content
}

func (m model) welcomeView() string {
	title := titleStyle.Render("🦎 Pangolin Installer")
	subtitle := subtitleStyle.Render("Secure gateway to your private networks")

	var welcomeText string

	// Check if config already exists
	if _, err := os.Stat("config/config.yml"); err == nil {
		welcomeText = `🔄 Existing Installation Detected!

It looks like you already have Pangolin configured.
Your existing configuration values have been loaded.

You can review and update your settings, or proceed directly to installation.

Press Enter to continue...`
	} else {
		welcomeText = `Welcome to the Pangolin installer!

This installer will help you set up Pangolin on your server.

Prerequisites:
• Open TCP ports 80 and 443
• Open UDP ports 51820 and 21820  
• Point your domain to this server's IP

Press Enter to continue...`
	}

	// Add port warnings if any
	if len(m.portWarnings) > 0 {
		welcomeText += "\n\n⚠️  Port Warnings:\n"
		for _, warning := range m.portWarnings {
			welcomeText += "• " + warning + "\n"
		}
		welcomeText += "\nPlease close any services on ports 80/443 before proceeding."
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		boxStyle.Render(welcomeText),
	)
}

func (m model) hybridModeView() string {
	title := titleStyle.Render("Installation Mode")
	question := "Do you want to install Pangolin as a cloud-managed (beta) node?"

	buttons := m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(question),
		buttons,
	)
}

func (m model) hybridCredentialsView() string {
	title := titleStyle.Render("Hybrid Credentials")
	question := "Do you already have credentials from the dashboard?\nIf not, we will create them later."

	buttons := m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(question),
		buttons,
	)
}

func (m model) domainConfigView() string {
	title := titleStyle.Render("Domain Configuration")

	var inputs []string
	for i, field := range m.fields {
		if field.fieldType == fieldInput {
			label := field.label
			if field.required {
				label += " *"
			}

			var inputStr string
			if i == m.focusIndex {
				inputStr = focusedInputStyle.Render(field.input.View())
			} else {
				inputStr = inputStyle.Render(field.input.View())
			}

			inputs = append(inputs, label+"\n"+inputStr)
		}
	}

	content := strings.Join(inputs, "\n\n")

	// Show live preview for standard mode
	if !m.config.HybridMode && len(m.fields) >= 2 {
		baseDomain := m.fields[0].input.Value()
		subdomain := m.fields[1].input.Value()

		// Use default subdomain if empty
		if subdomain == "" {
			subdomain = "pangolin"
		}

		if baseDomain != "" {
			fullDomain := subdomain + "." + baseDomain
			previewStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(orangeLight)).
				Bold(true).
				Margin(1, 0).
				Padding(0, 1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(orangeLight))

			preview := previewStyle.Render("📍 Dashboard URL: https://" + fullDomain)
			content += "\n\n" + preview
		}
	}

	// Add continue button
	if len(m.fields) > 0 && m.fields[len(m.fields)-1].fieldType == fieldButton {
		var button string
		if m.focusIndex == len(m.fields)-1 {
			button = focusedButtonStyle.Render("Continue")
		} else {
			button = buttonStyle.Render("Continue")
		}
		content += "\n\n" + button
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) emailConfigView() string {
	title := titleStyle.Render("Email Configuration")
	question := "Enable email functionality (SMTP)?"
	buttons := m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(question),
		buttons,
	)
}

func (m model) emailInputView() string {
	title := titleStyle.Render("Email Configuration")

	var inputs []string
	for i, field := range m.fields {
		if field.fieldType == fieldInput {
			label := field.label
			if field.required {
				label += " *"
			}

			var inputStr string
			if i == m.focusIndex {
				inputStr = focusedInputStyle.Render(field.input.View())
			} else {
				inputStr = inputStyle.Render(field.input.View())
			}

			inputs = append(inputs, label+"\n"+inputStr)
		}
	}

	content := strings.Join(inputs, "\n\n")

	// Add continue button
	if len(m.fields) > 0 && m.fields[len(m.fields)-1].fieldType == fieldButton {
		var button string
		if m.focusIndex == len(m.fields)-1 {
			button = focusedButtonStyle.Render("Continue")
		} else {
			button = buttonStyle.Render("Continue")
		}
		content += "\n\n" + button
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) advancedConfigView() string {
	title := titleStyle.Render("Advanced Configuration")
	question := "Is your server IPv6 capable?"
	buttons := m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(question),
		buttons,
	)
}

func (m model) containerView() string {
	title := titleStyle.Render("Container Runtime")
	question := "Which container runtime would you like to use?"
	buttons := m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(question),
		buttons,
	)
}

func (m model) installView() string {
	title := titleStyle.Render("🦎 Installing Pangolin")

	var content string

	if m.installing {
		// Show current step with animation
		step := m.installStep
		if step == "" {
			step = "Preparing installation"
		}

		content = fmt.Sprintf("%s %s", m.spinner.View(), step)

		// Always show logs viewport during installation
		logsTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(primaryOrange)).
			Bold(true).
			Render("📋 Installation Logs")

		logsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(primaryOrange)).
			Padding(0, 1).
			Width(74).
			Height(14)

		m.logsViewport.Width = 72
		m.logsViewport.Height = 12

		// Update viewport content with current logs
		if len(m.installLogs) > 0 {
			m.logsViewport.SetContent(strings.Join(m.installLogs, "\n"))
		}

		content += "\n\n" + logsTitle + "\n" + logsBox.Render(m.logsViewport.View())
		content += "\n" + infoStyle.Render("Use ↑↓ to scroll through logs")
	} else {
		// Installation complete - show logs for review
		content = "✅ Installation Complete - Review Output"

		// Show logs viewport for review
		logsTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(primaryOrange)).
			Bold(true).
			Render("📋 Installation Results")

		logsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(primaryOrange)).
			Padding(0, 1).
			Width(74).
			Height(14)

		m.logsViewport.Width = 72
		m.logsViewport.Height = 12

		// Update viewport content with current logs
		if len(m.installLogs) > 0 {
			m.logsViewport.SetContent(strings.Join(m.installLogs, "\n"))
		}

		content += "\n\n" + logsTitle + "\n" + logsBox.Render(m.logsViewport.View())
		content += "\n" + infoStyle.Render("Use ↑↓ to scroll • Press Ctrl+C to exit")
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) installContainersView() string {
	title := titleStyle.Render("🚀 Install Containers")

	content := `Would you like to install and start the containers now?

This will:
• Pull the required Docker/Podman images
• Start all Pangolin services
• Make your dashboard available

You can also install containers manually later.`

	content += "\n\n" + m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) crowdsecView() string {
	title := titleStyle.Render("🛡️ CrowdSec Security")

	content := `Would you like to install CrowdSec?

CrowdSec is a collaborative security solution that:
• Protects against brute force attacks
• Shares threat intelligence
• Provides real-time IP reputation

Note: This constitutes a minimal CrowdSec deployment. You'll need to configure it manually for optimal security.`

	content += "\n\n" + m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) crowdsecManageView() string {
	title := titleStyle.Render("⚙️ CrowdSec Management")

	content := `Are you willing to manage CrowdSec configuration?

CrowdSec will add complexity to your installation and requires:
• Manual configuration adjustments
• Ongoing maintenance
• Understanding of security policies

Consult the CrowdSec documentation for detailed setup instructions.`

	content += "\n\n" + m.renderButtons()

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) crowdsecInstallView() string {
	title := titleStyle.Render("🛡️ Installing CrowdSec")

	var content string

	if m.installing {
		// Show current step with animation
		step := m.installStep
		if step == "" {
			step = "Preparing CrowdSec installation"
		}

		content = fmt.Sprintf("%s %s", m.spinner.View(), step)

		// Show logs viewport during installation
		logsTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(primaryOrange)).
			Bold(true).
			Render("📋 CrowdSec Installation Logs")

		logsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(primaryOrange)).
			Padding(0, 1).
			Width(74).
			Height(14)

		m.logsViewport.Width = 72
		m.logsViewport.Height = 12

		// Update viewport content with current logs
		if len(m.installLogs) > 0 {
			m.logsViewport.SetContent(strings.Join(m.installLogs, "\n"))
		}

		content += "\n\n" + logsTitle + "\n" + logsBox.Render(m.logsViewport.View())
		content += "\n" + infoStyle.Render("Use ↑↓ to scroll through logs")
	} else {
		// Installation complete - show logs for review
		content = "✅ CrowdSec Installation Complete - Review Output"

		// Show logs viewport for review
		logsTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(primaryOrange)).
			Bold(true).
			Render("📋 Installation Results")

		logsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(primaryOrange)).
			Padding(0, 1).
			Width(74).
			Height(14)

		m.logsViewport.Width = 72
		m.logsViewport.Height = 12

		// Update viewport content with current logs
		if len(m.installLogs) > 0 {
			m.logsViewport.SetContent(strings.Join(m.installLogs, "\n"))
		}

		content += "\n\n" + logsTitle + "\n" + logsBox.Render(m.logsViewport.View())
		content += "\n" + infoStyle.Render("Use ↑↓ to scroll • Press Ctrl+C to exit")
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) setupTokenView() string {
	title := titleStyle.Render("🔑 Setup Token")

	content := `Generating setup token for first-time configuration...

This will create a secure token that allows you to:
• Complete the initial dashboard setup
• Create your first admin account
• Configure basic settings

The token will be displayed after installation completes.`

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		boxStyle.Render(content),
	)
}

func (m model) completeView() string {
	title := titleStyle.Render("🎉 Installation Complete!")

	message := fmt.Sprintf(`Pangolin has been successfully installed!

Dashboard URL: https://%s
Next steps:
1. Complete the initial setup at the dashboard
2. Create your first admin account  
3. Start creating secure tunnels

Thank you for using Pangolin! 🦎`, m.config.DashboardDomain)

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		successStyle.Render(boxStyle.Render(message)),
	)
}

func (m model) renderButtons() string {
	if len(m.fields) == 0 {
		return ""
	}

	var buttons []string
	for i, field := range m.fields {
		if field.fieldType == fieldButton {
			if i == m.focusIndex {
				buttons = append(buttons, focusedButtonStyle.Render(field.label))
			} else {
				buttons = append(buttons, buttonStyle.Render(field.label))
			}
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, buttons...)
}

// loadExistingConfig tries to load configuration from existing files
func loadExistingConfig() (*Config, error) {
	// Check if config.yml exists
	if _, err := os.Stat("config/config.yml"); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist")
	}

	// Try to read the app config to get dashboard URL
	appConfig, err := ReadAppConfig("config/config.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read app config: %v", err)
	}

	// Try to read Traefik config to get more values
	traefikConfig, err := ReadTraefikConfig("config/traefik/traefik_config.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read traefik config: %v", err)
	}

	// Parse dashboard URL to extract base domain and dashboard subdomain
	config := &Config{}

	// Extract domains from dashboard URL
	// Expected format: https://dashboard.example.com or https://subdomain.example.com
	if appConfig.DashboardURL != "" {
		dashboardURL := appConfig.DashboardURL
		// Remove protocol
		if strings.HasPrefix(dashboardURL, "https://") {
			dashboardURL = strings.TrimPrefix(dashboardURL, "https://")
		} else if strings.HasPrefix(dashboardURL, "http://") {
			dashboardURL = strings.TrimPrefix(dashboardURL, "http://")
		}

		// Split domain parts
		parts := strings.Split(dashboardURL, ".")
		if len(parts) >= 2 {
			config.DashboardDomain = dashboardURL
			// Try to extract base domain (last two parts)
			if len(parts) >= 2 {
				config.BaseDomain = strings.Join(parts[len(parts)-2:], ".")
			}
		}
	}

	// Set other values from Traefik config
	config.LetsEncryptEmail = traefikConfig.LetsEncryptEmail

	// Try to detect if it's hybrid mode by checking if certain fields exist
	// This is a best-effort detection
	config.HybridMode = false // Default to standard mode for existing installs

	// Set default container type to docker for existing installations
	config.InstallationContainerType = Docker

	return config, nil
}

// TUI installer entry point
func runTUIInstaller() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
