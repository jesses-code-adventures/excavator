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
	searchInputHeight := 2 // Assuming the search input height is approximately 2 lines
	verticalPadding := 2   // Adjust based on your app's padding around the viewport
	// Calculate available height differently if in SearchableSelectableList mode
	if m.Window.Type() == SearchableSelectableListWindow {
		m.Viewport.Height = msg.Height - headerHeight - footerHeight - searchInputHeight - verticalPadding
	} else {
		m.Viewport.Height = msg.Height - headerHeight - footerHeight - verticalPadding
	}
	m.Viewport.Width = msg.Width
	m.Ready = true
	return m
}

// Handle viewport positioning
func (m Model) EnsureCursorVerticallyCentered() viewport.Model {
	if m.Window.Type() != DirectoryWalker {
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
		m.Viewport = m.EnsureCursorVerticallyCentered()
		m.Viewport.SetContent(m.Window.Type().DirectoryView(
			m.Server.State.Choices,
			m.Cursor,
			m.Viewport.Width,
			m.ShowCollections,
			m.SearchableSelectableList.Search.Input,
		))
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
	case FuzzySearchRootWindow:
		files := m.Server.FuzzyFind("", true)
		for _, file := range files {
			m.Server.State.Choices = append(m.Server.State.Choices, file)
		}
	case FuzzySearchCurrentWindow:
		files := m.Server.FuzzyFind("", false)
		for _, file := range files {
			m.Server.State.Choices = append(m.Server.State.Choices, file)
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
	log.Printf("got window name %s", window.String())
	if m.Window.Name() == window {
		log.Println("going back to main window")
		m, cmd = m.GoToMainWindow(msg, cmd)
		return m, cmd
	}
	m = m.ClearModel()
	m.Window = window.Window()
	log.Println("have set module window type to ", m.Window.Type())
	switch m.Window.Type() {
	case FormWindow:
		m, cmd = m.HandleForm(msg, cmd, window)
	case SearchableSelectableListWindow, ListSelectionWindow:
		log.Println("going to window ", window)
		m, cmd = m.HandleTitledList(msg, cmd, m.Window.Name())
	}
	m.Cursor = 0
	return m, cmd
}

// // Main handler to be called any time the window changes
// func (m Model) SetWindowType(msg tea.Msg, cmd tea.Cmd, windowType WindowType, title string) (Model, tea.Cmd) {
// 	if m.Window.Type() == windowType && m.SearchableSelectableList.Title == title {
// 		m, cmd = m.GoToMainWindow(msg, cmd)
// 		return m, cmd
// 	}
// 	switch windowType {
// 	case DirectoryWalker:
// 		m = m.ClearModel()
// 		m, cmd = m.GoToMainWindow(msg, cmd)
// 	case FormWindow:
// 		if title == "" {
// 			log.Fatalf("Title required for forms")
// 		}
// 		m, cmd = m.HandleForm(msg, cmd, title)
// 	case ListSelectionWindow:
// 		m = m.ClearModel()
// 		if title == "" {
// 			log.Fatalf("Title required for lists")
// 		}
// 		m, cmd = m.HandleTitledList(msg, cmd, title)
// 	case SearchableSelectableListWindow:
// 		m = m.ClearModel()
// 		if title == "" {
// 			log.Fatalf("Title required for lists")
// 		}
// 		m, cmd = m.HandleTitledList(msg, cmd, title)
// 	default:
// 		log.Fatalf("Invalid window type")
// 	}
//     log.Println("setting window type to ", windowType)
// 	// m.Window.SetType(windowType)
//     m.Window = NewWindow(windowType)
//     log.Println("window type is now ", m.Window.Type())
// 	m.Cursor = 0
// 	return m, cmd
// }

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
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetCollectionWindow)
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.CreateQuickTag):
		choice := m.Server.State.Choices[m.Cursor]
		log.Println("creating quick tag for ", choice.Name())
		if !choice.IsDir() {
			m.Server.CreateQuickTag(choice.Name())
		}
		m.Server.UpdateChoices()
	case key.Matches(msg, m.Keys.CreateTag):
		m, cmd = m.SetWindow(msg, cmd, NewTagWindow)
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
		m, cmd = m.SetWindow(msg, cmd, NewCollectionWindow)
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetCollectionWindow)
	case key.Matches(msg, m.Keys.SetTargetSubCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetSubCollectionWindow)
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
	case key.Matches(msg, m.Keys.FuzzySearchFromRoot):
		m, cmd = m.SetWindow(msg, cmd, FuzzySearchRootWindow)
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.Enter):
		log.Println("handling selectable list enter for ", m.SelectableList)
		switch m.Window.Name() {
		case SetTargetCollectionWindow:
			if collection, ok := m.Server.State.Choices[m.Cursor].(core.CollectionMetadata); ok {
				m.Server.UpdateTargetCollection(collection)
				m, cmd = m.GoToMainWindow(msg, cmd)
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
	case key.Matches(msg, m.Keys.Quit), key.Matches(msg, m.Keys.NewCollection), key.Matches(msg, m.Keys.SetTargetSubCollection), key.Matches(msg, m.Keys.FuzzySearchFromRoot): // TODO: make these nav properly
		m, cmd = m.GoToMainWindow(msg, cmd)
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetCollectionWindow)
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
		case "create export":
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
		m, cmd = m.GoToMainWindow(msg, cmd)
	}
	return m, cmd
}

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

// Form writing
func (m Model) HandleFormWritingKey(msg tea.KeyMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		m.Form.Writing = false
		if m.Window.Type() == SearchableSelectableListWindow {
			m.SearchableSelectableList.Search.Input.Blur()
			m = m.FilterListItems()
			log.Printf("filtered items in form writing key quit %v", m.Server.State.Choices)
			m.Cursor = 0
		} else {
			m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		}
	case key.Matches(msg, m.Keys.Enter):
		m.Form.Writing = false
		if m.Window.Type() == SearchableSelectableListWindow {
			m.SearchableSelectableList.Search.Input.Blur()
			m = m.FilterListItems()
			log.Printf("filtered items in form writing key enter %v", m.Server.State.Choices)
			m.Cursor = 0
		} else {
			m.Form.Inputs[m.Form.FocusedInput].Input.Blur()
		}
	default:
		var newInput textinput.Model
		if m.Window.Type() == SearchableSelectableListWindow {
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
		m, cmd = m.SetWindow(msg, cmd, NewCollectionWindow)
	case key.Matches(msg, m.Keys.SetTargetSubCollection):
		m.Form = core.GetTargetSubCollectionForm()
		m, cmd = m.SetWindow(msg, cmd, SetTargetSubCollectionWindow)
	case key.Matches(msg, m.Keys.SetTargetSubCollectionRoot):
		m.Server.UpdateTargetSubCollection("")
	case key.Matches(msg, m.Keys.SetTargetCollection):
		m, cmd = m.SetWindow(msg, cmd, SetTargetCollectionWindow)
	case key.Matches(msg, m.Keys.InsertMode) || key.Matches(msg, m.Keys.SearchBuf):
		m.SearchableSelectableList.Search.Input.Focus()
		m.Form.Writing = true
	case key.Matches(msg, m.Keys.ToggleAutoAudition):
		m.Server.UpdateAutoAudition(!m.Server.User.AutoAudition)
	case key.Matches(msg, m.Keys.Enter):
		value := m.SearchableSelectableList.Search.Input.Value()
		switch m.Window.Name() {
		case FuzzySearchRootWindow:
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
		switch m.Window.Type() {
		case FormWindow:
			m, cmd = m.HandleFormKey(msg, cmd)
		case ListSelectionWindow:
			m, cmd = m.HandleListSelectionKey(msg, cmd)
		case DirectoryWalker:
			m, cmd = m.HandleDirectoryKey(msg, cmd)
			if m.Quitting {
				return m, tea.Quit
			}
		case SearchableSelectableListWindow:
			m, cmd = m.HandleSearchableListKey(msg, cmd)
		}
		m.KeyHack.UpdateLastKey(msg.String())
	}
	return m.SetViewportContent(msg, cmd)
}
