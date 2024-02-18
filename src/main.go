package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jesses-code-adventures/excavator/src/utils"

	_ "github.com/charmbracelet/bubbles/list"
	_ "github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

//////////////////////// UI ////////////////////////

var (
	appStyle   = lipgloss.NewStyle().Padding(0, 1)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(1, 1)
	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"})
)

type model struct {
	choices  []dirEntryWithTags
	cursor   int
	viewport viewport.Model
	ready    bool
}

func initialModel(choices []dirEntryWithTags) model {
	return model{
		ready:   false,
		choices: choices,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) headerView() string {
	title := titleStyle.Render("Excavator - Samples")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func controlView(key string, control string) string {
    return lipgloss.JoinVertical(lipgloss.Center, key, control)
}

func controlsView(width int) string {
	quitControl := controlView("Q", "Quit")
	downControl := controlView("J", "Down")
	upControl := controlView("K", "Up")
    joinedControls := lipgloss.JoinHorizontal(lipgloss.Center, quitControl, downControl, upControl)
    return lipgloss.PlaceHorizontal(width, lipgloss.Center, joinedControls)
}

func (m model) footerView() string {
    line := strings.Repeat("─", max(0, m.viewport.Width))
    controls := controlsView(m.viewport.Width)
	return lipgloss.JoinVertical(lipgloss.Center, line, controls)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Handles waiting for the window to instantiate so the viewport can be created
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.getContent())
			m.viewport.YPosition = headerHeight + 1
            m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	// Controls
	case tea.KeyMsg:

		switch msg.String() {
		// Exit
		case "ctrl+c", "q":
			log.Println("Received exit command", msg.String())
			return m, tea.Quit

		// Navigate up
		case "up", "k":
			log.Println("Received up command", msg.String())
			if m.cursor > 0 {
				m.cursor--
			}

		// Navigate down
		case "down", "j":
			log.Println("Received down command", msg.String())
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// vim jumps
		case "ctrl+d":
			log.Println("Received jump down command", msg.String())
			if m.cursor < len(m.choices)-8 {
				m.cursor += 8
			} else {
				m.cursor = len(m.choices) - 1
			}

		case "ctrl+u":
			log.Println("Received jump up command", msg.String())
			if m.cursor > 8 {
				m.cursor -= 8
			} else {
				m.cursor = 0
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			log.Println("Received select command", msg.String())
			// _, ok := m.selected[m.cursor]
			// if ok {
			// 	delete(m.selected, m.cursor)
			// } else {
			// 	m.selected[m.cursor] = struct{}{}
			// }
		}
        log.Println("Cursor: ", m.cursor)
        m.viewport.SetContent(m.getContent())
	}
    m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) getContent() string {
	s := ""
	// Iterate over our choices
	for i, choice := range m.choices {
		// Is the cursor pointing at this choice?
		cursor := "   " // no cursor
		// var tagsDisplay string
		if m.cursor == i {
			cursor = "-->" // cursor!
			// tagsDisplay = fmt.Sprintf("%v", choice.tags)
		} else {
			// tagsDisplay = fmt.Sprintf("")
		}
		// Render the row
		s += fmt.Sprintf("%s %s\n", cursor, choice.path)
	}

	// The footer
	return s
}

func (m model) View() string {
	// Render the viewport
	return appStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView()))
}

//////////////////////// SERVER ////////////////////////

// Struct holding the app's configuration
type Config struct {
	data              string
	samples           string
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
	return &Config{
		data:              data,
		samples:           samples,
		dbFileName:        dbFileName,
		createSqlCommands: sqlCommands,
	}
}

// Handles either creating or checking the existence of the data and samples directories
func (c *Config) handleDirectories() {
	if _, err := os.Stat(c.data); os.IsNotExist(err) {
		if err := os.MkdirAll(c.data, 0755); err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(c.samples); os.IsNotExist(err) {
		log.Fatal("No directory at config's samples directory ", c.samples)
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
	samples     string
	currentDir  string
	currentUser User
}

// Construct the server
func Sever() *Server {
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
		// log.Println("Database setup successfully.")
	}
	s := Server{
		db:         db,
		samples:    config.samples,
		currentDir: config.samples,
	}
	users := s.getUsers()
	if len(users) == 0 {
		println("No users found")
	}
	s.currentUser = users[0]
	return &s
}

type collectionTag struct {
	filePath       string
	collectionName string
	subCollection  string
}

type dirEntryWithTags struct {
	path string
	tags []collectionTag
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
			strings.HasSuffix(entry.Name(), ".aif") || strings.HasSuffix(entry.Name(), ".aiff") ||
			strings.HasSuffix(entry.Name(), ".flac") || strings.HasSuffix(entry.Name(), ".ogg") ||
			strings.HasSuffix(entry.Name(), ".m4a") {
			files = append(files, entry)
		}
	}
	return append(dirs, files...)
}

func (s *Server) listCurrentDir() []dirEntryWithTags {
	files, err := os.ReadDir(s.currentDir)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
    files = s.filterDirEntries(files)
	collectionTags := s.getCollectionTags(s.samples, &s.currentDir)
	var samples []dirEntryWithTags
	for _, file := range files {
		matchedTags := make([]collectionTag, 0)
		for _, tag := range collectionTags {
			if strings.Contains(tag.filePath, file.Name()) {
				matchedTags = append(matchedTags, tag)

			}
		}
		samples = append(samples, dirEntryWithTags{path: file.Name(), tags: matchedTags})
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

type App struct {
	server         *Server
	bubbleTeaModel model
	logFile        *os.File
}

func NewApp(server *Server, bubbleTeaModel model) App {
	f, err := os.OpenFile("logfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return App{
		server:         server,
		bubbleTeaModel: bubbleTeaModel,
		logFile:        f,
	}
}

func main() {
	server := Sever()
	app := NewApp(server, initialModel(server.listCurrentDir()))
	defer server.db.Close()
	defer app.logFile.Close()
	p := tea.NewProgram(
		app.bubbleTeaModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	if err != nil {
		log.Fatalf("Failed to run program: %v", err)
	}
}
