package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jesses-code-adventures/excavator/src/utils"

	_ "github.com/mattn/go-sqlite3"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"

	// "github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

//////////////////////// UI ////////////////////////

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
	viewportStyle = lipgloss.NewStyle()
	// Padding(1, 1).
)

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
	Up            key.Binding
	Down          key.Binding
	Quit          key.Binding
	JumpUp        key.Binding
	JumpDown      key.Binding
	JumpBottom    key.Binding
	Audition      key.Binding
	Enter         key.Binding
	NewCollection key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Audition, k.Up, k.Down, k.JumpUp, k.JumpDown, k.NewCollection}
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
		key.WithKeys("q", "ctrl+c"),
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
	Audition: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "audition sample"),
	),
	NewCollection: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "new collection"),
	),
}

type model struct {
	ready        bool
	quitting     bool
	inFormWindow bool
	cursor       int
	prevCursor   int
	formFocus    int
	keys         KeyMap
	keyHack      keymapHacks
	server       *Server
	viewport     viewport.Model
	help         help.Model
	inputs       []formInput
}

func initialModel(server *Server) model {
	return model{
		ready:    false,
		quitting: false,
		server:   server,
		help:     help.New(),
		keys:     DefaultKeyMap,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

// Get the header of the viewport
func (m model) headerView() string {
	title := titleStyle.Render("Excavator - Samples")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

// Get the footer of the view
func (m model) footerView() string {
	return m.help.View(m.keys)
}

type formInput struct {
    name        string
    input      textinput.Model
}

func newFormInput(name string) formInput {
    return formInput{
        name: name,
        input: textinput.New(),
    }
}

func getNewCollectionInputs() []formInput {
    return []formInput{
        newFormInput("Name"),
        newFormInput("Description"),
    }
}

// Handle a single key press
func (m model) handleKey(msg tea.KeyMsg) model {
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
			m.server.audioPlayer.PlayAudioFile(filepath.Join(m.server.currentDir, choice.path))
		}
	case key.Matches(msg, m.keys.JumpBottom):
		m.viewport.GotoBottom()
		m.cursor = len(m.server.choices) - 1
	case key.Matches(msg, m.keys.NewCollection):
		if !m.inFormWindow {
			m.inFormWindow = true
			m.inputs = append(m.inputs, getNewCollectionInputs()...)
			m.formFocus = 0
		} else {
			m.inFormWindow = false
			m.inputs = make([]formInput, 0)
		}
	case key.Matches(msg, m.keys.Enter):
        if m.inFormWindow {
            if m.formFocus == len(m.inputs)-1 {
                // Create the new collection
                description := m.inputs[1].input.Value()
                m.server.createCollection(m.inputs[0].input.Value(), &description)
                m.inFormWindow = false
            } else {
                m.formFocus++
            }
        } else {
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

// Takes a message and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		if !m.ready {
			// Handles waiting for the window to instantiate so the viewport can be created
			m.viewport = viewport.New(msg.Width, (msg.Height)-verticalMarginHeight)
			m.viewport.SetContent(m.getContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	case tea.KeyMsg:
		m = m.handleKey(msg)
		if m.quitting {
			return m, tea.Quit
		}
		m.keyHack.updateLastKey(msg.String())
	}
	if m.inFormWindow {
		for i, input := range m.inputs {
			var newInput textinput.Model
			newInput, cmd = input.input.Update(msg)
			m.inputs[i].input = newInput
			if i == m.formFocus {
				m.inputs[i].input.Focus()
			}
		}
	}
	m.viewport.SetContent(m.getContent())
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) getContent() string {
	if m.inFormWindow {
		return m.formView()
	} else {
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
}

func (m model) formView() string {
	s := ""
	for _, input := range m.inputs {
		s += fmt.Sprintf("%v: %v\n", input.name, input.input.View())
	}
	return s
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return appStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), viewportStyle.Render(m.viewport.View()), m.footerView()))
}

//////////////////////// SERVER ////////////////////////

// Struct holding the app's configuration
type Config struct {
	data              string
	root              string
	dbFileName        string
	createSqlCommands []byte
}

// Constructor for the Config struct
func newConfig(data string, samples string, dbFileName string) *Config {
	data = utils.ExpandHomeDir(data)
	samples = utils.ExpandHomeDir(samples)
	sqlCommands, err := os.ReadFile("src/sql_commands/create_db.sql")
	if err != nil {
		log.Fatalf("Failed to read SQL commands: %v", err)
	}
	config := Config{
		data:              data,
		root:              samples,
		dbFileName:        dbFileName,
		createSqlCommands: sqlCommands,
	}
	return &config
}

// Handles either creating or checking the existence of the data and samples directories
func (c *Config) handleDirectories() {
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
func (c *Config) getDbPath() string {
	if c.dbFileName == "" {
		c.dbFileName = "excavator"
	}
	if !strings.HasSuffix(c.dbFileName, ".db") {
		c.dbFileName = c.dbFileName + ".db"
	}
	return filepath.Join(c.data, c.dbFileName)
}

// The main struct holding the server
type Server struct {
	db          *sql.DB
	root        string
	currentDir  string
	currentUser User
	choices     []dirEntryWithTags
	audioPlayer *AudioPlayer
}

// Construct the server
func NewServer(audioPlayer *AudioPlayer) *Server {
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
	s := Server{
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
	s._updateChoices()
	return &s
}

func (s *Server) _updateChoices() {
	if s.currentDir != s.root {
		s.choices = make([]dirEntryWithTags, 0)
		dirEntries := s.listDirEntries()
		s.choices = append(s.choices, dirEntryWithTags{path: "..", tags: make([]collectionTag, 0), isDir: true})
		s.choices = append(s.choices, dirEntries...)
	} else {
		s.choices = s.listDirEntries()
	}
}

func (s *Server) getWholeCurrentDir() string {
	return filepath.Join(s.root, s.currentDir)
}

func (s *Server) changeDir(dir string) {
	log.Println("Changing to dir: ", dir)
	s.currentDir = filepath.Join(s.currentDir, dir)
	log.Println("Current dir: ", s.currentDir)
	s._updateChoices()
}

func (s *Server) changeToRoot() {
	s.currentDir = s.root
	s._updateChoices()
}

func (s *Server) changeToParentDir() {
	log.Println("Changing to dir: ", filepath.Dir(s.currentDir))
	s.currentDir = filepath.Dir(s.currentDir)
	s._updateChoices()
}

type collectionTag struct {
	filePath       string
	collectionName string
	subCollection  string
}

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

func (s *Server) filterDirEntries(entries []os.DirEntry) []os.DirEntry {
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

func (s *Server) listDirEntries() []dirEntryWithTags {
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

func (s *Server) getCollectionTags(root string, subDirectory *string) []collectionTag {
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

type User struct {
	id   int
	name string
}

func (s *Server) getUsers() []User {
	statement := `select id, name from User`
	rows, err := s.db.Query(statement)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		users = append(users, User{id: id, name: name})
	}
	return users
}

func (s *Server) createUser(name string) int {
	res, err := s.db.Exec("insert ignore into User (name) values (?)", name)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

func (s *Server) updateUsername(id int, name string) {
	_, err := s.db.Exec("update User set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
}

func (s *Server) createCollection(name string,description *string) int {
	var err error
	var res sql.Result
	if description == nil {
		res, err = s.db.Exec("insert into Collection (name, user_id) values (?, ?)", name, s.currentUser.id)
	} else {
		res, err = s.db.Exec("insert into Collection (name, user_id, description) values (?, ?, ?)", name, s.currentUser.id, *description)
	}
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

func (s *Server) updateCollectionName(id int, name string) {
	_, err := s.db.Exec("update Collection set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
}

func (s *Server) updateCollectionDescription(id int, description string) {
	_, err := s.db.Exec("update Collection set description = ? where id = ?", description, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
}

func (s *Server) createTag(filePath string) int {
	res, err := s.db.Exec("insert ignore into Tag (file_path) values (?)", filePath)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

func (s *Server) addTagToCollection(tagId int, collectionId int, name string, subCollection *string) {
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

type App struct {
	server         *Server
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
	server := NewServer(audioPlayer)
	return App{
		server:         server,
		bubbleTeaModel: initialModel(server),
		logFile:        f,
	}
}

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
	format beep.Format
	// Add a mutex for safe access to the currently playing streamer
	mutex sync.Mutex
	// Track the current playing streamer for stopping if needed
	currentStreamer beep.StreamSeekCloser
}

func NewAudioPlayer() *AudioPlayer {
	sampleRate := beep.SampleRate(48000)
	format := beep.Format{SampleRate: sampleRate, NumChannels: 2, Precision: 4}
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	// speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	return &AudioPlayer{format: format}

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
		log.Fatal(err)
	}
	return streamer, format, nil
}

func (a *AudioPlayer) PlayAudioFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	done := make(chan bool)
	streamer, format, err := a.GetStreamer(path, f)
	defer streamer.Close()
	resampled := beep.Resample(4, format.SampleRate, a.format.SampleRate, streamer)
	speaker.Play(beep.Seq(resampled, beep.Callback(func() {
		done <- true
	})))
	<-done
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
