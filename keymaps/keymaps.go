package keymaps

import (
	"github.com/charmbracelet/bubbles/key"
)

// ////////////////////// KEYMAPS ////////////////////////
// I know this is bad but i want gg
type KeymapHacks struct {
	lastKey string
}

// Track this keystroke so it can be checked in the next one
func (k *KeymapHacks) UpdateLastKey(key string) {
	k.lastKey = key
}

// Get the previous keystroke
func (k *KeymapHacks) GetLastKey() string {
	return k.lastKey
}

// For jumping to the top of the list
func (k *KeymapHacks) LastKeyWasG() bool {
	return k.lastKey == "g"
}

// All possible keymap bindings
type KeyMap struct {
	Up                         key.Binding
	Down                       key.Binding
	Quit                       key.Binding
	JumpUp                     key.Binding
	JumpDown                   key.Binding
	JumpBottom                 key.Binding
	Audition                   key.Binding
	SearchBuf                  key.Binding
	Enter                      key.Binding
	NewCollection              key.Binding
	SetTargetCollection        key.Binding
	InsertMode                 key.Binding
	ToggleAutoAudition         key.Binding
	AuditionRandom             key.Binding
	CreateQuickTag             key.Binding
	CreateTag                  key.Binding
	SetTargetSubCollectionRoot key.Binding
	SetTargetSubCollection     key.Binding
	FuzzySearchFromRoot        key.Binding
	FuzzySearchFromCurrent     key.Binding
	ToggleShowCollections      key.Binding
	CreateExport               key.Binding
	RunExport                  key.Binding
	BrowseTargetCollection     key.Binding
	NextLocalSearchResult      key.Binding
	PreviousLocalSearchResult  key.Binding
	ShowHelp                   key.Binding
}

// The actual help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Audition, k.CreateQuickTag, k.CreateExport, k.RunExport, k.ShowHelp}
}

// Empty because not using
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.JumpUp, k.JumpDown, k.JumpBottom},
		{k.Audition, k.AuditionRandom, k.ToggleAutoAudition, k.ToggleShowCollections},
		{k.NewCollection, k.SetTargetCollection, k.SetTargetSubCollection, k.BrowseTargetCollection},
		{k.CreateQuickTag, k.CreateTag, k.CreateExport, k.RunExport},
		{k.SearchBuf, k.FuzzySearchFromRoot, k.FuzzySearchFromCurrent, k.InsertMode},
		{k.NextLocalSearchResult, k.PreviousLocalSearchResult, k.Quit},
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
		key.WithKeys("C"),
		key.WithHelp("C", "new collection"),
	),
	SetTargetSubCollectionRoot: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "no target subecollection"),
	),
	SetTargetSubCollection: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "target subcollection"),
	),
	SetTargetCollection: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "set target collection"),
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
	FuzzySearchFromCurrent: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "search sounds from current dir"),
	),
	ToggleShowCollections: key.NewBinding(
		key.WithKeys("K"),
		key.WithHelp("K", "show collections"),
	),
	CreateExport: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "create export"),
	),
	RunExport: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "run export"),
	),
	BrowseTargetCollection: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "browse target collection"),
	),
	NextLocalSearchResult: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next local search result"),
	),
	PreviousLocalSearchResult: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "previous local search result"),
	),
	ShowHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "show help"),
	),
}
