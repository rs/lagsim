package cmd

import (
	"lagsim/pkg/config"
	"lagsim/pkg/discovery"
	"lagsim/pkg/tc"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleSelected  = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236"))
	styleProfile   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	styleNoProfile = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleHelp      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleStatus    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleMenuCur   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	styleMenuDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleMenuBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("12")).Padding(0, 1)
)

type clientRow struct {
	IP      string
	Name    string
	MAC     string
	State   string
	Profile string // current applied profile
}

type mode int

const (
	modeList mode = iota
	modeMenu
	modeEdit
)

type model struct {
	clients      []clientRow
	profiles     []string // sorted profile names
	cursor       int      // selected client row
	scrollOffset int      // first visible client row index
	height       int      // terminal height
	width        int      // terminal width
	mode         mode
	menuCursor   int // selected item in profile menu
	menuItems    []string
	editBuf      string // text input buffer for name editing
	status       string
	quitting     bool
	runner       *tc.Runner
	cfg          *config.Config
	cfgPath      string
}

func newModel(c *config.Config, path string) model {
	r := &tc.Runner{DryRun: dryRun, Verbose: verbose}

	clients, _ := discovery.DiscoverClients(c.Interfaces.LAN)

	var rows []clientRow
	seen := make(map[string]bool)

	for _, cl := range clients {
		profile := c.Assignments[cl.IP]
		rows = append(rows, clientRow{
			IP:      cl.IP,
			Name:    resolveName(c, cl.MAC, cl.IP),
			MAC:     cl.MAC,
			State:   cl.State,
			Profile: profile,
		})
		seen[cl.IP] = true
	}

	for ip, profile := range c.Assignments {
		if !seen[ip] {
			rows = append(rows, clientRow{
				IP:      ip,
				Name:    reverseDNS(ip),
				MAC:     "-",
				State:   "OFFLINE",
				Profile: profile,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].IP < rows[j].IP })

	return model{
		clients:  rows,
		profiles: c.ProfileNames(),
		runner:   r,
		cfg:      c,
		cfgPath:  path,
	}
}

func (m *model) refreshClients() {
	discovered, err := discovery.DiscoverClients(m.cfg.Interfaces.LAN)
	if err != nil {
		return
	}

	// Build new row set, preserving applied profiles from config
	var rows []clientRow
	seen := make(map[string]bool)

	for _, cl := range discovered {
		profile := m.cfg.Assignments[cl.IP]
		rows = append(rows, clientRow{
			IP:      cl.IP,
			Name:    resolveName(m.cfg, cl.MAC, cl.IP),
			MAC:     cl.MAC,
			State:   cl.State,
			Profile: profile,
		})
		seen[cl.IP] = true
	}

	for ip, profile := range m.cfg.Assignments {
		if !seen[ip] {
			rows = append(rows, clientRow{
				IP:      ip,
				Name:    reverseDNS(ip),
				MAC:     "-",
				State:   "OFFLINE",
				Profile: profile,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].IP < rows[j].IP })

	// Preserve cursor position by tracking the selected IP
	var cursorIP string
	if m.cursor < len(m.clients) {
		cursorIP = m.clients[m.cursor].IP
	}

	m.clients = rows

	// Restore cursor to the same IP
	if cursorIP != "" {
		for i, c := range m.clients {
			if c.IP == cursorIP {
				m.cursor = i
				return
			}
		}
	}
	// Clamp cursor if the list shrank
	if m.cursor >= len(m.clients) && len(m.clients) > 0 {
		m.cursor = len(m.clients) - 1
	}
	m.clampScroll()
}

// visibleRows returns how many client rows fit on screen.
// Chrome lines: title, blank, header, blank, status?, edit?, blank, help, +2 for scroll indicators.
func (m model) visibleRows() int {
	if m.height <= 0 {
		return len(m.clients) // no size info yet, show all
	}
	chrome := 7 // title + blank + header + blank + blank + help + trailing newline
	if m.mode == modeEdit {
		chrome++
	}
	if m.status != "" {
		chrome++
	}
	// Reserve space for possible scroll indicators
	chrome += 2
	rows := m.height - chrome
	if rows < 1 {
		rows = 1
	}
	return rows
}

// clampScroll ensures the cursor is visible within the scroll window.
func (m *model) clampScroll() {
	visible := m.visibleRows()
	if m.scrollOffset > m.cursor {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func reverseDNS(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return "-"
	}
	return strings.TrimSuffix(names[0], ".")
}

func resolveName(c *config.Config, mac, ip string) string {
	if name, ok := c.Names[mac]; ok && name != "" {
		return name
	}
	return reverseDNS(ip)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd { return tickCmd() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.clampScroll()
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeMenu:
			return m.updateMenu(msg)
		case modeEdit:
			return m.updateEdit(msg)
		default:
			return m.updateList(msg)
		}
	case tickMsg:
		if m.mode == modeList {
			m.refreshClients()
		}
		return m, tickCmd()
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
		m.status = ""

	case "down", "j":
		if m.cursor < len(m.clients)-1 {
			m.cursor++
			m.clampScroll()
		}
		m.status = ""

	case "enter":
		if len(m.clients) > 0 {
			m.openMenu()
		}

	case "r", "delete":
		if len(m.clients) > 0 {
			m.removeProfile()
		}

	case "e":
		if len(m.clients) > 0 {
			c := m.clients[m.cursor]
			if c.MAC != "-" {
				m.editBuf = ""
				// Pre-fill with current custom name if any
				if name, ok := m.cfg.Names[c.MAC]; ok {
					m.editBuf = name
				}
				m.mode = modeEdit
				m.status = ""
			} else {
				m.status = "Cannot set name: no MAC address"
			}
		}
	}
	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "esc":
		m.mode = modeList
		m.status = ""

	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}

	case "down", "j":
		if m.menuCursor < len(m.menuItems)-1 {
			m.menuCursor++
		}

	case "enter":
		m.selectFromMenu()
	}
	return m, nil
}

func (m model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "esc":
		m.mode = modeList
		m.status = ""

	case "enter":
		c := &m.clients[m.cursor]
		name := strings.TrimSpace(m.editBuf)
		if name == "" {
			delete(m.cfg.Names, c.MAC)
			c.Name = reverseDNS(c.IP)
			m.status = fmt.Sprintf("Cleared custom name for %s", c.MAC)
		} else {
			m.cfg.Names[c.MAC] = name
			c.Name = name
			m.status = fmt.Sprintf("Set name %q for %s", name, c.MAC)
		}
		_ = config.Save(m.cfg, m.cfgPath)
		m.mode = modeList

	case "backspace":
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}

	case "ctrl+u":
		m.editBuf = ""

	default:
		// Only accept printable characters
		if len(msg.String()) == 1 && msg.String()[0] >= 32 {
			m.editBuf += msg.String()
		} else if len(msg.Runes) > 0 {
			m.editBuf += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *model) openMenu() {
	c := m.clients[m.cursor]
	// Menu: (none) + all profiles
	m.menuItems = append([]string{"(none)"}, m.profiles...)
	// Pre-select the current profile
	m.menuCursor = 0
	for i, item := range m.menuItems {
		if (item == c.Profile) || (item == "(none)" && c.Profile == "") {
			m.menuCursor = i
			break
		}
	}
	m.mode = modeMenu
	m.status = ""
}

func (m *model) selectFromMenu() {
	c := &m.clients[m.cursor]
	selected := m.menuItems[m.menuCursor]

	m.mode = modeList

	if selected == "(none)" {
		if c.Profile == "" {
			m.status = fmt.Sprintf("%s already has no profile", c.IP)
			return
		}
		m.applyToClient(c, "")
		return
	}

	if selected == c.Profile {
		m.status = fmt.Sprintf("%s already has %s", c.IP, selected)
		return
	}

	m.applyToClient(c, selected)
}

func (m *model) applyToClient(c *clientRow, profileName string) {
	// Auto-init if needed
	if !m.runner.QdiscExists(m.cfg.Interfaces.LAN, "htb 1:") {
		if err := tc.Setup(m.cfg, m.runner); err != nil {
			m.status = styleError.Render(fmt.Sprintf("init failed: %v", err))
			return
		}
	}

	if profileName == "" {
		_ = tc.RemoveFromAllDevices(m.runner, m.cfg, c.IP)
		delete(m.cfg.Assignments, c.IP)
		c.Profile = ""
		m.status = fmt.Sprintf("Removed profile from %s", c.IP)
	} else {
		if c.Profile != "" {
			_ = tc.RemoveFromAllDevices(m.runner, m.cfg, c.IP)
		}
		profile := m.cfg.Profiles[profileName]
		if err := tc.ApplyToAllDevices(m.runner, m.cfg, c.IP, profile); err != nil {
			m.status = styleError.Render(fmt.Sprintf("apply failed: %v", err))
			return
		}
		m.cfg.Assignments[c.IP] = profileName
		c.Profile = profileName
		m.status = fmt.Sprintf("Applied %s to %s", profileName, c.IP)
	}

	_ = config.Save(m.cfg, m.cfgPath)
}

func (m *model) removeProfile() {
	c := &m.clients[m.cursor]
	if c.Profile == "" {
		m.status = fmt.Sprintf("%s has no profile", c.IP)
		return
	}
	m.applyToClient(c, "")
}

func profileLabel(p string) string {
	if p == "" {
		return "(none)"
	}
	return p
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	const (
		wIP      = 15
		wName    = 22
		wMAC     = 19
		wState   = 11
		wProfile = 14
	)

	// Build all output lines first, then overlay the menu
	var lines []string

	lines = append(lines, styleHeader.Render("  lagsim — network condition simulator"))
	lines = append(lines, "")

	hdr := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s",
		wIP, "IP", wName, "NAME", wMAC, "MAC", wState, "STATE", wProfile, "PROFILE")
	lines = append(lines, styleHeader.Render(hdr))

	menuStartLine := -1 // line index where the menu overlay begins

	if len(m.clients) == 0 {
		lines = append(lines, "  No clients found on LAN.")
	}

	visible := m.visibleRows()
	end := m.scrollOffset + visible
	if end > len(m.clients) {
		end = len(m.clients)
	}

	if m.scrollOffset > 0 {
		lines = append(lines, styleHelp.Render(fmt.Sprintf("  ▲ %d more above", m.scrollOffset)))
	}

	for i := m.scrollOffset; i < end; i++ {
		c := m.clients[i]
		profStr := profileLabel(c.Profile)
		var profStyled string
		if c.Profile != "" {
			profStyled = styleProfile.Render(profStr)
		} else {
			profStyled = styleNoProfile.Render(profStr)
		}

		row := fmt.Sprintf("  %-*s %-*s %-*s %-*s ",
			wIP, c.IP, wName, c.Name, wMAC, c.MAC, wState, c.State)

		if i == m.cursor {
			lines = append(lines, styleSelected.Render(row)+profStyled)
			menuStartLine = len(lines) // menu starts on the line after the selected row
		} else {
			lines = append(lines, row+profStyled)
		}
	}

	if end < len(m.clients) {
		lines = append(lines, styleHelp.Render(fmt.Sprintf("  ▼ %d more below", len(m.clients)-end)))
	}

	// When the menu is open, add padding so the menu box overlays blank
	// space instead of clobbering the status/help lines at the bottom.
	if m.mode == modeMenu && menuStartLine >= 0 {
		// menu box height = items + 2 (top/bottom border)
		menuHeight := len(m.menuItems) + 2
		// lines available below the selected row before we add footer
		available := len(lines) - menuStartLine
		if pad := menuHeight - available; pad > 0 {
			for range pad {
				lines = append(lines, "")
			}
		}
	}

	lines = append(lines, "")
	if m.status != "" {
		lines = append(lines, "  "+styleStatus.Render(m.status))
	}

	// Edit input line
	if m.mode == modeEdit && len(m.clients) > 0 {
		c := m.clients[m.cursor]
		lines = append(lines, fmt.Sprintf("  Name for %s: %s%s",
			c.MAC,
			styleMenuCur.Render(m.editBuf),
			styleMenuCur.Render("█")))
	}

	lines = append(lines, "")
	switch m.mode {
	case modeMenu:
		lines = append(lines, styleHelp.Render("  ↑/↓ navigate   enter select   esc cancel"))
	case modeEdit:
		lines = append(lines, styleHelp.Render("  type name   enter save   esc cancel   ctrl+u clear"))
	default:
		lines = append(lines, styleHelp.Render("  ↑/↓ navigate   enter select profile   e edit name   r remove   q quit"))
	}

	// Overlay the menu box on the padded area
	if m.mode == modeMenu && menuStartLine >= 0 {
		lines = m.overlayMenu(lines, menuStartLine)
	}

	return strings.Join(lines, "\n") + "\n"
}

// overlayMenu composites the profile menu box on top of existing lines,
// starting at startLine, positioned near the right side of the table.
func (m model) overlayMenu(lines []string, startLine int) []string {
	// Build menu content
	var menuLines []string
	for i, item := range m.menuItems {
		if i == m.menuCursor {
			menuLines = append(menuLines, styleMenuCur.Render("▸ "+item))
		} else {
			menuLines = append(menuLines, styleMenuDim.Render("  "+item))
		}
	}
	content := strings.Join(menuLines, "\n")
	box := styleMenuBox.Render(content)
	boxLines := strings.Split(box, "\n")

	// Position: overlay starting at the PROFILE column area
	const overlayCol = 60

	// Ensure we have enough lines to overlay onto
	for len(lines) < startLine+len(boxLines) {
		lines = append(lines, "")
	}

	for i, boxLine := range boxLines {
		targetIdx := startLine + i
		lines[targetIdx] = overlayLine(lines[targetIdx], boxLine, overlayCol)
	}

	return lines
}

// overlayLine places overlay on top of base at the given column position.
// It uses visible character width to position correctly despite ANSI escapes.
func overlayLine(base, overlay string, col int) string {
	// Pad base to at least col visible characters
	baseRunes := []rune(stripAnsi(base))
	if len(baseRunes) < col {
		base += strings.Repeat(" ", col-len(baseRunes))
	}

	// Find the byte offset in base that corresponds to `col` visible chars.
	// We walk the base string, skipping ANSI sequences.
	byteOff := 0
	visible := 0
	raw := []byte(base)
	for byteOff < len(raw) && visible < col {
		if raw[byteOff] == '\x1b' {
			// Skip ANSI escape sequence
			j := byteOff + 1
			for j < len(raw) && raw[j] != 'm' {
				j++
			}
			if j < len(raw) {
				j++ // skip 'm'
			}
			byteOff = j
		} else {
			byteOff++
			visible++
		}
	}

	// Reset any styles before the overlay, then append overlay
	return string(raw[:byteOff]) + "\x1b[0m" + overlay
}

// stripAnsi removes ANSI escape sequences for measuring visible width.
func stripAnsi(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func runTUI() error {
	m := newModel(cfg, cfgPath)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
