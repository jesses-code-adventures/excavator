package window

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

// A generic Model defining app behaviour in all states
type Model struct {
	Ready                    bool
	Quitting                 bool
	ShowCollections          bool
	Cursor                   int
	Keys                     keymaps.KeyMap
	KeyHack                  keymaps.KeymapHacks
	Server                   *server.Server
	Help                     help.Model
	Window                   Window
	Viewport                 viewport.Model
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
		Window:          Home.Window(),
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
	msgRaw := fmt.Sprintf("collection: %v, subcollection: %v, window type: %v, window name: %v, num items: %v, descriptions: %v", m.Server.User.TargetCollection.Name(), m.Server.User.TargetSubCollection, m.Window.Type().String(), m.Window.Name(), len(m.Server.State.Choices), m.ShowCollections)
	items := []StatusDisplayItem{
		NewStatusDisplayItem("collection", m.Server.User.TargetCollection.Name()),
		NewStatusDisplayItem("subcollection", m.Server.User.TargetSubCollection),
		NewStatusDisplayItem("window type", fmt.Sprintf("%v", m.Window.Type().String())),
		NewStatusDisplayItem("window name", fmt.Sprintf("%v", m.Window.Name())),
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
	return m.GetStatusDisplay() + "\n" + centeredHelpText
}

// // Ui updating for window resize events
func (m Model) HandleWindowResize(msg tea.WindowSizeMsg) Model {
	headerHeight := lipgloss.Height(m.HeaderView())
	footerHeight := lipgloss.Height(m.FooterView())
	// searchInputHeight := 2 // Assuming the search input height is approximately 2 lines
	// verticalPadding := 2   // Adjust based on your app's padding around the viewport
	// Calculate available height differently if in SearchableSelectableList mode
	// if m.Window.Type() == SearchableSelectableListWindow {
	// 	m.Viewport.Height = msg.Height - headerHeight - footerHeight - searchInputHeight - verticalPadding
	// } else {
	// }
	m.Viewport.Height = msg.Height - headerHeight - footerHeight
	m.Viewport.Width = msg.Width
	m.Ready = true
	return m
}

// Handle viewport positioning
func (m Model) EnsureCursorVerticallyCentered() viewport.Model {
	if m.Window.Type() == FormWindow {
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

// Set the content of the viewport based on the window type
func (m Model) SetViewportContent(msg tea.Msg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch m.Window.Type() {
	case FormWindow:
		log.Println("setting viewport content for ", m.Form.Title)
		m.Viewport.SetContent(FormWindow.FormView(m.Form))
	default:
		m.Viewport.SetContent(m.Window.Type().DirectoryView(
			m.Server.State.Choices,
			m.Cursor,
			m.Viewport.Width,
			m.ShowCollections,
			m.SearchableSelectableList.Search.Input,
		))
		m.Viewport = m.EnsureCursorVerticallyCentered()
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
func (m Model) GoToHome(msg tea.Msg, cmd tea.Cmd) (Model, tea.Cmd) {
	m = m.ClearModel()
	m.Window = Home.Window()
	m.Server.UpdateChoices()
	return m, cmd
}

// Individual logic handlers for each list - setting window type handled outside this function
func (m Model) HandleTitledList(msg tea.Msg, cmd tea.Cmd, window WindowName) (Model, tea.Cmd) {
	switch window {
	case SetTargetSubCollectionWindow:
		subCollections := m.Server.GetCollectionSubcollections()
		for _, subCollection := range subCollections {
			m.Server.State.Choices = append(m.Server.State.Choices, subCollection)
		}
	case SetTargetCollectionWindow:
		collections := m.Server.GetCollections()
		m.SelectableList = window.String()
		for _, collection := range collections {
			m.Server.State.Choices = append(m.Server.State.Choices, collection)
		}
	case FuzzySearchRootWindow, FuzzySearchCurrentWindow:
		m.Server.State.Choices = make([]core.SelectableListItem, 0)
	case Home:
	case BrowseCollectionWindow:
		tags := m.Server.GetCollectionTagsAsListItem(m.Server.User.TargetCollection.Id())
		for _, tag := range tags {
			m.Server.State.Choices = append(m.Server.State.Choices, tag)
		}
	case RunExportWindow:
		exports := m.Server.GetExports()
		for _, export := range exports {
			m.Server.State.Choices = append(m.Server.State.Choices, export)
		}
		m.SelectableList = window.String()
	default:
		log.Fatalf("Invalid searchable selectable list title")
	}
	m.SearchableSelectableList = core.NewSearchableList(window.String())
	return m, cmd
}

func (m Model) HandleForm(msg tea.Msg, cmd tea.Cmd, window WindowName) (Model, tea.Cmd) {
	switch window {
	case NewCollectionWindow:
		m = m.ClearModel()
		m.Form = core.GetNewCollectionForm()
	case CreateExportWindow:
		m = m.ClearModel()
		form := core.NewForm(window.String(), []core.FormInput{
			core.NewFormInput("name"),
			core.NewFormInput("output_dir"),
			core.NewFormInput("concrete"),
		})
		m.Form = form
	case NewTagWindow:
		m.Form = core.GetCreateTagForm(path.Base(m.Server.State.Choices[m.Cursor].Name()), m.Server.User.TargetSubCollection)
	}
	return m, cmd
}

// Main handler to be called any time the window changes
func (m Model) SetWindow(msg tea.Msg, cmd tea.Cmd, window WindowName) (Model, tea.Cmd) {
	if m.Window.Name() == window {
		m, cmd = m.GoToHome(msg, cmd)
		return m, cmd
	}
	m = m.ClearModel()
	m.Window = window.Window()
	switch m.Window.Type() {
	case FormWindow:
		m, cmd = m.HandleForm(msg, cmd, window)
	case SearchableSelectableListWindow, ListSelectionWindow:
		m, cmd = m.HandleTitledList(msg, cmd, m.Window.Name())
	}
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
		if !strings.Contains(choice.Path(), m.Server.State.Dir) {
			path = filepath.Join(m.Server.State.Dir, choice.Path())
		} else {
			path = choice.Path()
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
	default:
		if msg.String() == "g" && m.KeyHack.GetLastKey() == "g" {
			m.Cursor = 0
		}
	}
	return m
}

func (m Model) HandleWindowChangeKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		if m.Window.Name() == Home {
			m.Quitting = true
			return m, cmd
		} else {
			return m.GoToHome(msg, cmd)
		}
	case key.Matches(msg, m.Keys.NewCollection):
		m, cmd = m.SetWindow(msg, cmd, NewCollectionWindow)
	case key.Matches(msg, m.Keys.CreateExport):
		m, cmd = m.SetWindow(msg, cmd, CreateExportWindow)
	case key.Matches(msg, m.Keys.RunExport):
		m, cmd = m.SetWindow(msg, cmd, RunExportWindow)
	case key.Matches(msg, m.Keys.SetTargetSubCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetSubCollectionWindow)
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
	case key.Matches(msg, m.Keys.FuzzySearchFromRoot):
		m, cmd = m.SetWindow(msg, cmd, FuzzySearchRootWindow)
	case key.Matches(msg, m.Keys.FuzzySearchFromCurrent):
		m, cmd = m.SetWindow(msg, cmd, FuzzySearchCurrentWindow)
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetCollectionWindow)
	case key.Matches(msg, m.Keys.BrowseTargetCollection):
		m, cmd = m.SetWindow(msg, cmd, BrowseCollectionWindow)
	case key.Matches(msg, m.Keys.CreateTag):
		m, cmd = m.SetWindow(msg, cmd, NewTagWindow)
	}
	return m, cmd
}

func (m Model) IsWindowKeyMsg(msg tea.KeyMsg) bool {
	switch {
	case key.Matches(msg, m.Keys.FuzzySearchFromRoot), key.Matches(msg, m.Keys.FuzzySearchFromCurrent), key.Matches(msg, m.Keys.BrowseTargetCollection), key.Matches(msg, m.Keys.SetTargetCollection), key.Matches(msg, m.Keys.SetTargetSubCollection), key.Matches(msg, m.Keys.SetTargetSubCollectionRoot), key.Matches(msg, m.Keys.NewCollection), key.Matches(msg, m.Keys.CreateExport), key.Matches(msg, m.Keys.RunExport), key.Matches(msg, m.Keys.CreateTag):
		return true
	default:
		return false
	}
}

// Handle a single key press
func (m Model) HandleDirectoryKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	m = m.HandleStandardMovementKey(msg)
	m, cmd = m.HandleWindowChangeKey(msg, cmd)
	switch {
	case key.Matches(msg, m.Keys.ToggleShowCollections):
		m.ShowCollections = !m.ShowCollections
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
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.CreateQuickTag):
		choice := m.Server.State.Choices[m.Cursor]
		log.Println("creating quick tag for ", choice.Name())
		if !choice.IsDir() {
			m.Server.CreateQuickTag(choice.Name())
		}
		m.Server.UpdateChoices()
	}
	return m, cmd
}

// List selection navigation
func (m Model) HandleListSelectionKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	m = m.HandleStandardMovementKey(msg)
	m, cmd = m.HandleWindowChangeKey(msg, cmd)
	switch {
	case key.Matches(msg, m.Keys.ToggleShowCollections):
		m.ShowCollections = !m.ShowCollections
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.Enter):
		log.Println("handling selectable list enter for ", m.SelectableList)
		switch m.Window.Name() {
		case SetTargetCollectionWindow:
			if collection, ok := m.Server.State.Choices[m.Cursor].(core.CollectionMetadata); ok {
				m.Server.UpdateTargetCollection(collection)
				m, cmd = m.GoToHome(msg, cmd)
			} else {
				log.Fatalf("Invalid list selection item type")
			}
		case RunExportWindow:
			if export, ok := m.Server.State.Choices[m.Cursor].(core.Export); ok {
				m.Server.ExportCollection(m.Server.User.TargetCollection.Id(), export.Id())
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
	m, cmd = m.HandleWindowChangeKey(msg, cmd)
	switch {
	case key.Matches(msg, m.Keys.Up):
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		m.Form.FocusedInput--
		if m.Form.FocusedInput < 0 {
			m.Form.FocusedInput = len(m.Form.Inputs) - 1
		}
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	case key.Matches(msg, m.Keys.Down):
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		m.Form.FocusedInput++
		if m.Form.FocusedInput > len(m.Form.Inputs)-1 {
			m.Form.FocusedInput = 0
		}
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	case key.Matches(msg, m.Keys.InsertMode):
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		m.Form.Writing = true
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
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
		switch m.Window.Name() {
		case NewCollectionWindow:
			m.Server.CreateCollection(m.Form.Inputs[0].Input.Value(), m.Form.Inputs[1].Input.Value())
		case NewTagWindow:
			m.Server.CreateTag(m.Server.State.Choices[m.Cursor].Name(), m.Form.Inputs[0].Input.Value(), m.Form.Inputs[1].Input.Value())
		case CreateExportWindow:
			if len(m.Form.Inputs) < 3 {
				log.Println("not enough inputs for export")
				return m, cmd
			}
			if m.Form.Inputs[0].Input.Value() == "" || m.Form.Inputs[1].Input.Value() == "" || m.Form.Inputs[2].Input.Value() == "" {
				log.Println("please fill out all fields")
				return m, cmd
			}
			var concrete bool
			if strings.HasPrefix(m.Form.Inputs[2].Input.Value(), "t") || m.Form.Inputs[2].Input.Value() == "1" {
				concrete = true
			} else {
				concrete = false
			}
			m.Server.CreateExport(m.Form.Inputs[0].Input.Value(), m.Form.Inputs[1].Input.Value(), concrete)
		}
		m, cmd = m.GoToHome(msg, cmd)
	}
	return m, cmd
}

// Utility function handling searches
func (m Model) FilterListItems() Model {
	var resp []core.SelectableListItem
	switch m.Window.Name() {
	case SetTargetSubCollectionWindow:
		r := m.Server.SearchCollectionSubcollections(m.SearchableSelectableList.Search.Input.Value())
		newArray := make([]core.SelectableListItem, 0)
		for _, item := range r {
			newArray = append(newArray, item)
		}
		resp = newArray
		m.Server.State.Choices = resp
	case Home:
		m.Server.SearchCurrentChoices(m.SearchableSelectableList.Search.Input.Value())
	case FuzzySearchRootWindow:
		m.Server.State.Choices = make([]core.SelectableListItem, 0)
		m.Server.FuzzyFind(m.SearchableSelectableList.Search.Input.Value(), true)
	case FuzzySearchCurrentWindow:
		m.Server.State.Choices = make([]core.SelectableListItem, 0)
		m.Server.FuzzyFind(m.SearchableSelectableList.Search.Input.Value(), false)
	}
	return m
}

// Form writing
func (m Model) HandleFormWritingKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	log.Println("handling form writing key")
	switch {
	case key.Matches(msg, m.Keys.Enter):
		m.Form.Writing = false
		m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
	default:
		var newInput textinput.Model
		newInput, cmd = m.Form.Inputs[m.Form.FocusedInput].Input.Update(msg)
		m.Form.Inputs[m.Form.FocusedInput].Input = newInput
		m.Form.Inputs[m.Form.FocusedInput].Input.Focus()
	}
	return m, cmd
}

// List selection navigation
func (m Model) HandleSearchableListNavKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	m = m.HandleStandardMovementKey(msg)
	m, cmd = m.HandleWindowChangeKey(msg, cmd)
	switch {
	case key.Matches(msg, m.Keys.InsertMode) || key.Matches(msg, m.Keys.SearchBuf):
		m.SearchableSelectableList.Search.Input.Focus()
		m.Cursor = len(m.Server.State.Choices) - 1
		m.Form.Writing = true
	case key.Matches(msg, m.Keys.Enter):
		value := m.SearchableSelectableList.Search.Input.Value()
		switch m.Window.Name() {
		case FuzzySearchRootWindow:
			log.Println("got value ", value)
			if value == "" {
				return m, cmd
			}
			m.Cursor = 0
			m.Server.FuzzyFind(value, true)
			return m, cmd
		case FuzzySearchCurrentWindow:
			if value == "" {
				return m, cmd
			}
			m.Cursor = 0
			m.Server.FuzzyFind(value, false)
			return m, cmd
		case Home:
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
		case SetTargetSubCollectionWindow:
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
			m, cmd = m.GoToHome(msg, cmd)
		}
	case key.Matches(msg, m.Keys.ToggleShowCollections):
		m.ShowCollections = !m.ShowCollections
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	}
	return m, cmd
}

// Searchbar writing
func (m Model) HandleSearchableListWritingKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Enter):
		m.Form.Writing = false
		m.SearchableSelectableList.Search.Input.Blur()
		m = m.FilterListItems()
		m.Cursor = 0
	default:
		var newInput textinput.Model
		newInput, cmd = m.SearchableSelectableList.Search.Input.Update(msg)
		m.SearchableSelectableList.Search.Input = newInput
		m.SearchableSelectableList.Search.Input.Focus()
		switch m.Window.Name() {
		case SetTargetSubCollectionWindow, Home:
			m = m.FilterListItems()
			m.Cursor = 0
		}
	}
	return m, cmd
}

// Form key
func (m Model) HandleSearchableListKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	if m.Form.Writing {
		m, cmd = m.HandleSearchableListWritingKey(msg, cmd)
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
		switch m.Window.Type() {
		case FormWindow:
			m, cmd = m.HandleFormKey(msg, cmd)
		case ListSelectionWindow:
			m, cmd = m.HandleListSelectionKey(msg, cmd)
		case DirectoryWalker:
			m, cmd = m.HandleDirectoryKey(msg, cmd)
		case SearchableSelectableListWindow:
			m, cmd = m.HandleSearchableListKey(msg, cmd)
		}
		if m.Quitting {
			return m, tea.Quit
		}
		m.KeyHack.UpdateLastKey(msg.String())
	}
	return m.SetViewportContent(msg, cmd)
}
