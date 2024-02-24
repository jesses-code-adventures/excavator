package main

import (
	"database/sql"
	"flag"
	"fmt"
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
	Enter                  key.Binding
	NewCollection          key.Binding
	SelectCollection       key.Binding
	InsertMode             key.Binding
	ToggleAutoAudition     key.Binding
	AuditionRandom         key.Binding
	CreateQuickTag         key.Binding
	CreateTag              key.Binding
	SetTargetSubCollection key.Binding
}

// The actual help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Audition, k.AuditionRandom, k.ToggleAutoAudition, k.NewCollection, k.SelectCollection, k.SetTargetSubCollection, k.CreateQuickTag, k.CreateTag}
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
	SelectCollection: key.NewBinding(
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
}

// ////////////////////// UI ////////////////////////

// All styles to be used throughout the ui
var (
	green    = lipgloss.Color("#25A065")
	pink     = lipgloss.Color("#E441B5")
	white    = lipgloss.Color("#FFFDF5")
	appStyle = lipgloss.NewStyle().
			Padding(1, 1)
	titleStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(green).
			Padding(1, 1)
	selectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder()).
			Foreground(pink)
	unselectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder())
	bottomTextBoxesStyle = lipgloss.NewStyle().
				BorderTop(true).
				Height(8).
				Width(255)
	viewportStyle  = lipgloss.NewStyle()
	unfocusedInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Width(100).
			Margin(1, 1).
			Border(lipgloss.HiddenBorder())
	focusedInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(100).
			Margin(1, 1).
			Background(lipgloss.Color("236"))
	formStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Border(lipgloss.RoundedBorder()).
			Margin(0, 0, 0)
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
func getSubcollectionInput() []formInput {
	return []formInput{
		newFormInput("subcollection"),
	}
}

// Get the new collection form
func getTargetSubCollectionForm() form {
	return newForm("set target subcollection", getSubcollectionInput())
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
}

// A list where a single item can be selected
type selectableList struct {
	title    string
	items    []selectableListItem
	selected int
}

// A generic model defining app behaviour in all states
type model struct {
	ready          bool
	quitting       bool
	cursor         int
	prevCursor     int
	viewportHeight int
	viewportWidth  int
	keys           KeyMap
	keyHack        keymapHacks
	server         *server
	viewport       viewport.Model
	help           help.Model
	windowType     windowType
	form           form
	selectableList selectableList
}

// Different window types
type windowType int

const (
	DirectoryWalker windowType = iota
	FormWindow
	ListSelectionWindow
)

// Constructor for the app's model
func excavatorModel(server *server) model {
	return model{
		ready:    false,
		quitting: false,
		server:   server,
		help:     help.New(),
		keys:     DefaultKeyMap,
	}
}

// Get the header of the viewport
func (m model) headerView() string {
	title := titleStyle.Render("Excavator - Samples")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
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
	return centeredHelpText
}

// Formview handler
func (m model) formView() string {
	s := ""
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
func (m model) listSelectionView() string {
	s := ""
	for i, choice := range m.selectableList.items {
		if i == m.selectableList.selected {
			cursor := "-->"
			s += selectedStyle.Render(fmt.Sprintf("%s %s    %v", cursor, choice.Name(), choice.Description()))
		} else {
			s += unselectedStyle.Render(fmt.Sprintf("     %s", choice.Name()))
		}
	}
	return s
}

// Standard content handler
func (m model) directoryView() string {
	s := ""
	for i, choice := range m.server.choices {
		var newLine string
		if m.cursor == i {
			cursor := "-->"
			newLine = fmt.Sprintf("%s %s", cursor, choice.path)
		} else {
			newLine = fmt.Sprintf("     %s", choice.path)
		}
		if len(newLine) > m.viewport.Width {
			newLine = newLine[:m.viewport.Width-2]
		}
		if m.cursor == i {
			newLine = selectedStyle.Render(newLine, fmt.Sprintf("    %v", choice.displayTags()))
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

// Ui updating for window resize events
func (m model) handleWindowResize(msg tea.WindowSizeMsg) model {
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := headerHeight + footerHeight
	listItemStyleHeight := 2 // TODO: make this better
	if !m.ready {
		// Instantiate a viewport when the program starts
		m.viewportWidth = msg.Width
		m.viewportHeight = (msg.Height - verticalMarginHeight) / listItemStyleHeight
		m.viewport = viewport.New(msg.Width, (msg.Height - verticalMarginHeight))
		m.viewport.SetContent(m.directoryView())
		m.ready = true
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMarginHeight
	}
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
		m.viewport.SetContent(m.listSelectionView())
	default:
		m.viewport.SetContent("Invalid window type")
	}
	return m, cmd
}

// Render the model
func (m model) View() string {
	if m.quitting {
		return ""
	}
	switch m.windowType {
	case FormWindow:
		return appStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), formStyle.Render(m.formView()), m.footerView()))
	case DirectoryWalker:
		return appStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), viewportStyle.Render(m.viewport.View()), m.footerView()))
	case ListSelectionWindow:
		return appStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), viewportStyle.Render(m.viewport.View()), m.footerView()))
	default:
		return "Invalid window type"
	}
}

// ////////////////////// UI UPDATING ////////////////////////

// List selection navigation
func (m model) handleListSelectionKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.selectableList.selected++
		if m.selectableList.selected >= len(m.selectableList.items) {
			m.selectableList.selected = 0
		}
	case key.Matches(msg, m.keys.Down):
		m.selectableList.selected--
		if m.selectableList.selected < 0 {
			m.selectableList.selected = len(m.selectableList.items) - 1
		}
	case key.Matches(msg, m.keys.NewCollection):
		m.selectableList.items = make([]selectableListItem, 0)
		m.form = getNewCollectionForm()
		m.windowType = FormWindow
	case key.Matches(msg, m.keys.SetTargetSubCollection):
		m.selectableList.items = make([]selectableListItem, 0)
		m.form = getTargetSubCollectionForm()
		m.windowType = FormWindow
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.SelectCollection):
		m.selectableList.items = make([]selectableListItem, 0)
		m.server.updateChoices()
		m.windowType = DirectoryWalker
	case key.Matches(msg, m.keys.ToggleAutoAudition):
		m.server.updateAutoAudition(!m.server.currentUser.autoAudition)
	case key.Matches(msg, m.keys.Enter):
		switch m.selectableList.title {
		case "select collection":
			if collection, ok := m.selectableList.items[m.selectableList.selected].(collection); ok {
				m.server.updateTargetCollection(collection)
				m.selectableList.items = make([]selectableListItem, 0)
				m.windowType = DirectoryWalker
			} else {
				log.Fatalf("Invalid list selection item type")
			}
		}
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
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.NewCollection), key.Matches(msg, m.keys.SetTargetSubCollection):
		m.windowType = DirectoryWalker
		m.form.inputs = make([]formInput, 0)
		m.server.updateChoices()
	case key.Matches(msg, m.keys.SelectCollection):
		collections := m.server.getCollections()
		m.selectableList = selectableList{title: "select collection", items: make([]selectableListItem, 0), selected: 0}
		for _, collection := range collections {
			m.selectableList.items = append(m.selectableList.items, collection)
		}
		m.windowType = ListSelectionWindow
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
			m.form.inputs = make([]formInput, 0)
			m.windowType = DirectoryWalker
		case "set target subcollection":
			m.server.updateTargetSubCollection(m.form.inputs[0].input.Value())
			m.form.inputs = make([]formInput, 0)
			m.windowType = DirectoryWalker
		case "create tag":
			m.server.createTag(m.server.choices[m.cursor].path, m.form.inputs[0].input.Value(), m.form.inputs[1].input.Value())
			m.form.inputs = make([]formInput, 0)
			m.windowType = DirectoryWalker
		}
	}
	return m, cmd
}

// Form writing
func (m model) handleFormWritingKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.form.writing = false
		m.form.inputs[m.form.focusedInput].input.Blur()
	case key.Matches(msg, m.keys.Enter):
		m.form.writing = false
		m.form.inputs[m.form.focusedInput].input.Blur()
	default:
		var newInput textinput.Model
		newInput, cmd = m.form.inputs[m.form.focusedInput].input.Update(msg)
		m.form.inputs[m.form.focusedInput].input = newInput
		m.form.inputs[m.form.focusedInput].input.Focus()
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

// Audition the file under the cursor
func (m model) auditionCurrentlySelectedFile() {
	choice := m.server.choices[m.cursor]
	if !choice.isDir {
		go m.server.audioPlayer.PlayAudioFile(filepath.Join(m.server.currentDir, choice.path))
	}
}

// These functions should run every time the cursor moves in directory view
func (m model) dirVerticalNavEffect() {
	if m.server.currentUser.autoAudition {
		m.auditionCurrentlySelectedFile()
	}
}

// Handle a single key press
func (m model) handleDirectoryKey(msg tea.KeyMsg) model {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.server.choices)-1 {
			m.cursor++
		}
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.JumpDown):
		if m.cursor < len(m.server.choices)-8 {
			m.cursor += 8
		} else {
			m.cursor = len(m.server.choices) - 1
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
		fileIndex := m.server.getRandomAudioFileIndex()
		if fileIndex != -1 {
			m.cursor = fileIndex
			m.dirVerticalNavEffect()
		}
		if !m.server.currentUser.autoAudition {
			m.auditionCurrentlySelectedFile()
		}
	case key.Matches(msg, m.keys.JumpBottom):
		m.viewport.GotoBottom()
		m.cursor = len(m.server.choices) - 1
		m.dirVerticalNavEffect()
	case key.Matches(msg, m.keys.NewCollection):
		m.form = getNewCollectionForm()
		m.windowType = FormWindow
	case key.Matches(msg, m.keys.SetTargetSubCollection):
		m.form = getTargetSubCollectionForm()
		m.windowType = FormWindow
	case key.Matches(msg, m.keys.SelectCollection):
		collections := m.server.getCollections()
		m.selectableList = selectableList{title: "select collection", items: make([]selectableListItem, 0), selected: 0}
		for _, collection := range collections {
			m.selectableList.items = append(m.selectableList.items, collection)
		}
		m.windowType = ListSelectionWindow
	case key.Matches(msg, m.keys.ToggleAutoAudition):
		m.server.updateAutoAudition(!m.server.currentUser.autoAudition)
	case key.Matches(msg, m.keys.CreateQuickTag):
		choice := m.server.choices[m.cursor]
		if !choice.isDir {
			m.server.createQuickTag(choice.path)
		}
		m.server.updateChoices()
	case key.Matches(msg, m.keys.CreateTag):
		m.form = getCreateTagForm(path.Base(m.server.choices[m.cursor].path), m.server.currentUser.targetSubCollection)
		m.windowType = FormWindow
	case key.Matches(msg, m.keys.Enter):
		choice := m.server.choices[m.cursor]
		if choice.isDir {
			if choice.path == ".." {
				m.cursor = 0
				m.server.changeToParentDir()
				m.dirVerticalNavEffect()
			} else {
				m.cursor = 0
				m.server.changeDir(choice.path)
				m.dirVerticalNavEffect()
			}
		}
	default:
		switch msg.String() {
		case "g":
			if m.keyHack.getLastKey() == "g" {
				// m.viewport.GotoTop()
				m.cursor = 0
			}
		}
	}
	return m
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
			m = m.handleDirectoryKey(msg)
			if m.quitting {
				return m, tea.Quit
			}
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
}

// Struct holding the app's configuration
type config struct {
	data              string
	root              string
	dbFileName        string
	createSqlCommands []byte
}

// Constructor for the Config struct
func newConfig(data string, samples string, dbFileName string) *config {
	data = utils.ExpandHomeDir(data)
	samples = utils.ExpandHomeDir(samples)
	sqlCommands, err := os.ReadFile("src/sql_commands/create_db.sql")
	if err != nil {
		log.Fatalf("Failed to read SQL commands: %v", err)
	}
	config := config{
		data:              data,
		root:              samples,
		dbFileName:        dbFileName,
		createSqlCommands: sqlCommands,
	}
	return &config
}

// Handles either creating or checking the existence of the data and samples directories
func (c *config) handleDirectories() {
	if _, err := os.Stat(c.data); os.IsNotExist(err) {
		if err := os.MkdirAll(c.data, 0755); err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(c.root); os.IsNotExist(err) {
		log.Fatal("No directory at config's samples directory ", c.root)
	}
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

// The main struct holding the server
type server struct {
	db          *sql.DB
	root        string
	currentDir  string
	currentUser user
	choices     []dirEntryWithTags
	audioPlayer *AudioPlayer
}

// Construct the server
func newServer(audioPlayer *AudioPlayer) *server {
	var data = flag.String("data", "~/.excavator-tui", "Local data storage path")
	var samples = flag.String("samples", "~/Library/Audio/Sounds/Samples", "Root samples directory")
	var dbFileName = flag.String("db", "excavator", "Database file name")
	flag.Parse()
	config := newConfig(*data, *samples, *dbFileName)
	config.handleDirectories()
	dbPath := config.getDbPath()
	dbExists := true
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dbExists = false
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	if !dbExists {
		_, err = db.Exec(string(config.createSqlCommands))
	}
	if err != nil {
		log.Fatalf("Failed to execute SQL commands: %v", err)
	} else {
		log.Print("Database setup successfully.")
	}
	s := server{
		db:          db,
		root:        config.root,
		currentDir:  config.root,
		audioPlayer: audioPlayer,
	}
	users := s.getUsers()
	if len(users) == 0 {
		log.Fatal("No users found")
	}
	s.currentUser = users[0]
	log.Printf("Current user: %v, selected collection: %v, target subcollection: %v", s.currentUser, s.currentUser.targetCollection.name, s.currentUser.targetSubCollection)
	s.updateChoices()
	return &s
}

// Grab an index of some audio file within the current directory
func (s *server) getRandomAudioFileIndex() int {
	if len(s.choices) == 0 {
		return -1
	}
	possibleIndexes := make([]int, 0)
	for i, choice := range s.choices {
		if !choice.isDir {
			possibleIndexes = append(possibleIndexes, i)
		}
	}
	return possibleIndexes[rand.Intn(len(possibleIndexes))]
}

// Populate the choices array with the current directory's contents
func (s *server) updateChoices() {
	if s.currentDir != s.root {
		s.choices = make([]dirEntryWithTags, 0)
		dirEntries := s.listDirEntries()
		s.choices = append(s.choices, dirEntryWithTags{path: "..", tags: make([]collectionTag, 0), isDir: true})
		s.choices = append(s.choices, dirEntries...)
	} else {
		s.choices = s.listDirEntries()
	}
}

// Set the current user's auto audition preference and update in db
func (s *server) updateAutoAudition(autoAudition bool) {
	s.currentUser.autoAudition = autoAudition
	s.updateAutoAuditionInDb(autoAudition)
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

// Get the full path of the current directory
func (s *server) getCurrentDirPath() string {
	return filepath.Join(s.root, s.currentDir)
}

// Change the current directory
func (s *server) changeDir(dir string) {
	log.Println("Changing to dir: ", dir)
	s.currentDir = filepath.Join(s.currentDir, dir)
	log.Println("Current dir: ", s.currentDir)
	s.updateChoices()
}

// Change the current directory to the root
func (s *server) changeToRoot() {
	s.currentDir = s.root
	s.updateChoices()
}

// Change the current directory to the parent directory
func (s *server) changeToParentDir() {
	log.Println("Changing to dir: ", filepath.Dir(s.currentDir))
	s.currentDir = filepath.Dir(s.currentDir)
	s.updateChoices()
}

// Return only directories and valid audio files
func (s *server) filterDirEntries(entries []os.DirEntry) []os.DirEntry {
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
func (s *server) listDirEntries() []dirEntryWithTags {
	files, err := os.ReadDir(s.currentDir)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	files = s.filterDirEntries(files)
	collectionTags := s.getCollectionTags(s.currentDir)
	var samples []dirEntryWithTags
	for _, file := range files {
		matchedTags := make([]collectionTag, 0)
		isDir := file.IsDir()
		if !isDir {
			for _, tag := range collectionTags {
				if strings.Contains(tag.filePath, file.Name()) {
					matchedTags = append(matchedTags, tag)
				}
			}
		}
		samples = append(samples, dirEntryWithTags{path: file.Name(), tags: matchedTags, isDir: isDir})
	}
	return samples
}

// ////////////////////// DATABASE ENDPOINTS ////////////////////////

// Get collection tags associated with a directory
func (s *server) getCollectionTags(dir string) []collectionTag {
	log.Println("get collection tags")
	log.Println("dir: ", dir)
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

// Get all users
func (s *server) getUsers() []user {
	statement := `select u.id as user_id, u.name as user_name, c.id as collection_id, c.name as collection_name, c.description, u.auto_audition, u.selected_subcollection from User u left join Collection c on u.selected_collection = c.id`
	rows, err := s.db.Query(statement)
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
		if err := rows.Scan(&id, &name, &collectionId, &collectionName, &collectionDescription, &autoAudition, &selectedSubCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		var selectedCollection *collection
		if collectionId != nil && collectionName != nil && collectionDescription != nil {
			selectedCollection = &collection{id: *collectionId, name: *collectionName, description: *collectionDescription}
		} else {
			selectedCollection = &collection{id: 0, name: "", description: ""}
		}
		users = append(users, user{id: id, name: name, autoAudition: autoAudition, targetCollection: selectedCollection, targetSubCollection: selectedSubCollection})
	}
	return users
}

// Create a user in the database
func (s *server) createUser(name string) int {
	res, err := s.db.Exec("insert ignore into User (name) values (?)", name)
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
	if !strings.Contains(filePath, s.root) {
		filePath = filepath.Join(s.currentDir, filePath)
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
}

// Push a play command to the audio player's commands channel
func (a *AudioPlayer) pushPlayCommand(path string) {
	log.Println("Pushing play command", path)
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
func (a *AudioPlayer) handlePlayCommnad(path string) {
	log.Println("Handling play command", path)
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
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
			a.handlePlayCommnad(path)
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
	f, err := os.OpenFile("logfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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
