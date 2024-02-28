package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// Different window types
type WindowType int

const (
	DirectoryWalker WindowType = iota
	FormWindow
	ListSelectionWindow
	SearchableSelectableListWindow
)

func (w WindowType) String() string {
	return [...]string{"DirectoryWalker", "FormWindow", "ListSelectionWindow", "SearchableSelectableList"}[w]
}

// A singular form input control
type FormInput struct {
	Name  string
	Input textinput.Model
}

// Constructor for a form input
func NewFormInput(name string) FormInput {
	return FormInput{
		Name:  name,
		Input: textinput.New(),
	}
}

// A generic Form
type Form struct {
	Title        string
	Inputs       []FormInput
	Writing      bool
	FocusedInput int
}

// A form constructor
func NewForm(title string, inputs []FormInput) Form {
	return Form{
		Title:        title,
		Inputs:       inputs,
		Writing:      false,
		FocusedInput: 0,
	}
}

//// Form implementations ////

// Get the inputs for the new collection form
func GetNewCollectionInputs() []FormInput {
	return []FormInput{
		NewFormInput("name"),
		NewFormInput("description"),
	}
}

// Get the new collection form
func GetNewCollectionForm() Form {
	return NewForm("create collection", GetNewCollectionInputs())
}

// Get the inputs for the new collection form
func GetSearchInput() []FormInput {
	return []FormInput{
		NewFormInput("search"),
	}
}

// Get the new collection form
func GetTargetSubCollectionForm() Form {
	return NewForm("set target subcollection", GetSearchInput())
}

// Get the new collection form
func GetFuzzySearchRootForm() Form {
	return NewForm("fuzzy search from root", GetSearchInput())
}

// Get the inputs for the new collection form
func GetCreateCollectionTagInputs(defaultName string, defaultSubCollection string) []FormInput {
	name := NewFormInput("name")
	name.Input.SetValue(defaultName)
	subcollection := NewFormInput("subcollection")
	subcollection.Input.SetValue(defaultSubCollection)
	return []FormInput{
		name,
		subcollection,
	}
}

// Get the new collection form
func GetCreateTagForm(defaultName string, defaultSubCollection string) Form {
	return NewForm("create tag", GetCreateCollectionTagInputs(defaultName, defaultSubCollection))
}

/// List selection ///

// Interface for list selection items so the list can easily be reused
type SelectableListItem interface {
	Id() int
	Name() string
	Description() string
	IsDir() bool
	IsFile() bool
}

// A list where a single item can be selected
type SelectableList struct {
	Title string
}

// A constructor for a selectable list
type SearchableSelectableList struct {
	Title  string
	Search FormInput
}

func NewSearchableList(title string) SearchableSelectableList {
	return SearchableSelectableList{
		Title: title,
		Search: FormInput{
			Name:  "search",
			Input: textinput.New(),
		},
	}
}

// A connection between a tag and a collection
type CollectionTag struct {
	FilePath       string
	CollectionName string
	SubCollection  string
}

// A directory entry with associated tags
type TaggedDirentry struct {
	Path string
	Tags []CollectionTag
	Dir  bool
}

func (d TaggedDirentry) Id() int {
	return 0
}

func (d TaggedDirentry) Name() string {
	return d.Path
}

func (d TaggedDirentry) Description() string {
	return d.DisplayTags()
}

func (d TaggedDirentry) IsDir() bool {
	return d.Dir
}

func (d TaggedDirentry) IsFile() bool {
	return !d.Dir
}

// A string representing the collection tags associated with a directory entry
func (d TaggedDirentry) DisplayTags() string {
	first := true
	resp := ""
	for _, tag := range d.Tags {
		if first {
			resp = fmt.Sprintf("%s: %s", tag.CollectionName, tag.SubCollection)
			first = false
		} else {
			resp = fmt.Sprintf("%s, %s: %s", resp, tag.CollectionName, tag.SubCollection)
		}
	}
	return resp
}

// A User
type User struct {
	Id                  int
	Name                string
	AutoAudition        bool
	TargetCollection    *Collection
	TargetSubCollection string
	Root                string
}

// Struct holding the app's configuration
type Config struct {
	Data              string
	Root              string
	DbFileName        string
	CreateSqlCommands []byte
}

// Constructor for the Config struct
func NewConfig(data string, root string, dbFileName string) *Config {
	log.Printf("data: %v, samples: %v", data, root)
	sqlCommands, err := os.ReadFile("sql_commands/create_db.sql")
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
	config := Config{
		Data:              data,
		Root:              root,
		DbFileName:        dbFileName,
		CreateSqlCommands: sqlCommands,
	}
	return &config
}

func (c *Config) SetRoot(root string) {
	root = ExpandHomeDir(root)
	c.Root = root
}

func CreateDirectories(dir string) {
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
func (c *Config) CreateDataDirectory() {
	CreateDirectories(c.Data)
}

// Standardized file structure for the database file
func (c *Config) GetDbPath() string {
	if c.DbFileName == "" {
		c.DbFileName = "excavator"
	}
	if !strings.HasSuffix(c.DbFileName, ".db") {
		c.DbFileName = c.DbFileName + ".db"
	}
	return filepath.Join(c.Data, c.DbFileName)
}

// A Collection
type Collection struct {
	id          int
	name        string
	description string
}

func NewCollection(id int, name string, description string) Collection {
	return Collection{id: id, name: name, description: description}
}

// Requirement for a listSelectionItem
func (c Collection) Id() int {
	return c.id
}

// Requirement for a listSelectionItem
func (c Collection) Name() string {
	return c.name
}

// Requirement for a listSelectionItem
func (c Collection) Description() string {
	return c.description
}

// Requirement for a listSelectionItem
func (c Collection) IsDir() bool {
	return false
}

// Requirement for a listSelectionItem
func (c Collection) IsFile() bool {
	return false
}

type SubCollection struct {
	name string
}

func NewSubCollection(name string) SubCollection {
	return SubCollection{name: name}
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
