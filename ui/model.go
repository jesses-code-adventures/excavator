package ui

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/jesses-code-adventures/excavator/core"
	"github.com/jesses-code-adventures/excavator/keymaps"
	"github.com/jesses-code-adventures/excavator/server"
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
	Server                   *server.Server
	Viewport                 viewport.Model
	Help                     help.Model
	WindowType               core.WindowType
	Form                     core.Form
	SelectableList           string
	SearchableSelectableList core.SearchableSelectableList
}

// Constructor for the app's model
func ExcavatorModel(server *server.Server) Model {
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
