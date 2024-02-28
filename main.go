package main

import (
	"bufio"
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

	"github.com/jesses-code-adventures/excavator/audio"
	"github.com/jesses-code-adventures/excavator/core"
	"github.com/jesses-code-adventures/excavator/keymaps"

	// Database
	_ "github.com/mattn/go-sqlite3"

	// Frontend
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// ////////////////////// STYLING ////////////////////////

// All styles to be used throughout the ui
var (
	// colours
	Green = lipgloss.Color("#25A065")
	Pink  = lipgloss.Color("#E441B5")
	White = lipgloss.Color("#FFFDF5")
	// App
	AppStyle = lipgloss.NewStyle().
			Padding(1, 1)
	TitleStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Green).
			Padding(1, 1).
			Height(3)
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#909090",
			Dark:  "#626262",
		})
	HelpValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#B2B2B2",
			Dark:  "#4A4A4A",
		})
	// Directory Walker
	ViewportStyle = lipgloss.NewStyle()
	SelectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder()).
			Foreground(Pink)
	UnselectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder())
		// Form
	FormStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Border(lipgloss.HiddenBorder()).
			Margin(0, 0, 0)
	UnfocusedInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Width(100).
			Margin(1, 1).
			Border(lipgloss.HiddenBorder())
	FocusedInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Width(100).
			Margin(1, 1)
		// Searchable list
	SearchableListItemsStyle = lipgloss.NewStyle().
					Border(lipgloss.HiddenBorder())
	SearchInputBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()).
				AlignVertical(lipgloss.Bottom).
				AlignHorizontal(lipgloss.Left)
	SearchableSelectableListStyle = lipgloss.NewStyle().
					Border(lipgloss.HiddenBorder())
)

func (m Model) FilterListItems() Model {
	var resp []core.SelectableListItem
	switch m.SearchableSelectableList.Title {
	case "set target subcollection":
		r := m.Server.SearchCollectionSubcollections(m.SearchableSelectableList.Search.Input.Value())
		newArray := make([]core.SelectableListItem, 0)
		for _, item := range r {
			newArray = append(newArray, item)
		}
		resp = newArray
		m.Server.State.Choices = resp
	case "fuzzy search from root":
		log.Println("performing fuzzy search")
		m.Server.FuzzyFind(m.SearchableSelectableList.Search.Input.Value(), true)
	case "fuzzy search window":
		log.Println("performing fuzzy search")
		m.Server.FuzzyFind(m.SearchableSelectableList.Search.Input.Value(), false)
	}
	return m
}

// A generic Model defining app behaviour in all states
type Model struct {
	Ready                    bool
	Quitting                 bool
	ShowCollections          bool
	Cursor                   int
	PrevCursor               int
	ViewportHeight           int
	ViewportWidth            int
	Keys                     keymaps.KeyMap
	KeyHack                  keymaps.KeymapHacks
	Server                   *Server
	Viewport                 viewport.Model
	Help                     help.Model
	WindowType               core.WindowType
	Form                     core.Form
	SelectableList           string
	SearchableSelectableList core.SearchableSelectableList
}

// Constructor for the app's model
func ExcavatorModel(server *Server) Model {
	return Model{
		Ready:           false,
		Quitting:        false,
		ShowCollections: false,
		Server:          server,
		Help:            help.New(),
		Keys:            keymaps.DefaultKeyMap,
		WindowType:      core.DirectoryWalker,
	}
}

// Get the header of the viewport
func (m Model) HeaderView() string {
	title := TitleStyle.Render("Excavator - Samples")
	line := strings.Repeat("─", max(0, m.Viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

type StatusDisplayItem struct {
	key        string
	value      string
	keyStyle   lipgloss.Style
	valueStyle lipgloss.Style
}

func (s StatusDisplayItem) View() string {
	return s.keyStyle.Render(s.key+": ") + s.valueStyle.Render(s.value)
}

func NewStatusDisplayItem(key string, value string) StatusDisplayItem {
	return StatusDisplayItem{
		key:        key,
		value:      value,
		keyStyle:   HelpKeyStyle,
		valueStyle: HelpValueStyle,
	}
}

// Display key info to user
func (m Model) GetStatusDisplay() string {
	termWidth := m.Viewport.Width
	msg := ""
	// hack to make centering work
	msgRaw := fmt.Sprintf("collection: %v, subcollection: %v, window: %v, num items: %v, descriptions: %v", m.Server.User.TargetCollection.Name(), m.Server.User.TargetSubCollection, m.WindowType.String(), len(m.Server.State.Choices), m.ShowCollections)
	items := []StatusDisplayItem{
		NewStatusDisplayItem("collection", m.Server.User.TargetCollection.Name()),
		NewStatusDisplayItem("subcollection", m.Server.User.TargetSubCollection),
		NewStatusDisplayItem("window", fmt.Sprintf("%v", m.WindowType.String())),
		NewStatusDisplayItem("num items", fmt.Sprintf("%v", len(m.Server.State.Choices))),
		NewStatusDisplayItem("descriptions", fmt.Sprintf("%v", m.ShowCollections)),
	}
	for i, item := range items {
		msg += item.View()
		if i != len(items)-1 {
			msg = msg + HelpValueStyle.Render(", ")
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
func (m Model) FooterView() string {
	helpText := m.Help.View(m.Keys)
	termWidth := m.Viewport.Width
	helpTextLength := lipgloss.Width(helpText)
	padding := (termWidth - helpTextLength) / 2
	if padding < 0 {
		padding = 0
	}
	paddedHelpStyle := lipgloss.NewStyle().PaddingLeft(padding).PaddingRight(padding)
	centeredHelpText := paddedHelpStyle.Render(helpText)
	var searchInput string
	if m.WindowType == core.SearchableSelectableListWindow {
		searchInput = SearchInputBoxStyle.Render(m.SearchableSelectableList.Search.Input.View())
		return searchInput + "\n" + m.GetStatusDisplay() + "\n" + centeredHelpText
	}
	return m.GetStatusDisplay() + "\n" + centeredHelpText
}

// Formview handler
func (m Model) FormView() string {
	s := ""
	log.Println("got form ", m.Form)
	for i, input := range m.Form.Inputs {
		if m.Form.FocusedInput == i {
			s += FocusedInput.Render(fmt.Sprintf("%v: %v\n", input.Name, input.Input.View()))
		} else {
			s += UnfocusedInput.Render(fmt.Sprintf("%v: %v\n", input.Name, input.Input.View()))
		}
	}
	return s
}

// Standard content handler
func (m Model) DirectoryView() string {
	s := ""
	for i, choice := range m.Server.State.Choices {
		var newLine string
		if m.Cursor == i {
			cursor := "--> "
			newLine = fmt.Sprintf("%s %s", cursor, choice.Name())
		} else {
			newLine = fmt.Sprintf("     %s", choice.Name())
		}
		if len(newLine) > m.Viewport.Width {
			newLine = newLine[:m.Viewport.Width-2]
		}
		if m.Cursor == i {
			newLine = SelectedStyle.Render(newLine, fmt.Sprintf("    %v", choice.Description()))
		} else {
			if m.ShowCollections {
				newLine = UnselectedStyle.Render(newLine, fmt.Sprintf("    %v", choice.Description()))
			} else {
				newLine = UnselectedStyle.Render(newLine)
			}
		}
		s += newLine
	}
	return s
}

// // Ui updating for window resize events
func (m Model) HandleWindowResize(msg tea.WindowSizeMsg) Model {
	headerHeight := lipgloss.Height(m.HeaderView())
	footerHeight := lipgloss.Height(m.FooterView())
	searchInputHeight := 2 // Assuming the search input height is approximately 2 lines
	verticalPadding := 2   // Adjust based on your app's padding around the viewport
	// Calculate available height differently if in SearchableSelectableList mode
	if m.WindowType == core.SearchableSelectableListWindow {
		m.ViewportHeight = msg.Height - headerHeight - footerHeight - searchInputHeight - verticalPadding
	} else {
		m.ViewportHeight = msg.Height - headerHeight - footerHeight - verticalPadding
	}
	m.Viewport.Width = msg.Width
	m.Viewport.Height = m.ViewportHeight
	m.Ready = true
	return m
}

// Handle viewport positioning
func (m Model) EnsureCursorVerticallyCentered() viewport.Model {
	if m.WindowType != core.DirectoryWalker {
		m.Viewport.GotoTop()
		return m.Viewport
	}
	viewport := m.Viewport
	itemHeight := 2
	cursorPosition := m.Cursor * itemHeight
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
func (m Model) SetViewportContent(msg tea.Msg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch m.WindowType {
	case core.FormWindow:
		m.Viewport.SetContent(m.FormView())
	case core.DirectoryWalker:
		m.Viewport = m.EnsureCursorVerticallyCentered()
		m.Viewport.SetContent(m.DirectoryView())
	case core.ListSelectionWindow:
		m.Viewport.SetContent(m.DirectoryView())
	case core.SearchableSelectableListWindow:
		m.Viewport = m.EnsureCursorVerticallyCentered()
		m.Viewport.SetContent(m.DirectoryView())
	default:
		m.Viewport.SetContent("Invalid window type")
	}
	return m, cmd
}

// Handle all view rendering
func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	return AppStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.HeaderView(), ViewportStyle.Render(m.Viewport.View()), m.FooterView()))
}

// Necessary for bubbletea model interface
func (m Model) Init() tea.Cmd {
	return nil
}

// ////////////////////// UI UPDATING ////////////////////////

func (m Model) ClearModel() Model {
	m.Form = core.Form{}
	m.Server.State.Choices = make([]core.SelectableListItem, 0)
	m.SearchableSelectableList = core.SearchableSelectableList{}
	m.Cursor = 0
	return m
}

// Standard "home" view
func (m Model) GoToMainWindow(msg tea.Msg, cmd tea.Cmd) (Model, tea.Cmd) {
	m = m.ClearModel()
	m.WindowType = core.DirectoryWalker
	m.Server.UpdateChoices()
	return m, cmd
}

// Individual logic handlers for each list - setting window type handled outside this function
func (m Model) HandleTitledList(msg tea.Msg, cmd tea.Cmd, title string) (Model, tea.Cmd) {
	switch title {
	case "set target subcollection":
		subCollections := m.Server.GetCollectionSubcollections()
		for _, subCollection := range subCollections {
			m.Server.State.Choices = append(m.Server.State.Choices, subCollection)
		}
	case "select collection":
		collections := m.Server.GetCollections()
		m.SelectableList = title
		for _, collection := range collections {
			m.Server.State.Choices = append(m.Server.State.Choices, collection)
		}
	case "search for collection":
		collections := m.Server.GetCollections()
		m.SelectableList = title
		for _, collection := range collections {
			m.Server.State.Choices = append(m.Server.State.Choices, collection)
		}
	case "fuzzy search from root":
		files := m.Server.FuzzyFind("", true)
		for _, subCollection := range files {
			m.Server.State.Choices = append(m.Server.State.Choices, subCollection)
		}
	default:
		log.Fatalf("Invalid searchable selectable list title")
	}
	m.SearchableSelectableList = core.NewSearchableList(title)
	return m, cmd
}

func (m Model) HandleForm(msg tea.Msg, cmd tea.Cmd, title string) (Model, tea.Cmd) {
	switch title {
	case "new collection":
		m = m.ClearModel()
		m.Form = core.GetNewCollectionForm()
	case "create tag":
		m.Form = core.GetCreateTagForm(path.Base(m.Server.State.Choices[m.Cursor].Name()), m.Server.User.TargetSubCollection)
	}
	return m, cmd
}

// Main handler to be called any time the window changes
func (m Model) SetWindowType(msg tea.Msg, cmd tea.Cmd, windowType core.WindowType, title string) (Model, tea.Cmd) {
	if m.WindowType == windowType && m.SearchableSelectableList.Title == title {
		m, cmd = m.GoToMainWindow(msg, cmd)
		return m, cmd
	}
	log.Println("got window type ", windowType.String())
	switch windowType {
	case core.DirectoryWalker:
		m = m.ClearModel()
		m, cmd = m.GoToMainWindow(msg, cmd)
	case core.FormWindow:
		if title == "" {
			log.Fatalf("Title required for forms")
		}
		m, cmd = m.HandleForm(msg, cmd, title)
	case core.ListSelectionWindow:
		m = m.ClearModel()
		if title == "" {
			log.Fatalf("Title required for lists")
		}
		m, cmd = m.HandleTitledList(msg, cmd, title)
	case core.SearchableSelectableListWindow:
		m = m.ClearModel()
		if title == "" {
			log.Fatalf("Title required for lists")
		}
		m, cmd = m.HandleTitledList(msg, cmd, title)
	default:
		log.Fatalf("Invalid window type")
	}
	m.WindowType = windowType
	m.Cursor = 0
	return m, cmd
}

// Audition the file under the cursor
func (m Model) AuditionCurrentlySelectedFile() {
	if len(m.Server.State.Choices) == 0 {
		return
	}
	choice := m.Server.State.Choices[m.Cursor]
	if !choice.IsDir() && choice.IsFile() {
		var path string
		if !strings.Contains(choice.Name(), m.Server.State.Dir) {
			path = filepath.Join(m.Server.State.Dir, choice.Name())
		} else {
			path = choice.Name()
		}
		go m.Server.Player.PlayAudioFile(path)
	}
}

// These functions should run every time the cursor moves in directory view
func (m Model) DirVerticalNavEffect() {
	if m.Server.User.AutoAudition {
		m.AuditionCurrentlySelectedFile()
	}
}

// To be used across many window types for navigation
func (m Model) HandleStandardMovementKey(msg tea.KeyMsg) Model {
	switch {
	case key.Matches(msg, m.Keys.Up):
		if m.Cursor > 0 {
			m.Cursor--
		}
		m.DirVerticalNavEffect()
	case key.Matches(msg, m.Keys.Down):
		if m.Cursor < len(m.Server.State.Choices)-1 {
			m.Cursor++
		}
		m.DirVerticalNavEffect()
	case key.Matches(msg, m.Keys.JumpDown):
		if m.Cursor < len(m.Server.State.Choices)-8 {
			m.Cursor += 8
		} else {
			m.Cursor = len(m.Server.State.Choices) - 1
		}
		m.DirVerticalNavEffect()
	case key.Matches(msg, m.Keys.JumpUp):
		if m.Cursor > 8 {
			m.Cursor -= 8
		} else {
			m.Cursor = 0
		}
		m.DirVerticalNavEffect()
	case key.Matches(msg, m.Keys.Audition):
		m.AuditionCurrentlySelectedFile()
	case key.Matches(msg, m.Keys.AuditionRandom):
		fileIndex := m.Server.State.GetRandomAudioFileIndex()
		if fileIndex != -1 {
			m.Cursor = fileIndex
			m.DirVerticalNavEffect()
		}
		if !m.Server.User.AutoAudition {
			m.AuditionCurrentlySelectedFile()
		}
	case key.Matches(msg, m.Keys.JumpBottom):
		m.Viewport.GotoBottom()
		m.Cursor = len(m.Server.State.Choices) - 1
		m.DirVerticalNavEffect()
	}
	return m
}

// Handle a single key press
func (m Model) HandleDirectoryKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		m.Quitting = true
		return m, cmd
	case key.Matches(msg, m.Keys.Up) || key.Matches(msg, m.Keys.Down) || key.Matches(msg, m.Keys.JumpDown) || key.Matches(msg, m.Keys.JumpUp) || key.Matches(msg, m.Keys.Audition) || key.Matches(msg, m.Keys.AuditionRandom) || key.Matches(msg, m.Keys.JumpBottom):
		m = m.HandleStandardMovementKey(msg)
	case key.Matches(msg, m.Keys.Enter):
		choice := m.Server.State.Choices[m.Cursor]
		if choice.IsDir() {
			if choice.Name() == ".." {
				m.Cursor = 0
				m.Server.State.ChangeToParentDir()
			} else {
				m.Cursor = 0
				m.Server.State.ChangeDir(choice.Name())
			}
		}
	case key.Matches(msg, m.Keys.ToggleShowCollections):
		m.ShowCollections = !m.ShowCollections
	case key.Matches(msg, m.Keys.NewCollection):
		m, cmd = m.SetWindowType(msg, cmd, core.FormWindow, "new collection")
	case key.Matches(msg, m.Keys.SetTargetSubCollection):
		log.Println("going to set target subcollection")
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "set target subcollection")
	case key.Matches(msg, m.Keys.FuzzySearchFromRoot):
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "fuzzy search from root")
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindowType(msg, cmd, core.ListSelectionWindow, "search for collection")
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.CreateQuickTag):
		choice := m.Server.State.Choices[m.Cursor]
		if !choice.IsDir() {
			m.Server.CreateQuickTag(choice.Name())
		}
		m.Server.UpdateChoices()
	case key.Matches(msg, m.Keys.CreateTag):
		m, cmd = m.SetWindowType(msg, cmd, core.FormWindow, "create tag")
	default:
		switch msg.String() {
		case "g":
			if m.KeyHack.GetLastKey() == "g" {
				m.Cursor = 0
			}
		}
	}
	return m, cmd
}

// List selection navigation
func (m Model) HandleListSelectionKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		return m.GoToMainWindow(msg, cmd)
	case key.Matches(msg, m.Keys.Up) || key.Matches(msg, m.Keys.Down) || key.Matches(msg, m.Keys.JumpDown) || key.Matches(msg, m.Keys.JumpUp) || key.Matches(msg, m.Keys.Audition) || key.Matches(msg, m.Keys.AuditionRandom) || key.Matches(msg, m.Keys.JumpBottom):
		m = m.HandleStandardMovementKey(msg)
	case key.Matches(msg, m.Keys.ToggleShowCollections):
		m.ShowCollections = !m.ShowCollections
	case key.Matches(msg, m.Keys.NewCollection):
		m.Form = core.GetNewCollectionForm()
		m, cmd = m.SetWindowType(msg, cmd, core.FormWindow, "")
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "search for collection")
	case key.Matches(msg, m.Keys.SetTargetSubCollection):
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "set target subcollection")
	case key.Matches(msg, m.Keys.FuzzySearchFromRoot):
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "fuzzy search from root")
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.Enter):
		switch m.SelectableList {
		case "search for collection":
			if collection, ok := m.Server.State.Choices[m.Cursor].(core.Collection); ok {
				m.Server.UpdateTargetCollection(collection)
				m, cmd = m.GoToMainWindow(msg, cmd)
			} else {
				log.Fatalf("Invalid list selection item type")
			}
		}
	}
	return m, cmd
}

// Form key
func (m Model) HandleFormKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	if m.Form.Writing {
		m, cmd = m.HandleFormWritingKey(msg, cmd)
	} else {
		m, cmd = m.HandleFormNavigationKey(msg, cmd)
	}
	return m, cmd
}

// Form navigation
func (m Model) HandleFormNavigationKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Up):
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		m.Form.FocusedInput++
		if m.Form.FocusedInput >= len(m.Form.Inputs) {
			m.Form.FocusedInput = 0
		}
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	case key.Matches(msg, m.Keys.Down):
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		m.Form.FocusedInput--
		if m.Form.FocusedInput < 0 {
			m.Form.FocusedInput = len(m.Form.Inputs) - 1
		}
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	case key.Matches(msg, m.Keys.InsertMode):
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		m.Form.Writing = true
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	case key.Matches(msg, m.Keys.Quit), key.Matches(msg, m.Keys.NewCollection), key.Matches(msg, m.Keys.SetTargetSubCollection), key.Matches(msg, m.Keys.FuzzySearchFromRoot):
		m, cmd = m.GoToMainWindow(msg, cmd)
	case key.Matches(msg, m.Keys.SetTargetCollection):
		// m, cmd = m.handleListSelectionKey(msg, cmd)
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "search for collection")
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.Enter):
		for i, input := range m.Form.Inputs {
			if input.Input.Value() == "" {
				m.Form.FocusedInput = i
				m.Form.Inputs[i].Input.Focus()
				return m, cmd
			}
		}
		switch m.Form.Title {
		case "create collection":
			m.Server.CreateCollection(m.Form.Inputs[0].Input.Value(), m.Form.Inputs[1].Input.Value())
		case "create tag":
			m.Server.CreateTag(m.Server.State.Choices[m.Cursor].Name(), m.Form.Inputs[0].Input.Value(), m.Form.Inputs[1].Input.Value())
		}
		// case "set target subcollection":
		// 	m.server.updateTargetSubCollection(m.form.inputs[0].input.Value())
		m, cmd = m.GoToMainWindow(msg, cmd)
	}
	return m, cmd
}

// Form writing
func (m Model) HandleFormWritingKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		m.Form.Writing = false
		if m.WindowType == core.SearchableSelectableListWindow {
			m.SearchableSelectableList.Search.Input.Blur()
			m = m.FilterListItems()
			log.Printf("filtered items in form writing key quit %v", m.Server.State.Choices)
			m.Cursor = 0
		} else {
			m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		}
	case key.Matches(msg, m.Keys.Enter):
		m.Form.Writing = false
		if m.WindowType == core.SearchableSelectableListWindow {
			m.SearchableSelectableList.Search.Input.Blur()
			m = m.FilterListItems()
			log.Printf("filtered items in form writing key enter %v", m.Server.State.Choices)
			m.Cursor = 0
		} else {
			m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		}
	default:
		var newInput textinput.Model
		if m.WindowType == core.SearchableSelectableListWindow {
			newInput, cmd = m.SearchableSelectableList.Search.Input.Update(msg)
			m.SearchableSelectableList.Search.Input = newInput
			m.SearchableSelectableList.Search.Input.Focus()
			if m.SelectableList == "set target subcollection" {
				m = m.FilterListItems()
				m.Cursor = 0
			}
		} else {
			newInput, cmd = m.Form.Inputs[m.Form.FocusedInput].Input.Update(msg)
			m.Form.Inputs[m.Form.FocusedInput].Input = newInput
			m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
		}
	}
	return m, cmd
}

// List selection navigation
func (m Model) HandleSearchableListNavKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		return m.GoToMainWindow(msg, cmd)
	case key.Matches(msg, m.Keys.Up) || key.Matches(msg, m.Keys.Down) || key.Matches(msg, m.Keys.JumpDown) || key.Matches(msg, m.Keys.JumpUp) || key.Matches(msg, m.Keys.Audition) || key.Matches(msg, m.Keys.AuditionRandom) || key.Matches(msg, m.Keys.JumpBottom):
		m = m.HandleStandardMovementKey(msg)
	case key.Matches(msg, m.Keys.ToggleShowCollections):
		m.ShowCollections = !m.ShowCollections
	case key.Matches(msg, m.Keys.NewCollection):
		m.Form = core.GetNewCollectionForm()
		m, cmd = m.SetWindowType(msg, cmd, core.FormWindow, "")
	case key.Matches(msg, m.Keys.SetTargetSubCollection):
		m.Form = core.GetTargetSubCollectionForm()
		m, cmd = m.SetWindowType(msg, cmd, core.FormWindow, "")
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindowType(msg, cmd, core.SearchableSelectableListWindow, "search for collection")
	case key.Matches(msg, m.Keys.InsertMode) || key.Matches(msg, m.Keys.SearchBuf):
		m.SearchableSelectableList.Search.Input.Focus()
		m.Form.Writing = true
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.Enter):
		value := m.SearchableSelectableList.Search.Input.Value()
		switch m.SearchableSelectableList.Title {
		case "fuzzy search from root":
			if value == "" {
				return m, cmd
			}
			m.Cursor = 0
			m.Server.FuzzyFind(value, true)
			return m, cmd
		case "fuzzy search window":
			if value == "" {
				return m, cmd
			}
			m.Cursor = 0
			m.Server.FuzzyFind(value, false)
			return m, cmd
		case "set target subcollection":
			if len(m.Server.State.Choices) == 0 && len(value) > 0 {
				m.Server.UpdateTargetSubCollection(value)
			} else {
				selected := m.Server.State.Choices[m.Cursor]
				log.Printf("selected: %v", selected)
				if collection, ok := selected.(core.SelectableListItem); ok {
					log.Printf("selected collection: %v", collection.Name())
					m.Server.UpdateTargetSubCollection(collection.Name())
				} else {
					log.Fatalf("Invalid list selection item type")
				}
			}
			m, cmd = m.GoToMainWindow(msg, cmd)
		}
	}
	return m, cmd
}

// Form key
func (m Model) HandleSearchableListKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	if m.Form.Writing {
		m, cmd = m.HandleFormWritingKey(msg, cmd)
	} else {
		m, cmd = m.HandleSearchableListNavKey(msg, cmd)
	}
	return m, cmd
}

// Takes a message and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.HandleWindowResize(msg)
	case tea.KeyMsg:
		switch m.WindowType {
		case core.FormWindow:
			m, cmd = m.HandleFormKey(msg, cmd)
		case core.ListSelectionWindow:
			m, cmd = m.HandleListSelectionKey(msg, cmd)
		case core.DirectoryWalker:
			m, cmd = m.HandleDirectoryKey(msg, cmd)
			if m.Quitting {
				return m, tea.Quit
			}
		case core.SearchableSelectableListWindow:
			m, cmd = m.HandleSearchableListKey(msg, cmd)
		}
		m.KeyHack.UpdateLastKey(msg.String())
	}
	return m.SetViewportContent(msg, cmd)
}

//////////////////////// LOCAL SERVER ////////////////////////

type State struct {
	Root           string
	Dir            string
	choiceChannel  chan core.SelectableListItem
	Choices        []core.SelectableListItem
	CollectionTags func(path string) []core.CollectionTag
}

func NewNavState(root string, currentDir string, collectionTags func(path string) []core.CollectionTag) *State {
	choiceChannel := make(chan core.SelectableListItem)
	navState := State{
		Root:           root,
		Dir:            currentDir,
		choiceChannel:  choiceChannel,
		Choices:        make([]core.SelectableListItem, 0),
		CollectionTags: collectionTags,
	}
	go navState.Run()
	return &navState
}

func (n *State) Run() {
	for {
		select {
		case choice := <-n.choiceChannel:
			n.Choices = append(n.Choices, choice)
		}
	}
}

func (n *State) pushChoice(choice core.SelectableListItem) {
	n.choiceChannel <- choice
}

// Grab an index of some audio file within the current directory
func (n *State) GetRandomAudioFileIndex() int {
	if len(n.Choices) == 0 {
		return -1
	}
	possibleIndexes := make([]int, 0)
	for i, choice := range n.Choices {
		if !choice.IsDir() {
			possibleIndexes = append(possibleIndexes, i)
		}
	}
	return possibleIndexes[rand.Intn(len(possibleIndexes))]
}

// Populate the choices array with the current directory's contents
func (n *State) UpdateChoices() {
	if n.Dir != n.Root {
		n.Choices = make([]core.SelectableListItem, 0)
		dirEntries := n.ListDirEntries()
		n.Choices = append(n.Choices, core.TaggedDirentry{Path: "..", Tags: make([]core.CollectionTag, 0), Dir: true})
		n.Choices = append(n.Choices, dirEntries...)
	} else {
		n.Choices = n.ListDirEntries()
	}
}

// Return only directories and valid audio files
func (f *State) FilterDirEntries(entries []os.DirEntry) []os.DirEntry {
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
func (f *State) ListDirEntries() []core.SelectableListItem {
	files, err := os.ReadDir(f.Dir)
	log.Printf("current dir: %v", f.Dir)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	files = f.FilterDirEntries(files)
	var samples []core.SelectableListItem
	for _, file := range files {
		matchedTags := make([]core.CollectionTag, 0)
		isDir := file.IsDir()
		if !isDir {
			for _, tag := range f.CollectionTags(f.Dir) {
				if strings.Contains(tag.FilePath, file.Name()) {
					matchedTags = append(matchedTags, tag)
				}
			}
		}
		samples = append(samples, core.TaggedDirentry{Path: file.Name(), Tags: matchedTags, Dir: isDir})
	}
	return samples
}

// Get the full path of the current directory
func (n *State) GetCurrentDirPath() string {
	return filepath.Join(n.Root, n.Dir)
}

// Change the current directory
func (n *State) ChangeDir(dir string) {
	log.Println("Changing to dir: ", dir)
	n.Dir = filepath.Join(n.Dir, dir)
	log.Println("Current dir: ", n.Dir)
	n.UpdateChoices()
}

// Change the current directory to the root
func (n *State) ChangeToRoot() {
	n.Dir = n.Root
	n.UpdateChoices()
}

// Change the current directory to the parent directory
func (n *State) ChangeToParentDir() {
	log.Println("Changing to dir: ", filepath.Dir(n.Dir))
	n.Dir = filepath.Dir(n.Dir)
	n.UpdateChoices()
}

func (s *Server) GetAllDirectories(path string) []string {
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

// The main struct holding the Server
type Server struct {
	Db     *sql.DB
	User   core.User
	State  *State
	Player *audio.Player
}

func (s *Server) HandleUserArg(userCliArg *string) core.User {
	var user core.User
	users := s.GetUsers(userCliArg)
	if len(*userCliArg) == 0 && len(users) == 0 {
		log.Fatal("No users found")
	}
	if len(*userCliArg) == 0 && len(users) > 0 {
		user = users[0]
		return user
	}
	if len(*userCliArg) > 0 && len(users) == 0 {
		id := s.CreateUser(*userCliArg)
		if id == 0 {
			log.Fatal("Failed to create user")
		}
		user = s.GetUser(id)
		return user
	}
	if len(*userCliArg) > 0 && len(users) > 0 {
		for _, u := range users {
			if u.Name == *userCliArg {
				return u
			}
		}
		id := s.CreateUser(*userCliArg)
		user = s.GetUser(id)
		return user
	}
	log.Fatal("We should never get here")
	return user
}

type Flags struct {
	data       string
	dbFileName string
	logFile    string
	root       string
	user       string
	watch      bool
}

func ParseFlags() *Flags {
	var data = flag.String("data", "~/.excavator-tui", "Local data storage path")
	var dbFileName = flag.String("db", "excavator", "Database file name")
	var logFile = flag.String("log", "logfile", "Log file name")
	var samples = flag.String("root", "~/Library/Audio/Sounds/Samples", "Root samples directory")
	var userArg = flag.String("user", "", "User name to launch with")
	var watch = flag.Bool("watch", false, "Watch for changes in the samples directory")
	flag.Parse()
	return &Flags{data: core.ExpandHomeDir(*data), dbFileName: *dbFileName, logFile: *logFile, root: core.ExpandHomeDir(*samples), user: *userArg, watch: *watch}
}

// Part of newServer constructor
func (s *Server) HandleRootConstruction(config *core.Config) *Server {
	if s.User.Root == "" && config.Root == "" {
		log.Fatal("No root found")
	} else if config.Root == "" {
		config.Root = s.User.Root
	} else if s.User.Root == "" {
		s.User.Root = config.Root // TODO: prompt the user to see if they want to save the root
		s.UpdateRootInDb(config.Root)
	} else if s.User.Root != config.Root {
		log.Println("launched with temporary root ", config.Root)
		s.User.Root = config.Root
	}
	log.Printf("Current user: %v, selected collection: %v, target subcollection: %v", s.User, s.User.TargetCollection.Name(), s.User.TargetSubCollection)
	return s
}

// Construct the server
func NewServer(audioPlayer *audio.Player, flags *Flags) *Server {
	config := core.NewConfig(flags.data, flags.root, flags.dbFileName)
	config.CreateDataDirectory()
	dbPath := config.GetDbPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to create sqlite file %v", err)
	}
	if _, err := os.Stat(config.GetDbPath()); os.IsNotExist(err) {
		_, innerErr := db.Exec(string(config.CreateSqlCommands))
		if innerErr != nil {
			log.Fatalf("Failed to execute SQL commands: %v", innerErr)
		}
	}
	s := Server{
		Db:     db,
		Player: audioPlayer,
	}
	s.User = s.HandleUserArg(&flags.user)
	s = *s.HandleRootConstruction(config)
	navState := NewNavState(config.Root, config.Root, s.GetCollectionTags)
	s.State = navState
	s.State.UpdateChoices()
	return &s
}

func (s *Server) SetRoot(path string) {
	s.State.Root = path
	s.State.Dir = path
	s.State.UpdateChoices()
	s.User.Root = path
	s.UpdateRootInDb(path)
}

// Set the current user's auto audition preference and update in db
func (s *Server) UpdateAutoAudition(autoAudition bool) {
	s.User.AutoAudition = autoAudition
	s.UpdateAutoAuditionInDb(autoAudition)
}

func (s *Server) UpdateChoices() {
	s.State.UpdateChoices()
}

// Set the current user's target collection and update in db
func (s *Server) UpdateTargetCollection(collection core.Collection) {
	s.User.TargetCollection = &collection
	s.UpdateSelectedCollectionInDb(collection.Id())
	s.UpdateTargetSubCollection("")
	s.User.TargetSubCollection = ""
}

// Set the current user's target subcollection and update in db
func (s *Server) UpdateTargetSubCollection(subCollection string) {
	if len(subCollection) > 0 && !strings.HasPrefix(subCollection, "/") {
		subCollection = "/" + subCollection
	}
	s.User.TargetSubCollection = subCollection
	s.UpdateTargetSubCollectionInDb(subCollection)
}

// Create a tag with the defaults based on the current state
func (s *Server) CreateQuickTag(filepath string) {
	s.CreateCollectionTagInDb(filepath, s.User.TargetCollection.Id(), path.Base(filepath), s.User.TargetSubCollection)
	s.UpdateChoices()
}

// Create a tag with all possible args
func (s *Server) CreateTag(filepath string, name string, subCollection string) {
	s.CreateCollectionTagInDb(filepath, s.User.TargetCollection.Id(), name, subCollection)
	s.UpdateChoices()
}

func ContainsAllSubstrings(s1 string, s2 string) bool {
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
func (s *Server) FuzzyFind(search string, fromRoot bool) []core.SelectableListItem {
	log.Println("in server fuzzy search fn")
	var dir string
	var entries []os.DirEntry = make([]os.DirEntry, 0)
	var files []fs.DirEntry
	var samples []core.SelectableListItem
	if len(search) == 0 {
		return make([]core.SelectableListItem, 0)
	}
	if fromRoot {
		dir = s.State.Root
	} else {
		dir = s.State.Dir
	}
	collectionTags := s.FuzzyFindCollectionTags(search)
	log.Println("collection tags", collectionTags)
	log.Println("searching for: ", search)
	log.Println("dir: ", dir)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !ContainsAllSubstrings(path, search) || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".asd") {
			return nil
		}
		if (strings.HasSuffix(path, ".wav") || strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".flac")) && !d.IsDir() {
			entries = append(entries, d)
		}
		files = append(files, d)
		matchedTags := make([]core.CollectionTag, 0)
		for _, tag := range collectionTags {
			if strings.Contains(tag.FilePath, path) {
				matchedTags = append(matchedTags, tag)
			}
		}
		s.State.pushChoice(core.TaggedDirentry{Path: path, Tags: matchedTags, Dir: false})
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
func (s *Server) GetCollectionTags(dir string) []core.CollectionTag {
	statement := `select t.file_path, col.name, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where t.file_path like ?`
	dir = dir + "%"
	rows, err := s.Db.Query(statement, dir)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]core.CollectionTag, 0)
	for rows.Next() {
		var filePath, collectionName, subCollection string
		if err := rows.Scan(&filePath, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		tags = append(tags, core.CollectionTag{FilePath: filePath, CollectionName: collectionName, SubCollection: subCollection})
	}
	return tags
}

func (s *Server) FuzzyFindCollectionTags(search string) []core.CollectionTag {
	words := strings.Fields(search)
	if len(words) == 0 {
		return make([]core.CollectionTag, 0)
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
	rows, err := s.Db.Query(statement, search)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]core.CollectionTag, 0)
	log.Println("collection tags")
	for rows.Next() {
		var filePath, collectionName, subCollection string
		if err := rows.Scan(&filePath, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		log.Printf("filepath: %s, collection name: %s, subcollection: %s", filePath, collectionName, subCollection)
		tags = append(tags, core.CollectionTag{FilePath: filePath, CollectionName: collectionName, SubCollection: subCollection})
	}
	return tags
}

// Get collection tags associated with a directory
func (s *Server) SearchCollectionTags(search string) []core.CollectionTag {
	statement := `select t.file_path, col.name, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where t.file_path like ?`
	search = "%" + search + "%"
	rows, err := s.Db.Query(statement, search)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]core.CollectionTag, 0)
	log.Println("collection tags")
	for rows.Next() {
		var filePath, collectionName, subCollection string
		if err := rows.Scan(&filePath, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		log.Printf("filepath: %s, collection name: %s, subcollection: %s", filePath, collectionName, subCollection)
		tags = append(tags, core.CollectionTag{FilePath: filePath, CollectionName: collectionName, SubCollection: subCollection})
	}
	return tags
}

func (s *Server) GetUser(id int) core.User {
	fmt.Println("getting user ", id)
	statement := `select u.name as user_name, c.id as collection_id, c.name as collection_name, c.description, u.auto_audition, u.selected_subcollection, u.root from User u left join Collection c on u.selected_collection = c.id where u.id = ?`
	row := s.Db.QueryRow(statement, id)
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
	var selectedCollection *core.Collection
	if collectionId != nil && collectionName != nil && collectionDescription != nil {
		collection := core.NewCollection(*collectionId, *collectionName, *collectionDescription)
		selectedCollection = &collection
	} else {
		collection := core.NewCollection(0, "", "")
		selectedCollection = &collection
	}
	return core.User{Id: id, Name: name, AutoAudition: autoAudition, TargetCollection: selectedCollection, TargetSubCollection: selectedSubCollection, Root: root}
}

// Get all users
func (s *Server) GetUsers(name *string) []core.User {
	var whereClause string
	var rows *sql.Rows
	var err error
	if name != nil && len(*name) > 0 {
		whereClause = "where u.name = ?"
	}
	statement := `select u.id as user_id, u.name as user_name, c.id as collection_id, c.name as collection_name, c.description, u.auto_audition, u.selected_subcollection, u.root from User u left join Collection c on u.selected_collection = c.id`
	if whereClause != "" {
		statement = statement + " " + whereClause
		rows, err = s.Db.Query(statement, name)
	} else {
		rows, err = s.Db.Query(statement)
	}
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getUsers: %v", err)
	}
	defer rows.Close()
	users := make([]core.User, 0)
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
		var selectedCollection *core.Collection
		if collectionId != nil && collectionName != nil && collectionDescription != nil {
			collection := core.NewCollection(*collectionId, *collectionName, *collectionDescription)
			selectedCollection = &collection
		} else {
			collection := core.NewCollection(0, "", "")
			selectedCollection = &collection
		}
		users = append(users, core.User{Id: id, Name: name, AutoAudition: autoAudition, TargetCollection: selectedCollection, TargetSubCollection: selectedSubCollection, Root: root})
	}
	return users
}

// Create a user in the database
func (s *Server) CreateUser(name string) int {
	res, err := s.Db.Exec("insert or ignore into User (name) values (?)", name)
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
func (s *Server) UpdateSelectedCollectionInDb(collection int) {
	_, err := s.Db.Exec("update User set selected_collection = ? where id = ?", collection, s.User.Id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateSelectedCollectionInDb: %v", err)
	}
}

// Update the current user's auto audition preference in the database
func (s *Server) UpdateRootInDb(path string) {
	_, err := s.Db.Exec("update User set root = ? where id = ?", path, s.User.Id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in update root in db: %v", err)
	}
}

// Update the current user's auto audition preference in the database
func (s *Server) UpdateAutoAuditionInDb(autoAudition bool) {
	_, err := s.Db.Exec("update User set auto_audition = ? where id = ?", autoAudition, s.User.Id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateAutoAuditionInDb: %v", err)
	}
}

// Update the current user's name in the database
func (s *Server) UpdateUsername(id int, name string) {
	_, err := s.Db.Exec("update User set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateUsername: %v", err)
	}
}

// Create a collection in the database
func (s *Server) CreateCollection(name string, description string) int {
	var err error
	var res sql.Result
	res, err = s.Db.Exec("insert into Collection (name, user_id, description) values (?, ?, ?)", name, s.User.Id, description)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in createCollection: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

// Get all collections for the current user
func (s *Server) GetCollections() []core.Collection {
	statement := `select id, name, description from Collection where user_id = ?`
	rows, err := s.Db.Query(statement, s.User.Id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getCollections: %v", err)
	}
	defer rows.Close()
	collections := make([]core.Collection, 0)
	for rows.Next() {
		var id int
		var name string
		var description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		collection := core.NewCollection(id, name, description)
		collections = append(collections, collection)
	}
	return collections
}

// Update a collection's name in the database
func (s *Server) UpdateCollectionNameInDb(id int, name string) {
	_, err := s.Db.Exec("update Collection set name = ? where id = ?", name, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateCollectionNameInDb: %v", err)
	}
}

// Requirement for a listSelectionItem
func (s *Server) UpdateCollectionDescriptionInDb(id int, description string) {
	_, err := s.Db.Exec("update Collection set description = ? where id = ?", description, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in updateCollectionDescriptionInDb: %v", err)
	}
}

// Create a tag in the database
func (s *Server) CreateTagInDb(filePath string) int {
	if !strings.Contains(filePath, s.State.Root) {
		filePath = filepath.Join(s.State.Dir, filePath)
	}
	res, err := s.Db.Exec("insert or ignore into Tag (file_path, user_id) values (?, ?)", filePath, s.User.Id)
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
func (s *Server) AddTagToCollectionInDb(tagId int, collectionId int, name string, subCollection string) {
	log.Printf("Tag id: %d, collectionId: %d, name: %s, subCollection: %s", tagId, collectionId, name, subCollection)
	res, err := s.Db.Exec("insert or ignore into CollectionTag (tag_id, collection_id, name, sub_collection) values (?, ?, ?, ?)", tagId, collectionId, name, subCollection)
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
func (s *Server) CreateCollectionTagInDb(filePath string, collectionId int, name string, subCollection string) {
	tagId := s.CreateTagInDb(filePath)
	log.Printf("Tag id: %d", tagId)
	s.AddTagToCollectionInDb(tagId, collectionId, name, subCollection)
}

func (s *Server) UpdateTargetSubCollectionInDb(subCollection string) {
	_, err := s.Db.Exec("update User set selected_subcollection = ? where id = ?", subCollection, s.User.Id)
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

func (s *Server) GetCollectionSubcollections() []SubCollection {
	statement := `select distinct sub_collection from CollectionTag where collection_id = ? order by sub_collection asc`
	rows, err := s.Db.Query(statement, s.User.TargetCollection.Id())
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

func (s *Server) SearchCollectionSubcollections(search string) []SubCollection {
	fuzzySearch := "%" + search + "%"
	statement := `SELECT DISTINCT sub_collection
                  FROM CollectionTag
                  WHERE collection_id = ? AND sub_collection LIKE ?
                  ORDER BY sub_collection ASC`
	rows, err := s.Db.Query(statement, s.User.TargetCollection.Id(), fuzzySearch)
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

// ////////////////////// APP ////////////////////////
type App struct {
	server         *Server
	bubbleTeaModel Model
	logFile        *os.File
}

// Construct the app
func NewApp(cliFlags *Flags) App {
	logFilePath := filepath.Join(cliFlags.data, cliFlags.logFile)
	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	audioPlayer := audio.NewAudioPlayer()
	server := NewServer(audioPlayer, cliFlags)
	return App{
		server:         server,
		bubbleTeaModel: ExcavatorModel(server),
		logFile:        f,
	}
}

// watches the logfile
func Watch(filePath string, n int) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			lines := make([]string, 0)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
				if len(lines) > n {
					lines = lines[1:]
				}
			}

			fmt.Print("\033[H\033[2J")
			for _, line := range lines {
				fmt.Println(line)
			}

			// Handle error from scanner.Err()
			if err := scanner.Err(); err != nil {
				return err
			}
		}
	}
}

// chris_brown_run_it.ogg
func main() {
	cliFlags := ParseFlags()
	core.CreateDirectories(cliFlags.data)
	logFilePath := filepath.Join(cliFlags.data, cliFlags.logFile)
	if cliFlags.watch {
		Watch(logFilePath, 10)
	} else {
		app := NewApp(cliFlags)
		defer app.logFile.Close()
		defer app.server.Player.Close()
		defer app.server.Db.Close()
		p := tea.NewProgram(
			app.bubbleTeaModel,
			tea.WithAltScreen(),
		)
		_, err := p.Run()
		if err != nil {
			log.Fatalf("Failed to run program: %v", err)
		}
	}
}
