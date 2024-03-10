package core

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

//go:embed create_db.sql
var createSqlCommands []byte

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
	newForm := Form{
		Title:        title,
		Inputs:       inputs,
		Writing:      false,
		FocusedInput: 0,
	}
	return newForm
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
	Path() string
	IsDir() bool
	IsFile() bool
	TaggedDirEntry() (TaggedDirEntry, error)
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
	id             int
	FilePath       string
	CollectionName string
	SubCollection  string
	name           string
}

func NewCollectionTag(id int, name string, filePath string, collectionName string, subCollection string) CollectionTag {
	return CollectionTag{
		id:             id,
		FilePath:       filePath,
		CollectionName: collectionName,
		SubCollection:  subCollection,
		name:           name,
	}
}

func (t CollectionTag) Id() int {
	return t.id
}

func (t CollectionTag) Name() string {
	return t.name
}

func (d CollectionTag) Path() string {
	return d.FilePath
}

func (d CollectionTag) Description() string {
	return d.SubCollection
}

func (t CollectionTag) IsDir() bool {
	return false
}

func (t CollectionTag) IsFile() bool {
	return !t.IsDir()
}

func (t CollectionTag) TaggedDirEntry() (TaggedDirEntry, error) {
	return TaggedDirEntry{}, errors.New("Collection tags do not have collection tags")
}

// A directory entry with associated tags
type TaggedDirEntry struct {
	FilePath string
	Tags     []CollectionTag
	Dir      bool
}

func NewTaggedDirEntry(filePath string, tags []CollectionTag, dir bool) TaggedDirEntry {
	return TaggedDirEntry{
		FilePath: filePath,
		Tags:     tags,
		Dir:      dir,
	}
}

func (d TaggedDirEntry) Id() int {
	return 0
}

func (d TaggedDirEntry) Name() string {
	return path.Base(d.FilePath)
}

func (d TaggedDirEntry) Path() string {
	return d.FilePath
}

func (d TaggedDirEntry) Description() string {
	return d.DisplayTags()
}

func (d TaggedDirEntry) IsDir() bool {
	return d.Dir
}

func (d TaggedDirEntry) IsFile() bool {
	return !d.Dir
}

func (d TaggedDirEntry) TaggedDirEntry() (TaggedDirEntry, error) {
	return d, nil
}

// A string representing the collection tags associated with a directory entry
func (d TaggedDirEntry) DisplayTags() string {
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
	TargetCollection    *CollectionMetadata
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
		CreateSqlCommands: createSqlCommands,
	}
	return &config
}

func (c *Config) SetRoot(root string) {
	root = ExpandPath(root)
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

// A CollectionMetadata
type CollectionMetadata struct {
	id          int
	name        string
	description string
}

func NewCollection(id int, name string, description string) CollectionMetadata {
	return CollectionMetadata{id: id, name: name, description: description}
}

// Requirement for a listSelectionItem
func (c CollectionMetadata) Id() int {
	return c.id
}

// Requirement for a listSelectionItem
func (c CollectionMetadata) Name() string {
	return c.name
}

func (c CollectionMetadata) Path() string {
	return ""
}

// Requirement for a listSelectionItem
func (c CollectionMetadata) Description() string {
	return c.description
}

// Requirement for a listSelectionItem
func (c CollectionMetadata) IsDir() bool {
	return false
}

// Requirement for a listSelectionItem
func (c CollectionMetadata) IsFile() bool {
	return false
}

func (c CollectionMetadata) TaggedDirEntry() (TaggedDirEntry, error) {
	return TaggedDirEntry{}, errors.New("Collection metadata doesnt have collection tags")
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

func (s SubCollection) SetName(name string) SubCollection {
	s.name = name
	return s
}

func (s SubCollection) Path() string {
	return ""
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

func (s SubCollection) TaggedDirEntry() (TaggedDirEntry, error) {
	return TaggedDirEntry{}, errors.New("Subcollection doesnt have collection tags")
}

type Export struct {
	id        int
	name      string
	outputDir string
	concrete  bool
}

func NewExport(id int, name string, outputDir string, concrete bool) Export {
	return Export{id: id, name: name, outputDir: outputDir, concrete: concrete}
}

func (e Export) Id() int {
	return e.id
}

func (e Export) IsDir() bool {
	return true
}

func (e Export) IsFile() bool {
	return false
}

func (e Export) Name() string {
	return e.name
}

func (e Export) Path() string {
	return e.outputDir
}

func (e Export) Description() string {
	if e.concrete {
		return "concrete"
	} else {
		return "abstract"
	}
}

func (e Export) TaggedDirEntry() (TaggedDirEntry, error) {
	return TaggedDirEntry{}, errors.New("Exports do not have collection tags")
}
