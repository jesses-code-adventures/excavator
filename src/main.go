package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
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

func (k *keymapHacks) updateLastKey(key string) {
	k.lastKey = key
}

func (k *keymapHacks) getLastKey() string {
	return k.lastKey
}

func (k *keymapHacks) lastKeyWasG() bool {
	return k.lastKey == "g"
}

type KeyMap struct {
	Up               key.Binding
	Down             key.Binding
	Quit             key.Binding
	JumpUp           key.Binding
	JumpDown         key.Binding
	JumpBottom       key.Binding
	Audition         key.Binding
	Enter            key.Binding
	NewCollection    key.Binding
	SelectCollection key.Binding
	InsertMode       key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Up, k.Down, k.JumpUp, k.JumpDown, k.Audition, k.NewCollection, k.SelectCollection}
}

// Short help only
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{},
	}
}

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
	SelectCollection: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "select collection"),
	),
}

// ////////////////////// UI ////////////////////////
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

// / Form ///
type formInput struct {
	name  string
	input textinput.Model
}

func newFormInput(name string) formInput {
	return formInput{
		name:  name,
		input: textinput.New(),
	}
}

func getNewCollectionInputs() []formInput {
	return []formInput{
		newFormInput("name"),
		newFormInput("description"),
	}
}

func getNewCollectionForm() form {
	return newForm("create collection", getNewCollectionInputs())
}

type form struct {
	title        string
	inputs       []formInput
	writing      bool
	focusedInput int
}

func newForm(title string, inputs []formInput) form {
	return form{
		title:        title,
		inputs:       inputs,
		writing:      false,
		focusedInput: 0,
	}
}

/// List selection ///

type listSelectionItem interface {
	Id() int
	Name() string
	Description() string
}

type listSelection struct {
	title    string
	items    []listSelectionItem
	selected int
}

type model struct {
	ready         bool
	quitting      bool
	cursor        int
	prevCursor    int
	keys          KeyMap
	keyHack       keymapHacks
	server        *server
	viewport      viewport.Model
	help          help.Model
	windowType    windowType
	form          form
	listSelection listSelection
}

type windowType int

const (
	DirectoryWalker windowType = iota
	FormWindow
	ListSelectionWindow
)

func initialModel(server *server) model {
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

func (m model) handleListSelectionKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.listSelection.selected++
		if m.listSelection.selected >= len(m.listSelection.items) {
			m.listSelection.selected = 0
		}
	case key.Matches(msg, m.keys.Down):
		m.listSelection.selected--
		if m.listSelection.selected < 0 {
			m.listSelection.selected = len(m.listSelection.items) - 1
		}
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.NewCollection):
		m.windowType = FormWindow
		m.listSelection.items = make([]listSelectionItem, 0)
		m.form = getNewCollectionForm()
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.SelectCollection):
		m.windowType = DirectoryWalker
		m.listSelection.items = make([]listSelectionItem, 0)
		m.server._updateChoices()
	case key.Matches(msg, m.keys.Enter):
		switch m.listSelection.title {
		case "select collection":
			if collection, ok := m.listSelection.items[m.listSelection.selected].(collection); ok {
				m.server.updateSelectedCollection(collection)
				m.listSelection.items = make([]listSelectionItem, 0)
				m.windowType = DirectoryWalker
			} else {
				log.Fatalf("Invalid list selection item type")
			}
		}
	}
	return m, cmd
}

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
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.NewCollection):
		m.windowType = DirectoryWalker
		m.form.inputs = make([]formInput, 0)
		m.server._updateChoices()
	case key.Matches(msg, m.keys.SelectCollection):
		collections := m.server.getCollections()
		m.listSelection = listSelection{title: "select collection", items: make([]listSelectionItem, 0), selected: 0}
		for _, collection := range collections {
			m.listSelection.items = append(m.listSelection.items, collection)
		}
		m.windowType = ListSelectionWindow
	case key.Matches(msg, m.keys.Enter):
		for i, input := range m.form.inputs {
			if input.input.Value() == "" {
				m.form.focusedInput = i
				m.form.inputs[i].input.Focus()
				break
			}
		}
		switch m.form.title {
		case "create collection":
			m.server.createCollection(m.form.inputs[0].input.Value(), m.form.inputs[1].input.Value())
			m.form.inputs = make([]formInput, 0)
			m.windowType = DirectoryWalker
		}
	}
	return m, cmd
}

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

func (m model) handleFormKey(msg tea.KeyMsg, cmd tea.Cmd) (model, tea.Cmd) {
	if m.form.writing {
		m, cmd = m.handleFormWritingKey(msg, cmd)
	} else {
		m, cmd = m.handleFormNavigationKey(msg, cmd)
	}
	return m, cmd
}

// Handle a single key press
func (m model) handleContentKey(msg tea.KeyMsg) model {
    // if m.server.currentUser.selectedCollection != nil {
    //     log.Printf("Handling content key user id: %s selected collection: %s", m.server.currentUser.name, m.server.currentUser.selectedCollection.name)
    //
    // } else {
    //     log.Printf("Handling content key user id: %s", m.server.currentUser.name)
    // }
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.server.choices)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.JumpDown):
		if m.cursor < len(m.server.choices)-8 {
			m.cursor += 8
		} else {
			m.cursor = len(m.server.choices) - 1
		}
	case key.Matches(msg, m.keys.JumpUp):
		if m.cursor > 8 {
			m.cursor -= 8
		} else {
			m.cursor = 0
		}
	case key.Matches(msg, m.keys.Audition):
		choice := m.server.choices[m.cursor]
		if !choice.isDir {
			go m.server.audioPlayer.PlayAudioFile(filepath.Join(m.server.currentDir, choice.path))
		}
	case key.Matches(msg, m.keys.JumpBottom):
		m.viewport.GotoBottom()
		m.cursor = len(m.server.choices) - 1
	case key.Matches(msg, m.keys.NewCollection):
		m.form = getNewCollectionForm()
		m.windowType = FormWindow
	case key.Matches(msg, m.keys.SelectCollection):
		collections := m.server.getCollections()
		m.listSelection = listSelection{title: "select collection", items: make([]listSelectionItem, 0), selected: 0}
		log.Println("got collections ", collections)
		for _, collection := range collections {
			m.listSelection.items = append(m.listSelection.items, collection)
		}
		m.windowType = ListSelectionWindow
	case key.Matches(msg, m.keys.Enter):
		choice := m.server.choices[m.cursor]
		if choice.isDir {
			if choice.path == ".." {
				m.cursor = 0
				m.viewport.GotoTop()
				m.server.changeToParentDir()
			} else {
				m.cursor = 0
				m.viewport.GotoTop()
				m.server.changeDir(choice.path)
			}
		} else {
			m.server.audioPlayer.PlayAudioFile(filepath.Join(m.server.currentDir, choice.path))
		}
	default:
		switch msg.String() {
		case "g":
			if m.keyHack.getLastKey() == "g" {
				m.viewport.GotoTop()
				m.cursor = 0
			}
		}
	}
	return m
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
	for i, choice := range m.listSelection.items {
		if i == m.listSelection.selected {
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
		if m.cursor == i {
			cursor := "-->"
			s += selectedStyle.Render(fmt.Sprintf("%s %s    %v", cursor, choice.path, choice.displayTags()), fmt.Sprintf("    %v", choice.displayTags()))
		} else {
			s += unselectedStyle.Render(fmt.Sprintf("     %s", choice.path))
		}
	}
	return s
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) handleWindowResize(msg tea.WindowSizeMsg) model {
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := headerHeight + footerHeight
	if !m.ready {
		// Handles waiting for the window to instantiate so the viewport can be created
		m.viewport = viewport.New(msg.Width, (msg.Height)-verticalMarginHeight)
		m.viewport.SetContent(m.directoryView())
		m.ready = true
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMarginHeight
	}
	return m
}

func (m model) setViewportContent(msg tea.Msg, cmd tea.Cmd) (model, tea.Cmd) {
	switch m.windowType {
	case FormWindow:
		m.viewport.SetContent(m.formView())
	case DirectoryWalker:
		m.viewport.SetContent(m.directoryView())
	case ListSelectionWindow:
		m.viewport.SetContent(m.listSelectionView())
	default:
		m.viewport.SetContent("Invalid window type")
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// Takes a message and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.handleWindowResize(msg)
	case tea.KeyMsg:
		// Making this a switch so it's easy if we change to more window types later
		switch m.windowType {
		case FormWindow:
			m, cmd = m.handleFormKey(msg, cmd)
		case ListSelectionWindow:
			m, cmd = m.handleListSelectionKey(msg, cmd)
		case DirectoryWalker:
			m = m.handleContentKey(msg)
			if m.quitting {
				return m, tea.Quit
			}
		}
		m.keyHack.updateLastKey(msg.String())
	}
	return m.setViewportContent(msg, cmd)
}

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

type user struct {
	id                 int
	name               string
	selectedCollection *collection
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
    log.Printf("Current user: %v, selected collection: %v", s.currentUser, s.currentUser.selectedCollection.name)
	s._updateChoices()
	return &s
}

func (s *server) _updateChoices() {
	if s.currentDir != s.root {
		s.choices = make([]dirEntryWithTags, 0)
		dirEntries := s.listDirEntries()
		s.choices = append(s.choices, dirEntryWithTags{path: "..", tags: make([]collectionTag, 0), isDir: true})
		s.choices = append(s.choices, dirEntries...)
		log.Println("Choices: ", s.choices)
	} else {
		s.choices = s.listDirEntries()
	}
}

func (s *server) updateSelectedCollection(collection collection) {
	s.currentUser.selectedCollection = &collection
	s.updateSelectedCollectionInDb(collection.id)
}

func (s *server) getWholeCurrentDir() string {
	return filepath.Join(s.root, s.currentDir)
}

func (s *server) changeDir(dir string) {
	log.Println("Changing to dir: ", dir)
	s.currentDir = filepath.Join(s.currentDir, dir)
	log.Println("Current dir: ", s.currentDir)
	s._updateChoices()
}

func (s *server) changeToRoot() {
	s.currentDir = s.root
	s._updateChoices()
}

func (s *server) changeToParentDir() {
	log.Println("Changing to dir: ", filepath.Dir(s.currentDir))
	s.currentDir = filepath.Dir(s.currentDir)
	s._updateChoices()
}

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

func (s *server) listDirEntries() []dirEntryWithTags {
	files, err := os.ReadDir(s.currentDir)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	files = s.filterDirEntries(files)
	collectionTags := s.getCollectionTags(s.root, &s.currentDir)
	var samples []dirEntryWithTags
	for _, file := range files {
		matchedTags := make([]collectionTag, 0)
		for _, tag := range collectionTags {
			if strings.Contains(tag.filePath, file.Name()) {
				matchedTags = append(matchedTags, tag)
			}
		}
		isDir := file.IsDir()
		samples = append(samples, dirEntryWithTags{path: file.Name(), tags: matchedTags, isDir: isDir})
	}
	return samples
}

// ////////////////////// DATABASE ENDPOINTS ////////////////////////
func (s *server) getCollectionTags(root string, subDirectory *string) []collectionTag {
	queryDir := root
	if subDirectory != nil {
		queryDir = filepath.Join(root, *subDirectory)
	}
	statement := `select t.file_path, col.name, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where t.file_path like '?%'`
	rows, err := s.db.Query(statement, queryDir)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]collectionTag, 0)
	for rows.Next() {
		var filePath, tagName, subCollection string
		if err := rows.Scan(&filePath, &tagName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		tags = append(tags, collectionTag{filePath: filePath, collectionName: tagName, subCollection: subCollection})
	}
	return tags
}

func (s *server) getUsers() []user {
	statement := `select u.id as user_id, u.name as user_name, c.id as collection_id, c.name as collection_name, c.description from User u left join Collection c on u.selected_collection = c.id`
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
		if err := rows.Scan(&id, &name, &collectionId, &collectionName, &collectionDescription); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		var selectedCollection *collection
		if collectionId != nil && collectionName != nil && collectionDescription != nil {
			selectedCollection = &collection{id: *collectionId, name: *collectionName, description: *collectionDescription}
		} else {
            selectedCollection = &collection{id: 0, name: "", description: ""}
        }
		users = append(users, user{id: id, name: name, selectedCollection: selectedCollection})
	}
	return users
}

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

func (s *server) updateSelectedCollectionInDb(collection int) {
	_, err := s.db.Exec("update User set selected_collection = ? where id = ?", collection, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateSelectedCollectionInDb: %v", err)
	}
}

func (s *server) updateUsername(id int, name string) {
	_, err := s.db.Exec("update User set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateUsername: %v", err)
	}
}

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

type collection struct {
	id          int
	name        string
	description string
}

func (c collection) Id() int {
	return c.id
}

func (c collection) Name() string {
	return c.name
}

func (c collection) Description() string {
	return c.description
}

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

func (s *server) updateCollectionName(id int, name string) {
	_, err := s.db.Exec("update Collection set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
}

func (s *server) updateCollectionDescription(id int, description string) {
	_, err := s.db.Exec("update Collection set description = ? where id = ?", description, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
}

func (s *server) createTag(filePath string) int {
	res, err := s.db.Exec("insert ignore into Tag (file_path, user_id) values (?)", filePath, s.currentUser.id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

func (s *server) addTagToCollection(tagId int, collectionId int, name string, subCollection *string) {
	var err error
	if subCollection == nil {
		_, err = s.db.Exec("insert ignore into CollectionTag (tag_id, collection_id) values (?, ?)", tagId, collectionId)
	} else {
		_, err = s.db.Exec("insert ignore into CollectionTag (tag_id, collection_id, sub_collection) values (?, ?, ?)", tagId, collectionId, *subCollection)
	}
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
}

// ////////////////////// AUDIO HANDLING ////////////////////////
type audioFileType int

const (
	MP3 audioFileType = iota
	WAV
	FLAC
)

func (a *audioFileType) String() string {
	return [...]string{"mp3", "wav", "flac"}[*a]
}

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

type AudioPlayer struct {
	format          beep.Format
	currentStreamer beep.StreamSeekCloser
	commands        chan string
	playing         bool
}

func (a *AudioPlayer) pushPlayCommand(path string) {
	log.Println("Pushing play command", path)
	a.commands <- path
}

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

func (a *AudioPlayer) Close() {
	speaker.Lock()
	if a.currentStreamer != nil {
		a.currentStreamer.Close()
	}
	speaker.Unlock()
	speaker.Close()
}

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

func (a *AudioPlayer) CloseStreamer() {
	if a.currentStreamer != nil {
		a.currentStreamer.Close()
	}
	a.currentStreamer = nil
}

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

func (a *AudioPlayer) Run() {
	for {
		select {
		case path := <-a.commands:
			log.Println("In Run, received play command", path)
			a.handlePlayCommnad(path)
		}
	}

}

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
		bubbleTeaModel: initialModel(server),
		logFile:        f,
	}
}

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
