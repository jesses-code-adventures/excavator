package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesses-code-adventures/excavator/src/utils"

	// Database
	_ "github.com/mattn/go-sqlite3"

	// Audio
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"

	// Frontend
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// ////////////////////// KEYMAPS ////////////////////////
// I know this is bad but i want gg
type keymapHacks struct {
	lastKey string
}

// Track this keystroke so it can be checked in the next one
func (k *keymapHacks) updateLastKey(key string) {
	k.lastKey = key
}

// Get the previous keystroke
func (k *keymapHacks) getLastKey() string {
	return k.lastKey
}

// For jumping to the top of the list
func (k *keymapHacks) lastKeyWasG() bool {
	return k.lastKey == "g"
}

// All possible keymap bindings
type KeyMap struct {
	Up                     key.Binding
	Down                   key.Binding
	Quit                   key.Binding
	JumpUp                 key.Binding
	JumpDown               key.Binding
	JumpBottom             key.Binding
	Audition               key.Binding
	SearchBuf              key.Binding
	Enter                  key.Binding
	NewCollection          key.Binding
	SetTargetCollection    key.Binding
	InsertMode             key.Binding
	ToggleAutoAudition     key.Binding
	AuditionRandom         key.Binding
	CreateQuickTag         key.Binding
	CreateTag              key.Binding
	SetTargetSubCollection key.Binding
	FuzzySearchFromRoot    key.Binding
}

// The actual help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Audition, k.SearchBuf, k.AuditionRandom, k.ToggleAutoAudition, k.NewCollection, k.SetTargetCollection, k.SetTargetSubCollection, k.CreateQuickTag, k.CreateTag}
}

// Empty because not using
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{},
	}
}

// The app's default key maps
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "esc"),
		key.WithHelp("q", "quit"),
	),
	JumpUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("^u", "jump up"),
	),
	JumpDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("^d", "jump down"),
	),
	JumpBottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "jump to bottom"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	InsertMode: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "text insert mode"),
	),
	Audition: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "audition sample"),
	),
	NewCollection: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "new collection"),
	),
	SetTargetSubCollection: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "target subcollection"),
	),
	SetTargetCollection: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "select collection"),
	),
	ToggleAutoAudition: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "auto audition"),
	),
	AuditionRandom: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "audition random sample"),
	),
	CreateQuickTag: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "quick tag"),
	),
	CreateTag: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "editable tag"),
	),
	SearchBuf: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search window"),
	),
	FuzzySearchFromRoot: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "search sounds from root"),
	),
}

// ////////////////////// STYLING ////////////////////////

// All styles to be used throughout the ui
var (
	// colours
	green = lipgloss.Color("#25A065")
	pink  = lipgloss.Color("#E441B5")
	white = lipgloss.Color("#FFFDF5")
	// App
	appStyle = lipgloss.NewStyle().
			Padding(1, 1)
	titleStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(green).
			Padding(1, 1).
			Height(3)
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#909090",
			Dark:  "#626262",
		})
	helpValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#B2B2B2",
			Dark:  "#4A4A4A",
		})
	// Directory Walker
	viewportStyle = lipgloss.NewStyle()
	selectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder()).
			Foreground(pink)
	unselectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder())
		// Form
	formStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Border(lipgloss.HiddenBorder()).
			Margin(0, 0, 0)
	unfocusedInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Width(100).
			Margin(1, 1).
			Border(lipgloss.HiddenBorder())
	focusedInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Width(100).
			Margin(1, 1)
		// Searchable list
	searchableListItemsStyle = lipgloss.NewStyle().
					Border(lipgloss.HiddenBorder())
	searchInputBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()).
				AlignVertical(lipgloss.Bottom).
				AlignHorizontal(lipgloss.Left)
	searchableSelectableListStyle = lipgloss.NewStyle().
					Border(lipgloss.HiddenBorder())
)

// A singular form input control
type formInput struct {
	name  string
	input textinput.Model
}

// Constructor for a form input
func newFormInput(name string) formInput {
	return formInput{
		name:  name,
		input: textinput.New(),
	}
}

// A generic form
type form struct {
	title        string
	inputs       []formInput
	writing      bool
	focusedInput int
}

// A form constructor
func newForm(title string, inputs []formInput) form {
	return form{
		title:        title,
		inputs:       inputs,
		writing:      false,
		focusedInput: 0,
	}
}

//// Form implementations ////

// Get the inputs for the new collection form
func getNewCollectionInputs() []formInput {
	return []formInput{
		newFormInput("name"),
		newFormInput("description"),
	}
}

// Get the new collection form
func getNewCollectionForm() form {
	return newForm("create collection", getNewCollectionInputs())
}

// Get the inputs for the new collection form
func getSearchInput() []formInput {
	return []formInput{
		newFormInput("search"),
	}
}

// Get the new collection form
func getTargetSubCollectionForm() form {
	return newForm("set target subcollection", getSearchInput())
}

// Get the new collection form
func getFuzzySearchRootForm() form {
	return newForm("fuzzy search from root", getSearchInput())
}

// Get the inputs for the new collection form
func getCreateCollectionTagInputs(defaultName string, defaultSubCollection string) []formInput {
	name := newFormInput("name")
	name.input.SetValue(defaultName)
	subcollection := newFormInput("subcollection")
	subcollection.input.SetValue(defaultSubCollection)
	return []formInput{
		name,
		subcollection,
	}
}

// Get the new collection form
func getCreateTagForm(defaultName string, defaultSubCollection string) form {
	return newForm("create tag", getCreateCollectionTagInputs(defaultName, defaultSubCollection))
}

/// List selection ///

// Interface for list selection items so the list can easily be reused
type selectableListItem interface {
	Id() int
	Name() string
	Description() string
	IsDir() bool
	IsFile() bool
}

// A list where a single item can be selected
type selectableList struct {
	title string
}

// A constructor for a selectable list
type searchableSelectableList struct {
	title  string
	search formInput
}

func newSearchableList(title string) searchableSelectableList {
	return searchableSelectableList{
		title: title,
		search: formInput{
			name:  "search",
			input: textinput.New(),
		},
	}
}

func (m model) filterListItems() model {
	var resp []selectableListItem
	switch m.searchableSelectableList.title {
	case "set target subcollection":
		r := m.server.searchCollectionSubcollections(m.searchableSelectableList.search.input.Value())
		newArray := make([]selectableListItem, 0)
		for _, item := range r {
			newArray = append(newArray, item)
		}
		resp = newArray
		m.server.navState.choices = resp
	case "fuzzy search from root":
		log.Println("performing fuzzy search")
		m.server.fuzzyFind(m.searchableSelectableList.search.input.Value(), true)
	case "fuzzy search window":
		log.Println("performing fuzzy search")
		m.server.fuzzyFind(m.searchableSelectableList.search.input.Value(), false)
	}
	return m
}

// A generic model defining app behaviour in all states
type model struct {
	ready                    bool
	quitting                 bool
	cursor                   int
	prevCursor               int
	viewportHeight           int
	viewportWidth            int
	keys                     KeyMap
	keyHack                  keymapHacks
	server                   *server
	viewport                 viewport.Model
	help                     help.Model
	windowType               windowType
	form                     form
	selectableList           string
	searchableSelectableList searchableSelectableList
}

// Different window types
type windowType int

const (
	DirectoryWalker windowType = iota
	FormWindow
	ListSelectionWindow
	SearchableSelectableList
)

func (w windowType) String() string {
	return [...]string{"DirectoryWalker", "FormWindow", "ListSelectionWindow", "SearchableSelectableList"}[w]
}

// Constructor for the app's model
func excavatorModel(server *server) model {
	return model{
		ready:      false,
		quitting:   false,
		server:     server,
		help:       help.New(),
		keys:       DefaultKeyMap,
		windowType: DirectoryWalker,
	}
}

// Get the header of the viewport
func (m model) headerView() string {
	title := titleStyle.Render("Excavator - Samples")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

type statusDisplayItem struct {
	key        string
	value      string
	keyStyle   lipgloss.Style
	valueStyle lipgloss.Style
}

func newStatusDisplayItem(key string, value string) statusDisplayItem {
	return statusDisplayItem{
		key:        key,
		value:      value,
		keyStyle:   helpKeyStyle,
		valueStyle: helpValueStyle,
	}
}

func (s statusDisplayItem) View() string {
	return s.keyStyle.Render(s.key+": ") + s.valueStyle.Render(s.value)
}

// Display key info to user
func (m model) getStatusDisplay() string {
	termWidth := m.viewport.Width
	msg := ""
	// hack to make centering work
	msgRaw := fmt.Sprintf("collection: %v, subcollection: %v, window: %v, num items: %v", m.server.currentUser.targetCollection.Name(), m.server.currentUser.targetSubCollection, m.windowType.String(), len(m.server.navState.choices))
	items := []statusDisplayItem{
		newStatusDisplayItem("collection", m.server.currentUser.targetCollection.Name()),
		newStatusDisplayItem("subcollection", m.server.currentUser.targetSubCollection),
		newStatusDisplayItem("window", fmt.Sprintf("%v", m.windowType.String())),
		newStatusDisplayItem("num items", fmt.Sprintf("%v", len(m.server.navState.choices))),
	}
	for i, item := range items {
		msg += item.View()
		if i != len(items)-1 {
			msg = msg + helpValueStyle.Render(", ")
		}
	}
	padding := (termWidth - len(msgRaw)) / 2
	if padding < 0 {
		padding = 0
	}
	paddedHelpStyle := lipgloss.NewStyle().PaddingLeft(padding).PaddingRight(padding).
		Render(msg)
	return paddedHelpStyle
}

// Get the footer of the view
func (m model) footerView() string {
	helpText := m.help.View(m.keys)
	termWidth := m.viewport.Width
	helpTextLength := lipgloss.Width(helpText)
	padding := (termWidth - helpTextLength) / 2
	if padding < 0 {
		padding = 0
	}
	paddedHelpStyle := lipgloss.NewStyle().PaddingLeft(padding).PaddingRight(padding)
	centeredHelpText := paddedHelpStyle.Render(helpText)
	var searchInput string
	if m.windowType == SearchableSelectableList {
		searchInput = searchInputBoxStyle.Render(m.searchableSelectableList.search.input.View())
		return searchInput + "\n" + m.getStatusDisplay() + "\n" + centeredHelpText

	}
	return m.getStatusDisplay() + "\n" + centeredHelpText
}

// Formview handler
func (m model) formView() string {
	s := ""
	log.Println("got form ", m.form)
	for i, input := range m.form.inputs {
		if m.form.focusedInput == i {
			s += focusedInput.Render(fmt.Sprintf("%v: %v\n", input.name, input.input.View()))
		} else {
			s += unfocusedInput.Render(fmt.Sprintf("%v: %v\n", input.name, input.input.View()))
		}
	}
	return s
}

// Standard content handler
func (m model) directoryView() string {
	s := ""
	for i, choice := range m.server.navState.choices {
		var newLine string
		if m.cursor == i {
			cursor := "--> "
			newLine = fmt.Sprintf("%s %s", cursor, choice.Name())
		} else {
			newLine = fmt.Sprintf("     %s", choice.Name())
		}
		if len(newLine) > m.viewport.Width {
			newLine = newLine[:m.viewport.Width-2]
		}
		if m.cursor == i {
			newLine = selectedStyle.Render(newLine, fmt.Sprintf("    %v", choice.Description()))
		} else {
			newLine = unselectedStyle.Render(newLine)
		}
		s += newLine
	}
	return s
}

// Necessary for bubbletea model interface
func (m model) Init() tea.Cmd {
	return nil
}

// // Ui updating for window resize events
func (m model) handleWindowResize(msg tea.WindowSizeMsg) model {
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	searchInputHeight := 2 // Assuming the search input height is approximately 2 lines
	verticalPadding := 2   // Adjust based on your app's padding around the viewport
	// Calculate available height differently if in SearchableSelectableList mode
	if m.windowType == SearchableSelectableList {
		m.viewportHeight = msg.Height - headerHeight - footerHeight - searchInputHeight - verticalPadding
	} else {
		m.viewportHeight = msg.Height - headerHeight - footerHeight - verticalPadding
	}

	m.viewport.Width = msg.Width
	m.viewport.Height = m.viewportHeight
	m.ready = true

	return m
}

// Handle viewport positioning
func (m model) ensureCursorVerticallyCentered() viewport.Model {
	if m.windowType != DirectoryWalker {
		m.viewport.GotoTop()
		return m.viewport
	}
	viewport := m.viewport
	itemHeight := 2
	cursorPosition := m.cursor * itemHeight
	viewportHeight := viewport.Height
	viewport.YOffset = (cursorPosition - viewportHeight + itemHeight) + (viewportHeight / 2)
	if viewport.PastBottom() {
		viewport.GotoBottom()
	}
	if viewport.YOffset < 0 {
		viewport.YOffset = 0
	}
	return viewport
}

// Helper function to find the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Set the content of the viewport based on the window type
func (m model) setViewportContent(msg tea.Msg, cmd tea.Cmd) (model, tea.Cmd) {
	switch m.windowType {
	case FormWindow:
		m.viewport.SetContent(m.formView())
	case DirectoryWalker:
		m.viewport = m.ensureCursorVerticallyCentered()
		m.viewport.SetContent(m.directoryView())
	case ListSelectionWindow:
		m.viewport.SetContent(m.directoryView())
	case SearchableSelectableList:
		m.viewport = m.ensureCursorVerticallyCentered()
		m.viewport.SetContent(m.directoryView())
	default:
		m.viewport.SetContent("Invalid window type")
	}
	return m, cmd
}

// Handle all view rendering
func (m model) View() string {
	if m.quitting {
		return ""
	}
	// Main content view
	var contentView string
	switch m.windowType {
	case DirectoryWalker, FormWindow, ListSelectionWindow, SearchableSelectableList:
		contentView = fmt.Sprintf("%s\n%s\n%s", m.headerView(), viewportStyle.Render(m.viewport.View()), m.footerView())
	// case SearchableSelectableList:
	// 	contentView = fmt.Sprintf("%s\n%s\n%s", m.headerView(), searchableSelectableListStyle.Render(m.directoryView()),  m.footerView())
	default:
		contentView = "Invalid window type"
	}

	return appStyle.Render(contentView)
}

// ////////////////////// UI UPDATING ////////////////////////

func (m model) clearModel() model {
	m.form = form{}
	m.server.navState.choices = make([]selectableListItem, 0)
	m.searchableSelectableList = searchableSelectableList{}
	m.cursor = 0
	return m
}

// Standard "home" view
func (m model) goToMainWindow(msg tea.Msg, cmd tea.Cmd) (model, tea.Cmd) {
	m = m.clearModel()
	m.windowType = DirectoryWalker
	m.server.updateChoices()
	return m, cmd
}

// Individual logic handlers for each list - setting window type handled outside this function
func (m model) handleTitledList(msg tea.Msg, cmd tea.Cmd, title string) (model, tea.Cmd) {
	switch title {
	case "set target subcollection":
		subCollections := m.server.getCollectionSubcollections()
		for _, subCollection := range subCollections {
			m.server.navState.choices = append(m.server.navState.choices, subCollection)
		}
	case "select collection":
		collections := m.server.getCollections()
		m.selectableList = title
		for _, collection := range collections {
			m.server.navState.choices = append(m.server.navState.choices, collection)
		}
	case "search for collection":
		collections := m.server.getCollections()
		m.selectableList = title
		for _, collection := range collections {
			m.server.navState.choices = append(m.server.navState.choices, collection)
		}
	case "fuzzy search from root":
		files := m.server.fuzzyFind("", true)
		for _, subCollection := range files {
			m.server.navState.choices = append(m.server.navState.choices, subCollection)
		}
	default:
		log.Fatalf("Invalid searchable selectable list title")
	}
	m.searchableSelectableList = newSearchableList(title)
	return m, cmd
}

func (m model) handleForm(msg tea.Msg, cmd tea.Cmd, title string) (model, tea.Cmd) {
	switch title {
	case "new collection":
		m = m.clearModel()
		m.form = getNewCollectionForm()
	case "create tag":
		m.form = getCreateTagForm(path.Base(m.server.navState.choices[m.cursor].Name()), m.server.currentUser.targetSubCollection)
	}
	return m, cmd
}

// Main handler to be called any time the window changes
func (m model) setWindowType(msg tea.Msg, cmd tea.Cmd, windowType windowType, title string) (model, tea.Cmd) {
	if m.windowType == windowType && m.searchableSelectableList.title == title {
		m, cmd = m.goToMainWindow(msg, cmd)
		return m, cmd
	}
	log.Println("got window type ", windowType.String())
	switch windowType {
	case DirectoryWalker:
		m = m.clearModel()
		m, cmd = m.goToMainWindow(msg, cmd)
	case FormWindow:
		if title == "" {
			log.Fatalf("Title required for forms")
		}
		m, cmd = m.handleForm(msg, cmd, title)
	case ListSelectionWindow:
		m = m.clearModel()
		if title == "" {
			log.Fatalf("Title required for lists")
		}
		m, cmd = m.handleTitledList(msg, cmd, title)
	case SearchableSelectableList:
		m = m.clearModel()
		if title == "" {
			log.Fatalf("Title required for lists")
		}
		m, cmd = m.handleTitledList(msg, cmd, title)
	default:
		log.Fatalf("Invalid window type")
	}
	m.windowType = windowType
	m.cursor = 0
	return m, cmd
}

// Audition the file under the cursor
func (m model) auditionCurrentlySelectedFile() {
	if len(m.server.navState.choices) == 0 {
		return
	}
	choice := m.server.navState.choices[m.cursor]
	if !choice.IsDir() && choice.IsFile() {
		var path string
		if !strings.Contains(choice.Name(), m.server.navState.currentDir) {
			path = filepath.Join(m.server.navState.currentDir, choice.Name())
		} else {
			path = choice.Name()
		}
		go m.server.audioPlayer.PlayAudioFile(path)
	}
}

// These functions should run every time the cursor moves in directory view
func (m model) dirVerticalNavEffect() {
	if m.server.currentUser.autoAudition {
		m.auditionCurrentlySelectedFile()
	}
}

// To be used across many window types for navigation
func (m model) handleStandardMovementKey(msg tea.KeyMsg) model {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.server.navState.choices)-1 {
			m.cursor++
		}
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.JumpDown):
		if m.cursor < len(m.server.navState.choices)-8 {
			m.cursor += 8
		} else {
			m.cursor = len(m.server.navState.choices) - 1
		}
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.JumpUp):
		if m.cursor > 8 {
			m.cursor -= 8
		} else {
			m.cursor = 0
		}
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.Audition):
		m.auditionCurrentlySelectedFile()
	case key.Matches(msg, m.keys.AuditionRandom):
		fileIndex := m.server.navState.getRandomAudioFileIndex()
		if fileIndex != -1 {
			m.cursor = fileIndex
			m.dirVerticalNavEffect()
		}
		if !m.server.currentUser.autoAudition {
			m.auditionCurrentlySelectedFile()
		}
	case key.Matches(msg, m.keys.JumpBottom):
		m.viewport.GotoBottom()
		m.cursor = len(m.server.navState.choices) - 1
		m.dirVerticalNavEffect()
	}
	return m
}

// Handle a single key press
func (m model) handleDirectoryKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, cmd
	case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) || key.Matches(msg, m.keys.JumpDown) || key.Matches(msg, m.keys.JumpUp) || key.Matches(msg, m.keys.Audition) || key.Matches(msg, m.keys.AuditionRandom) || key.Matches(msg, m.keys.JumpBottom):
		m = m.handleStandardMovementKey(msg)
	case key.Matches(msg, m.keys.Enter):
		choice := m.server.navState.choices[m.cursor]
		if choice.IsDir() {
			if choice.Name() == ".." {
				m.cursor = 0
				m.server.navState.changeToParentDir()
			} else {
				m.cursor = 0
				m.server.navState.changeDir(choice.Name())
			}
		}
	case key.Matches(msg, m.keys.NewCollection):
		m, cmd = m.setWindowType(msg, cmd, FormWindow, "new collection")
	case key.Matches(msg, m.keys.SetTargetSubCollection):
		log.Println("going to set target subcollection")
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "set target subcollection")
	case key.Matches(msg, m.keys.FuzzySearchFromRoot):
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "fuzzy search from root")
	case key.Matches(msg, m.keys.SetTargetCollection):
		m, cmd = m.setWindowType(msg, cmd, ListSelectionWindow, "search for collection")
	case key.Matches(msg, m.keys.ToggleAutoAudition):
		m.server.updateAutoAudition(!m.server.currentUser.autoAudition)
	case key.Matches(msg, m.keys.CreateQuickTag):
		choice := m.server.navState.choices[m.cursor]
		if !choice.IsDir() {
			m.server.createQuickTag(choice.Name())
		}
		m.server.updateChoices()
	case key.Matches(msg, m.keys.CreateTag):
		m, cmd = m.setWindowType(msg, cmd, FormWindow, "create tag")
	default:
		switch msg.String() {
		case "g":
			if m.keyHack.getLastKey() == "g" {
				m.cursor = 0
			}
		}
	}
	return m, cmd
}

// List selection navigation
func (m model) handleListSelectionKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m.goToMainWindow(msg, cmd)
	case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) || key.Matches(msg, m.keys.JumpDown) || key.Matches(msg, m.keys.JumpUp) || key.Matches(msg, m.keys.Audition) || key.Matches(msg, m.keys.AuditionRandom) || key.Matches(msg, m.keys.JumpBottom):
		m = m.handleStandardMovementKey(msg)
	case key.Matches(msg, m.keys.NewCollection):
		m.form = getNewCollectionForm()
		m, cmd = m.setWindowType(msg, cmd, FormWindow, "")
	case key.Matches(msg, m.keys.SetTargetCollection):
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "search for collection")
	case key.Matches(msg, m.keys.SetTargetSubCollection):
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "set target subcollection")
	case key.Matches(msg, m.keys.FuzzySearchFromRoot):
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "fuzzy search from root")
	case key.Matches(msg, m.keys.ToggleAutoAudition):
		m.server.updateAutoAudition(!m.server.currentUser.autoAudition)
	case key.Matches(msg, m.keys.Enter):
		switch m.selectableList {
		case "search for collection":
			if collection, ok := m.server.navState.choices[m.cursor].(collection); ok {
				m.server.updateTargetCollection(collection)
				m, cmd = m.goToMainWindow(msg, cmd)
			} else {
				log.Fatalf("Invalid list selection item type")
			}
		}
	}
	return m, cmd
}

// Form key
func (m model) handleFormKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	if m.form.writing {
		m, cmd = m.handleFormWritingKey(msg, cmd)
	} else {
		m, cmd = m.handleFormNavigationKey(msg, cmd)
	}
	return m, cmd
}

// Form navigation
func (m model) handleFormNavigationKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.form.inputs[m.form.focusedInput].input.Blur()
		m.form.focusedInput++
		if m.form.focusedInput >= len(m.form.inputs) {
			m.form.focusedInput = 0
		}
		m.form.inputs[m.form.focusedInput].input.Focus()
	case key.Matches(msg, m.keys.Down):
		m.form.inputs[m.form.focusedInput].input.Blur()
		m.form.focusedInput--
		if m.form.focusedInput < 0 {
			m.form.focusedInput = len(m.form.inputs) - 1
		}
		m.form.inputs[m.form.focusedInput].input.Focus()
	case key.Matches(msg, m.keys.InsertMode):
		m.form.inputs[m.form.focusedInput].input.Blur()
		m.form.writing = true
		m.form.inputs[m.form.focusedInput].input.Focus()
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.NewCollection), key.Matches(msg, m.keys.SetTargetSubCollection), key.Matches(msg, m.keys.FuzzySearchFromRoot):
		m, cmd = m.goToMainWindow(msg, cmd)
	case key.Matches(msg, m.keys.SetTargetCollection):
		// m, cmd = m.handleListSelectionKey(msg, cmd)
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "search for collection")
	case key.Matches(msg, m.keys.ToggleAutoAudition):
		m.server.updateAutoAudition(!m.server.currentUser.autoAudition)
	case key.Matches(msg, m.keys.Enter):
		for i, input := range m.form.inputs {
			if input.input.Value() == "" {
				m.form.focusedInput = i
				m.form.inputs[i].input.Focus()
				return m, cmd
			}
		}
		switch m.form.title {
		case "create collection":
			m.server.createCollection(m.form.inputs[0].input.Value(), m.form.inputs[1].input.Value())
		case "create tag":
			m.server.createTag(m.server.navState.choices[m.cursor].Name(), m.form.inputs[0].input.Value(), m.form.inputs[1].input.Value())
		}
		// case "set target subcollection":
		// 	m.server.updateTargetSubCollection(m.form.inputs[0].input.Value())
		m, cmd = m.goToMainWindow(msg, cmd)
	}
	return m, cmd
}

// Form writing
func (m model) handleFormWritingKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.form.writing = false
		if m.windowType == SearchableSelectableList {
			m.searchableSelectableList.search.input.Blur()
			m = m.filterListItems()
			log.Printf("filtered items in form writing key quit %v", m.server.navState.choices)
			m.cursor = 0
		} else {
			m.form.inputs[m.form.focusedInput].input.Blur()
		}
	case key.Matches(msg, m.keys.Enter):
		m.form.writing = false
		if m.windowType == SearchableSelectableList {
			m.searchableSelectableList.search.input.Blur()
			m = m.filterListItems()
			log.Printf("filtered items in form writing key enter %v", m.server.navState.choices)
			m.cursor = 0
		} else {
			m.form.inputs[m.form.focusedInput].input.Blur()
		}
	default:
		var newInput textinput.Model
		if m.windowType == SearchableSelectableList {
			newInput, cmd = m.searchableSelectableList.search.input.Update(msg)
			m.searchableSelectableList.search.input = newInput
			m.searchableSelectableList.search.input.Focus()
			if m.selectableList == "set target subcollection" {
				m = m.filterListItems()
				m.cursor = 0
			}
		} else {
			newInput, cmd = m.form.inputs[m.form.focusedInput].input.Update(msg)
			m.form.inputs[m.form.focusedInput].input = newInput
			m.form.inputs[m.form.focusedInput].input.Focus()
		}
	}
	return m, cmd
}

// List selection navigation
func (m model) handleSearchableListNavKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m.goToMainWindow(msg, cmd)
	case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) || key.Matches(msg, m.keys.JumpDown) || key.Matches(msg, m.keys.JumpUp) || key.Matches(msg, m.keys.Audition) || key.Matches(msg, m.keys.AuditionRandom) || key.Matches(msg, m.keys.JumpBottom):
		m = m.handleStandardMovementKey(msg)
	case key.Matches(msg, m.keys.NewCollection):
		m.form = getNewCollectionForm()
		m, cmd = m.setWindowType(msg, cmd, FormWindow, "")
	case key.Matches(msg, m.keys.SetTargetSubCollection):
		m.form = getTargetSubCollectionForm()
		m, cmd = m.setWindowType(msg, cmd, FormWindow, "")
	case key.Matches(msg, m.keys.SetTargetCollection):
		m, cmd = m.setWindowType(msg, cmd, SearchableSelectableList, "search for collection")
	case key.Matches(msg, m.keys.InsertMode) || key.Matches(msg, m.keys.SearchBuf):
		m.searchableSelectableList.search.input.Focus()
		m.form.writing = true
	case key.Matches(msg, m.keys.ToggleAutoAudition):
		m.server.updateAutoAudition(!m.server.currentUser.autoAudition)
	case key.Matches(msg, m.keys.Enter):
		value := m.searchableSelectableList.search.input.Value()
		switch m.searchableSelectableList.title {
		case "fuzzy search from root":
			if value == "" {
				return m, cmd
			}
			m.cursor = 0
			m.server.fuzzyFind(value, true)
			return m, cmd
		case "fuzzy search window":
			if value == "" {
				return m, cmd
			}
			m.cursor = 0
			m.server.fuzzyFind(value, false)
			return m, cmd
		case "set target subcollection":
			if len(m.server.navState.choices) == 0 && len(value) > 0 {
				m.server.updateTargetSubCollection(value)
			} else {
				selected := m.server.navState.choices[m.cursor]
				log.Printf("selected: %v", selected)
				if collection, ok := selected.(selectableListItem); ok {
					log.Printf("selected collection: %v", collection.Name())
					m.server.updateTargetSubCollection(collection.Name())
				} else {
					log.Fatalf("Invalid list selection item type")
				}
			}
			m, cmd = m.goToMainWindow(msg, cmd)
		}
	}
	return m, cmd
}

// Form key
func (m model) handleSearchableListKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	if m.form.writing {
		m, cmd = m.handleFormWritingKey(msg, cmd)
	} else {
		m, cmd = m.handleSearchableListNavKey(msg, cmd)
	}
	return m, cmd
}

// Takes a message and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.handleWindowResize(msg)
	case tea.KeyMsg:
		switch m.windowType {
		case FormWindow:
			m, cmd = m.handleFormKey(msg, cmd)
		case ListSelectionWindow:
			m, cmd = m.handleListSelectionKey(msg, cmd)
		case DirectoryWalker:
			m, cmd = m.handleDirectoryKey(msg, cmd)
			if m.quitting {
				return m, tea.Quit
			}
		case SearchableSelectableList:
			m, cmd = m.handleSearchableListKey(msg, cmd)
		}
		m.keyHack.updateLastKey(msg.String())
	}
	return m.setViewportContent(msg, cmd)
}

//////////////////////// LOCAL SERVER ////////////////////////

// A connection between a tag and a collection
type collectionTag struct {
	filePath       string
	collectionName string
	subCollection  string
}

// A directory entry with associated tags
type dirEntryWithTags struct {
	path  string
	tags  []collectionTag
	isDir bool
}

func (d dirEntryWithTags) Id() int {
	return 0
}

func (d dirEntryWithTags) Name() string {
	return d.path
}

func (d dirEntryWithTags) Description() string {
	return d.displayTags()
}

func (d dirEntryWithTags) IsDir() bool {
	return d.isDir
}

func (d dirEntryWithTags) IsFile() bool {
	return !d.isDir
}

// A string representing the collection tags associated with a directory entry
func (d dirEntryWithTags) displayTags() string {
	first := true
	resp := ""
	for _, tag := range d.tags {
		if first {
			resp = fmt.Sprintf("%s: %s", tag.collectionName, tag.subCollection)
			first = false
		} else {
			resp = fmt.Sprintf("%s, %s: %s", resp, tag.collectionName, tag.subCollection)
		}
	}
	return resp
}

// A user
type user struct {
	id                  int
	name                string
	autoAudition        bool
	targetCollection    *collection
	targetSubCollection string
	root                string
}

// Struct holding the app's configuration
type config struct {
	data              string
	root              string
	dbFileName        string
	createSqlCommands []byte
}

// Constructor for the Config struct
func newConfig(data string, root string, dbFileName string) *config {
	log.Printf("data: %v, samples: %v", data, root)
	data = utils.ExpandHomeDir(data)
	root = utils.ExpandHomeDir(root)
	log.Printf("expanded data: %v, samples: %v", data, root)
	sqlCommands, err := os.ReadFile("src/sql_commands/create_db.sql")
	if err != nil {
		log.Fatalf("Failed to read SQL commands: %v", err)
	}
	rootExists := true
	if _, err := os.Stat(root); os.IsNotExist(err) {
		rootExists = false
	}
	if !rootExists {
		log.Fatalf("No root samples directory found at %v", root)
	}
	config := config{
		data:              data,
		root:              root,
		dbFileName:        dbFileName,
		createSqlCommands: sqlCommands,
	}
	return &config
}

func (c *config) setRoot(root string) {
	root = utils.ExpandHomeDir(root)
	c.root = root
}

func createDirectories(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Fatal("Creating directory failed at ", dir)
	}
}

// Handles either creating or checking the existence of the data and samples directories
func (c *config) createDataDirectory() {
    createDirectories(c.data)
}

// Standardized file structure for the database file
func (c *config) getDbPath() string {
	if c.dbFileName == "" {
		c.dbFileName = "excavator"
	}
	if !strings.HasSuffix(c.dbFileName, ".db") {
		c.dbFileName = c.dbFileName + ".db"
	}
	return filepath.Join(c.data, c.dbFileName)
}

type navState struct {
	root           string
	currentDir     string
	choiceChannel  chan selectableListItem
	choices        []selectableListItem
	collectionTags func(path string) []collectionTag
}

func newNavState(root string, currentDir string, collectionTags func(path string) []collectionTag) *navState {
	choiceChannel := make(chan selectableListItem)
	navState := navState{
		root:           root,
		currentDir:     currentDir,
		choiceChannel:  choiceChannel,
		choices:        make([]selectableListItem, 0),
		collectionTags: collectionTags,
	}
	go navState.Run()
	return &navState
}

func (n *navState) Run() {
	for {
		select {
		case choice := <-n.choiceChannel:
			n.choices = append(n.choices, choice)
		}
	}
}

func (n *navState) pushChoice(choice selectableListItem) {
	n.choiceChannel <- choice
}

// Grab an index of some audio file within the current directory
func (n *navState) getRandomAudioFileIndex() int {
	if len(n.choices) == 0 {
		return -1
	}
	possibleIndexes := make([]int, 0)
	for i, choice := range n.choices {
		if !choice.IsDir() {
			possibleIndexes = append(possibleIndexes, i)
		}
	}
	return possibleIndexes[rand.Intn(len(possibleIndexes))]
}

// Populate the choices array with the current directory's contents
func (n *navState) updateChoices() {
	if n.currentDir != n.root {
		n.choices = make([]selectableListItem, 0)
		dirEntries := n.listDirEntries()
		n.choices = append(n.choices, dirEntryWithTags{path: "..", tags: make([]collectionTag, 0), isDir: true})
		n.choices = append(n.choices, dirEntries...)
	} else {
		n.choices = n.listDirEntries()
	}
}

// Return only directories and valid audio files
func (f *navState) filterDirEntries(entries []os.DirEntry) []os.DirEntry {
	dirs := make([]os.DirEntry, 0)
	files := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entry)
			continue
		}
		if strings.HasSuffix(entry.Name(), ".wav") || strings.HasSuffix(entry.Name(), ".mp3") ||
			strings.HasSuffix(entry.Name(), ".flac") {
			files = append(files, entry)
		}
	}
	return append(dirs, files...)
}

// Standard function for getting the necessary files from a dir with their associated tags
func (f *navState) listDirEntries() []selectableListItem {
	files, err := os.ReadDir(f.currentDir)
	log.Printf("current dir: %v", f.currentDir)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	files = f.filterDirEntries(files)
	var samples []selectableListItem
	for _, file := range files {
		matchedTags := make([]collectionTag, 0)
		isDir := file.IsDir()
		if !isDir {
			for _, tag := range f.collectionTags(f.currentDir) {
				if strings.Contains(tag.filePath, file.Name()) {
					matchedTags = append(matchedTags, tag)
				}
			}
		}
		samples = append(samples, dirEntryWithTags{path: file.Name(), tags: matchedTags, isDir: isDir})
	}
	return samples
}

// Get the full path of the current directory
func (n *navState) getCurrentDirPath() string {
	return filepath.Join(n.root, n.currentDir)
}

// Change the current directory
func (n *navState) changeDir(dir string) {
	log.Println("Changing to dir: ", dir)
	n.currentDir = filepath.Join(n.currentDir, dir)
	log.Println("Current dir: ", n.currentDir)
	n.updateChoices()
}

// Change the current directory to the root
func (n *navState) changeToRoot() {
	n.currentDir = n.root
	n.updateChoices()
}

// Change the current directory to the parent directory
func (n *navState) changeToParentDir() {
	log.Println("Changing to dir: ", filepath.Dir(n.currentDir))
	n.currentDir = filepath.Dir(n.currentDir)
	n.updateChoices()
}

func (s *server) getAllDirectories(path string) []string {
	paths, err := os.ReadDir(path)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	var dirs []string
	for _, path := range paths {
		if path.IsDir() {
			dirs = append(dirs, path.Name())
		}
	}
	return dirs
}

// The main struct holding the server
type server struct {
	db          *sql.DB
	currentUser user
	navState    *navState
	audioPlayer *AudioPlayer
}

// Construct the server
func newServer(audioPlayer *AudioPlayer) *server {
	var data = flag.String("data", "~/.excavator-tui", "Local data storage path")
	var samples = flag.String("root", "~/Library/Audio/Sounds/Samples", "Root samples directory")
	var user = flag.String("user", "jesse", "User name to launch with")
	var dbFileName = flag.String("db", "excavator", "Database file name")
	flag.Parse()
	config := newConfig(*data, *samples, *dbFileName)
	config.createDataDirectory()
	dbPath := config.getDbPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to create sqlite file %v", err)
	}
	if _, err := os.Stat(config.getDbPath()); os.IsNotExist(err) {
		_, innerErr := db.Exec(string(config.createSqlCommands))
		if innerErr != nil {
			log.Fatalf("Failed to execute SQL commands: %v", innerErr)
		}
	}
	s := server{
		db:          db,
		audioPlayer: audioPlayer,
	}
	users := s.getUsers(user)
    log.Print("users: ", users)
    log.Print("user: ", *user)
	if len(users) == 0 && len(*user) > 0 {
        id := s.createUser(*user)
        if id == 0 {
            log.Fatal("Failed to create user")
        }
        s.currentUser = s.getUser(id)
    } else if len(users) > 0 && len(*user) > 0  && users[0].name != *user {
        createdIdx := s.createUser(*user)
        s.currentUser = s.getUser(createdIdx)
    } else if len(users) > 0 && len(*user) == 0 {
        s.currentUser = users[0]
    } else if len(users) > 0 && len(*user) > 0 {
        found := 0
        for i, u := range users {
            if u.name == *user {
                found = i
            }
        }
        if found == 0 {
            createdIdx := s.createUser(*user)
            s.currentUser = s.getUser(createdIdx)
        } else {
            s.currentUser = users[found]
        }
	} else {
		log.Fatal("No users found")
	}
	if s.currentUser.root == "" && config.root == "" {
		log.Fatal("No root found")
	} else if config.root == "" {
		config.root = s.currentUser.root
	} else if s.currentUser.root == "" {
		s.currentUser.root = config.root // TODO: prompt the user to see if they want to save the root
		s.updateRootInDb(config.root)
	} else if s.currentUser.root != config.root {
		log.Println("launched with temporary root ", config.root)
		s.currentUser.root = config.root
	}
	log.Printf("Current user: %v, selected collection: %v, target subcollection: %v", s.currentUser, s.currentUser.targetCollection.name, s.currentUser.targetSubCollection)
	navState := newNavState(config.root, config.root, s.getCollectionTags)
	s.navState = navState
	s.navState.updateChoices()
	return &s
}

func (s *server) setRoot(path string) {
	s.navState.root = path
	s.navState.currentDir = path
	s.navState.updateChoices()
	s.currentUser.root = path
	s.updateRootInDb(path)
}

// Set the current user's auto audition preference and update in db
func (s *server) updateAutoAudition(autoAudition bool) {
	s.currentUser.autoAudition = autoAudition
	s.updateAutoAuditionInDb(autoAudition)
}

func (s *server) updateChoices() {
	s.navState.updateChoices()
}

// Set the current user's target collection and update in db
func (s *server) updateTargetCollection(collection collection) {
	s.currentUser.targetCollection = &collection
	s.updateSelectedCollectionInDb(collection.id)
	s.updateTargetSubCollection("")
	s.currentUser.targetSubCollection = ""
}

// Set the current user's target subcollection and update in db
func (s *server) updateTargetSubCollection(subCollection string) {
	if len(subCollection) > 0 && !strings.HasPrefix(subCollection, "/") {
		subCollection = "/" + subCollection
	}
	s.currentUser.targetSubCollection = subCollection
	s.updateTargetSubCollectionInDb(subCollection)
}

// Create a tag with the defaults based on the current state
func (s *server) createQuickTag(filepath string) {
	s.createCollectionTagInDb(filepath, s.currentUser.targetCollection.id, path.Base(filepath), s.currentUser.targetSubCollection)
	s.updateChoices()
}

// Create a tag with all possible args
func (s *server) createTag(filepath string, name string, subCollection string) {
	s.createCollectionTagInDb(filepath, s.currentUser.targetCollection.id, name, subCollection)
	s.updateChoices()
}

func containsAllSubstrings(s1 string, s2 string) bool {
	words := strings.Fields(s2)
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	for _, word := range words {
		if !strings.Contains(s1, word) {
			return false
		}
	}
	return true
}

// Standard function for getting the necessary files from a dir with their associated tags
func (s *server) fuzzyFind(search string, fromRoot bool) []selectableListItem {
	log.Println("in server fuzzy search fn")
	var dir string
	var entries []os.DirEntry = make([]os.DirEntry, 0)
	var files []fs.DirEntry
	var samples []selectableListItem
	if len(search) == 0 {
		return make([]selectableListItem, 0)
	}
	if fromRoot {
		dir = s.navState.root
	} else {
		dir = s.navState.currentDir
	}
	collectionTags := s.fuzzySearchCollectionTags(search)
	log.Println("collection tags", collectionTags)
	log.Println("searching for: ", search)
	log.Println("dir: ", dir)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !containsAllSubstrings(path, search) || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".asd") {
			return nil
		}
		if (strings.HasSuffix(path, ".wav") || strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".flac")) && !d.IsDir() {
			entries = append(entries, d)
		}
		files = append(files, d)
		matchedTags := make([]collectionTag, 0)
		for _, tag := range collectionTags {
			if strings.Contains(tag.filePath, path) {
				matchedTags = append(matchedTags, tag)
			}
		}
		s.navState.pushChoice(dirEntryWithTags{path: path, tags: matchedTags, isDir: false})
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	log.Println("files: ", len(files))
	return samples
}

// ////////////////////// DATABASE ENDPOINTS ////////////////////////

// Get collection tags associated with a directory
func (s *server) getCollectionTags(dir string) []collectionTag {
	statement := `select t.file_path, col.name, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where t.file_path like ?`
	dir = dir + "%"
	rows, err := s.db.Query(statement, dir)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]collectionTag, 0)
	for rows.Next() {
		var filePath, collectionName, subCollection string
		if err := rows.Scan(&filePath, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		tags = append(tags, collectionTag{filePath: filePath, collectionName: collectionName, subCollection: subCollection})
	}
	return tags
}

func (s *server) fuzzySearchCollectionTags(search string) []collectionTag {
	words := strings.Fields(search)
	if len(words) == 0 {
		return make([]collectionTag, 0)
	} else if len(words) == 1 {
		search = "%" + search + "%"
	} else {
		searchBuilder := ""
		for i, word := range words {
			if i == 0 {
				searchBuilder = "%" + word + "%"
			} else {
				searchBuilder = searchBuilder + " and t.file_path like %" + word + "%"
			}
		}
		search = searchBuilder
	}
	statement := `select t.file_path, col.name, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where t.file_path like ?`
	rows, err := s.db.Query(statement, search)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]collectionTag, 0)
	log.Println("collection tags")
	for rows.Next() {
		var filePath, collectionName, subCollection string
		if err := rows.Scan(&filePath, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		log.Printf("filepath: %s, collection name: %s, subcollection: %s", filePath, collectionName, subCollection)
		tags = append(tags, collectionTag{filePath: filePath, collectionName: collectionName, subCollection: subCollection})
	}
	return tags
}

// Get collection tags associated with a directory
func (s *server) searchCollectionTags(search string) []collectionTag {
	statement := `select t.file_path, col.name, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where t.file_path like ?`
	search = "%" + search + "%"
	rows, err := s.db.Query(statement, search)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]collectionTag, 0)
	log.Println("collection tags")
	for rows.Next() {
		var filePath, collectionName, subCollection string
		if err := rows.Scan(&filePath, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		log.Printf("filepath: %s, collection name: %s, subcollection: %s", filePath, collectionName, subCollection)
		tags = append(tags, collectionTag{filePath: filePath, collectionName: collectionName, subCollection: subCollection})
	}
	return tags
}

func (s *server) getUser(id int) user {
	statement := `select u.name as user_name, c.id as collection_id, c.name as collection_name, c.description, u.auto_audition, u.selected_subcollection, u.root from User u left join Collection c on u.selected_collection = c.id where u.id = ?`
    row := s.db.QueryRow(statement, id)
    var name string
    var collectionId *int
    var collectionName *string
    var collectionDescription *string
    var autoAudition bool
    var selectedSubCollection string
    var root string
    if err := row.Scan(&name, &collectionId, &collectionName, &collectionDescription, &autoAudition, &selectedSubCollection, &root); err != nil {
        log.Fatalf("Failed to scan row: %v", err)
    }
    var selectedCollection *collection
    if collectionId != nil && collectionName != nil && collectionDescription != nil {
        selectedCollection = &collection{id: *collectionId, name: *collectionName, description: *collectionDescription}
    } else {
        selectedCollection = &collection{id: 0, name: "", description: ""}
    }
    return user{id: id, name: name, autoAudition: autoAudition, targetCollection: selectedCollection, targetSubCollection: selectedSubCollection, root: root}
}

// Get all users
func (s *server) getUsers(name *string) []user {
	var whereClause string
	var rows *sql.Rows
	var err error
	if name != nil {
		whereClause = "where u.name = ?"
	}
	statement := `select u.id as user_id, u.name as user_name, c.id as collection_id, c.name as collection_name, c.description, u.auto_audition, u.selected_subcollection, u.root from User u left join Collection c on u.selected_collection = c.id`
	if whereClause != "" {
		statement = statement + " " + whereClause
		rows, err = s.db.Query(statement, name)
	} else {
		rows, err = s.db.Query(statement)
	}
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getUsers: %v", err)
	}
	defer rows.Close()
	users := make([]user, 0)
	for rows.Next() {
		var id int
		var name string
		var collectionId *int
		var collectionName *string
		var collectionDescription *string
		var autoAudition bool
		var selectedSubCollection string
		var root string
		if err := rows.Scan(&id, &name, &collectionId, &collectionName, &collectionDescription, &autoAudition, &selectedSubCollection, &root); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		var selectedCollection *collection
		if collectionId != nil && collectionName != nil && collectionDescription != nil {
			selectedCollection = &collection{id: *collectionId, name: *collectionName, description: *collectionDescription}
		} else {
			selectedCollection = &collection{id: 0, name: "", description: ""}
		}
		users = append(users, user{id: id, name: name, autoAudition: autoAudition, targetCollection: selectedCollection, targetSubCollection: selectedSubCollection, root: root})
	}
	return users
}

// Create a user in the database
func (s *server) createUser(name string) int {
	res, err := s.db.Exec("insert or ignore into User (name) values (?)", name)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in createUser: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

// Update the current user's selected collection in the database
func (s *server) updateSelectedCollectionInDb(collection int) {
	_, err := s.db.Exec("update User set selected_collection = ? where id = ?", collection, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateSelectedCollectionInDb: %v", err)
	}
}

// Update the current user's auto audition preference in the database
func (s *server) updateRootInDb(path string) {
	_, err := s.db.Exec("update User set root = ? where id = ?", path, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in update root in db: %v", err)
	}
}

// Update the current user's auto audition preference in the database
func (s *server) updateAutoAuditionInDb(autoAudition bool) {
	_, err := s.db.Exec("update User set auto_audition = ? where id = ?", autoAudition, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateAutoAuditionInDb: %v", err)
	}
}

// Update the current user's name in the database
func (s *server) updateUsername(id int, name string) {
	_, err := s.db.Exec("update User set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateUsername: %v", err)
	}
}

// Create a collection in the database
func (s *server) createCollection(name string, description string) int {
	var err error
	var res sql.Result
	res, err = s.db.Exec("insert into Collection (name, user_id, description) values (?, ?, ?)", name, s.currentUser.id, description)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in createCollection: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

// A collection
type collection struct {
	id          int
	name        string
	description string
}

// Requirement for a listSelectionItem
func (c collection) Id() int {
	return c.id
}

// Requirement for a listSelectionItem
func (c collection) Name() string {
	return c.name
}

// Requirement for a listSelectionItem
func (c collection) Description() string {
	return c.description
}

// Requirement for a listSelectionItem
func (c collection) IsDir() bool {
	return false
}

// Requirement for a listSelectionItem
func (c collection) IsFile() bool {
	return false
}

// Get all collections for the current user
func (s *server) getCollections() []collection {
	statement := `select id, name, description from Collection where user_id = ?`
	rows, err := s.db.Query(statement, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getCollections: %v", err)
	}
	defer rows.Close()
	collections := make([]collection, 0)
	for rows.Next() {
		var id int
		var name string
		var description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		collection := collection{id: id, name: name, description: description}
		collections = append(collections, collection)
	}
	return collections
}

// Update a collection's name in the database
func (s *server) updateCollectionNameInDb(id int, name string) {
	_, err := s.db.Exec("update Collection set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateCollectionNameInDb: %v", err)
	}
}

// Requirement for a listSelectionItem
func (s *server) updateCollectionDescriptionInDb(id int, description string) {
	_, err := s.db.Exec("update Collection set description = ? where id = ?", description, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateCollectionDescriptionInDb: %v", err)
	}
}

// Create a tag in the database
func (s *server) createTagInDb(filePath string) int {
	if !strings.Contains(filePath, s.navState.root) {
		filePath = filepath.Join(s.navState.currentDir, filePath)
	}
	res, err := s.db.Exec("insert or ignore into Tag (file_path, user_id) values (?, ?)", filePath, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in createTagInDb: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

// Add a tag to a collection in the database
func (s *server) addTagToCollectionInDb(tagId int, collectionId int, name string, subCollection string) {
	log.Printf("Tag id: %d, collectionId: %d, name: %s, subCollection: %s", tagId, collectionId, name, subCollection)
	res, err := s.db.Exec("insert or ignore into CollectionTag (tag_id, collection_id, name, sub_collection) values (?, ?, ?, ?)", tagId, collectionId, name, subCollection)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in addTagToCollectionInDb: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	log.Printf("Collection tag insert ID: %d", id)
}

// Add a CollectionTag to the database, handling creation of core tag if needed
func (s *server) createCollectionTagInDb(filePath string, collectionId int, name string, subCollection string) {
	tagId := s.createTagInDb(filePath)
	log.Printf("Tag id: %d", tagId)
	s.addTagToCollectionInDb(tagId, collectionId, name, subCollection)
}

func (s *server) updateTargetSubCollectionInDb(subCollection string) {
	_, err := s.db.Exec("update User set selected_subcollection = ? where id = ?", subCollection, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateSubCollectionInDb: %v", err)
	}
}

type SubCollection struct {
	name string
}

func (s SubCollection) Id() int {
	return 0
}

func (s SubCollection) Name() string {
	return s.name
}

func (s SubCollection) Description() string {
	return ""
}

func (s SubCollection) IsDir() bool {
	return false
}

func (s SubCollection) IsFile() bool {
	return false
}

func (s *server) getCollectionSubcollections() []SubCollection {
	statement := `select distinct sub_collection from CollectionTag where collection_id = ? order by sub_collection asc`
	rows, err := s.db.Query(statement, s.currentUser.targetCollection.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getCollectionSubcollections: %v", err)
	}
	defer rows.Close()
	subCollections := make([]SubCollection, 0)
	for rows.Next() {
		var subCollection string
		if err := rows.Scan(&subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		subCollections = append(subCollections, SubCollection{name: subCollection})
	}
	return subCollections
}

func (s *server) searchCollectionSubcollections(search string) []SubCollection {
	fuzzySearch := "%" + search + "%"
	statement := `SELECT DISTINCT sub_collection
                  FROM CollectionTag
                  WHERE collection_id = ? AND sub_collection LIKE ?
                  ORDER BY sub_collection ASC`
	rows, err := s.db.Query(statement, s.currentUser.targetCollection.id, fuzzySearch)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in searchCollectionSubcollections: %v", err)
	}
	defer rows.Close()
	subCollections := make([]SubCollection, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		subCollection := SubCollection{name: name}
		subCollections = append(subCollections, subCollection)
	}
	log.Printf("subcollections from db : %v", subCollections)
	return subCollections
}

// ////////////////////// AUDIO HANDLING ////////////////////////

// Audio file type enum
type audioFileType int

// Audio file type enum values
const (
	MP3 audioFileType = iota
	WAV
	FLAC
)

// String representation of an audio file type
func (a *audioFileType) String() string {
	return [...]string{"mp3", "wav", "flac"}[*a]
}

// Construct an audio file type from a string
func (a *audioFileType) fromExtension(s string) {
	switch s {
	case ".mp3":
		*a = MP3
	case ".wav":
		*a = WAV
	case ".flac":
		*a = FLAC
	default:
		log.Fatalf("Unsupported audio file type: %v", s)
	}
}

// Audio player struct
type AudioPlayer struct {
	format          beep.Format
	currentStreamer beep.StreamSeekCloser
	commands        chan string
	playing         bool
	nextCommand     *string
}

// Push a play command to the audio player's commands channel
func (a *AudioPlayer) pushPlayCommand(path string) {
	log.Println("Pushing play command", path)
	a.nextCommand = &path
	a.commands <- path
}

// Construct the audio player
func NewAudioPlayer() *AudioPlayer {
	sampleRate := beep.SampleRate(48000)
	format := beep.Format{SampleRate: sampleRate, NumChannels: 2, Precision: 4}
	player := AudioPlayer{
		format:   format,
		playing:  false,
		commands: make(chan string),
	}
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	go func() {
		player.Run()
	}()
	return &player
}

// Close the audio player
func (a *AudioPlayer) Close() {
	speaker.Lock()
	if a.currentStreamer != nil {
		a.currentStreamer.Close()
	}
	speaker.Unlock()
	speaker.Close()
}

// Get a streamer which will buffer playback of one file
func (a *AudioPlayer) GetStreamer(path string, f *os.File) (beep.StreamSeekCloser, beep.Format, error) {
	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	switch filepath.Ext(path) {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	case ".flac":
		streamer, format, err = flac.Decode(f)
	}
	if err != nil {
		log.Print(err)
		return nil, format, err
	}
	return streamer, format, nil
}

// Close the current streamer
func (a *AudioPlayer) CloseStreamer() {
	if a.currentStreamer != nil {
		a.currentStreamer.Close()
	}
	a.currentStreamer = nil
}

// Handle a play command arriving in the audio player's commands channel
func (a *AudioPlayer) handlePlayCommand(path string) {
	log.Println("Handling play command", path)
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("error opening file ", err)
	}
	defer f.Close()
	a.playing = true
	streamer, format, err := a.GetStreamer(path, f)
	if err != nil {
		log.Printf("Failed to get streamer: %v", err)
		return
	}
	log.Printf("Playing file: \n--> path %s\n--> format%v", path, format)
	a.currentStreamer = streamer
	defer a.CloseStreamer()
	resampled := beep.Resample(4, format.SampleRate, a.format.SampleRate, streamer)
	done := make(chan bool)
	speaker.Play(beep.Seq(resampled, beep.Callback(func() {
		a.playing = false
		log.Println("Finished playing audio")
		if a.nextCommand != nil && *a.nextCommand == path {
			a.nextCommand = nil
		}
		done <- true
	})))
	<-done
}

// Run the audio player, feeding it paths as play commands
func (a *AudioPlayer) Run() {
	for {
		select {
		case path := <-a.commands:
			log.Println("In Run, received play command", path)
			if a.nextCommand != nil && *a.nextCommand != path {
				continue
			}
			a.handlePlayCommand(path)
		}
	}
}

// Play one audio file. If another file is already playing, close the current streamer and play the new file.
func (a *AudioPlayer) PlayAudioFile(path string) {
	if a.playing {
		// Close current streamer with any necessary cleanup
		a.CloseStreamer()
	}
	a.pushPlayCommand(path)
}

// ////////////////////// APP ////////////////////////
type App struct {
	server         *server
	bubbleTeaModel model
	logFile        *os.File
}

// Construct the app
func NewApp() App {
    path := utils.ExpandHomeDir("~/.excavator-tui")
    createDirectories(path)
	f, err := os.OpenFile(filepath.Join(path, "logfile"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	audioPlayer := NewAudioPlayer()
	server := newServer(audioPlayer)
	return App{
		server:         server,
		bubbleTeaModel: excavatorModel(server),
		logFile:        f,
	}
}

// chris_brown_run_it.ogg
func main() {
	app := NewApp()
	defer app.logFile.Close()
	defer app.server.audioPlayer.Close()
	defer app.server.db.Close()
	p := tea.NewProgram(
		app.bubbleTeaModel,
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	if err != nil {
		log.Fatalf("Failed to run program: %v", err)
	}
}
