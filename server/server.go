package server

import (
	"database/sql"
	"errors"
	"flag"

	// "io/fs"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"

	// Database
	_ "github.com/mattn/go-sqlite3"

	"github.com/jesses-code-adventures/excavator/audio"
	"github.com/jesses-code-adventures/excavator/core"
)

//////////////////////// LOCAL SERVER ////////////////////////

type State struct {
	Choices            []core.SelectableListItem
	choiceChannel      chan core.SelectableListItem
	CollectionTags     func(path string) []core.CollectionTag
	Dir                string
	MatchingIndexes    []int
	localSearchChannel chan string
	Root               string
}

func NewState(root string, currentDir string, collectionTags func(path string) []core.CollectionTag) *State {
	choiceChannel := make(chan core.SelectableListItem)
	navState := State{
		Root:            root,
		Dir:             currentDir,
		choiceChannel:   choiceChannel,
		Choices:         make([]core.SelectableListItem, 0),
		CollectionTags:  collectionTags,
		MatchingIndexes: make([]int, 0),
	}
	go navState.Run()
	return &navState
}

func (n *State) Run() {
	for {
		select {
		case choice := <-n.choiceChannel:
			n.Choices = append(n.Choices, choice)
		case search := <-n.localSearchChannel:
			n.SearchCurrentChoices(search)
		}
	}
}

func (n *State) LocalSearch(search string) {
	n.localSearchChannel <- search
}

func (n *State) SearchCurrentChoices(search string) {
	indexes := make([]int, 0)
	chunks := strings.Split(search, " ")
	for i, choice := range n.Choices {
		noMatch := false
		for _, chunk := range chunks {
			if !strings.Contains(strings.ToLower(choice.Name()), strings.ToLower(chunk)) {
				noMatch = true
				continue
			}
		}
		if noMatch == true {
			continue
		}
		indexes = append(indexes, i)
	}
	n.MatchingIndexes = indexes
}

func (n *State) GetNextMatchingIndex(position int) int {
	if len(n.MatchingIndexes) == 0 {
		return -1
	}
	for _, index := range n.MatchingIndexes {
		if index > position {
			return index
		}
	}
	return n.MatchingIndexes[0]
}

func (n *State) GetPreviousMatchingIndex(position int) int {
	if len(n.MatchingIndexes) == 0 {
		return -1
	}
	for i := len(n.MatchingIndexes) - 1; i >= 0; i-- {
		if n.MatchingIndexes[i] < position {
			return n.MatchingIndexes[i]
		}
	}
	return len(n.MatchingIndexes) - 1

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
		n.Choices = append(n.Choices, core.TaggedDirEntry{FilePath: "..", Tags: make([]core.CollectionTag, 0), Dir: true})
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
	if err != nil {
		log.Fatalf("Failed to read samples directory in ListDirEntries: %v", err)
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
		samples = append(samples, core.NewTaggedDirEntry(path.Join(f.Dir, file.Name()), matchedTags, isDir))
	}
	return samples
}

// Get the full path of the current directory
func (n *State) GetCurrentDirPath() string {
	return filepath.Join(n.Root, n.Dir)
}

// Change the current directory
func (n *State) ChangeDir(dir string) {
	n.Dir = filepath.Join(n.Dir, dir)
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
	Config *core.Config
	Db     *sql.DB
	Flags  *Flags
	Player *audio.Player
	State  *State
	User   core.User
}

func (s *Server) HandleUserArg(userCliArg *string) (core.User, error) {
	var user core.User
	users := s.GetUsers(userCliArg)
	if len(*userCliArg) == 0 && len(users) == 0 {
		return core.User{}, errors.New("No users found")
	}
	if len(*userCliArg) == 0 && len(users) > 0 {
		user = users[0]
		return user, nil
	}
	if len(*userCliArg) > 0 && len(users) == 0 {
		id := s.CreateUser(*userCliArg)
		if id == 0 {
			log.Fatal("Failed to create user")
		}
		user = s.GetUser(id)
		return user, nil
	}
	if len(*userCliArg) > 0 && len(users) > 0 {
		for _, u := range users {
			if u.Name == *userCliArg {
				return u, nil
			}
		}
		id := s.CreateUser(*userCliArg)
		user = s.GetUser(id)
		return user, nil
	}
	log.Fatal("We should never get here")
	return user, nil
}

func (s *Server) GetCollectionSubcollections() []core.SubCollection {
	statement := `select distinct sub_collection from CollectionTag where collection_id = ? order by sub_collection asc`
	rows, err := s.Db.Query(statement, s.User.TargetCollection.Id())
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getCollectionSubcollections: %v", err)
	}
	defer rows.Close()
	subCollections := make([]core.SubCollection, 0)
	for rows.Next() {
		var subCollection string
		if err := rows.Scan(&subCollection); err != nil {
			log.Fatalf("Failed to scan row in get collection subcollections: %v", err)
		}
		subCollections = append(subCollections, core.NewSubCollection(subCollection))
	}
	return subCollections
}

func (s *Server) SearchCurrentChoices(search string) {
	newChoices := make([]core.SelectableListItem, 0)
	chunks := strings.Split(search, " ")
	for _, choice := range s.State.Choices {
		noMatch := false
		for _, chunk := range chunks {
			if !strings.Contains(choice.Name(), chunk) {
				noMatch = true
				continue
			}
		}
		if noMatch == true {
			continue
		}
		newChoices = append(newChoices, choice)
	}
	s.State.Choices = newChoices
}

func (s *Server) SearchCollectionSubcollections(search string) []core.SubCollection {
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
	subCollections := make([]core.SubCollection, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("Failed to scan row in search collection subcollections: %v", err)
		}
		subCollection := core.NewSubCollection(name)
		subCollections = append(subCollections, subCollection)
	}
	log.Printf("subcollections from db : %v", subCollections)
	return subCollections
}

type Flags struct {
	Data       string
	DbFileName string
	LogFile    string
	Root       string
	User       string
	Watch      bool
}

func ParseFlags() *Flags {
	var data = flag.String("data", "~/.excavator-tui", "Local data storage path")
	var dbFileName = flag.String("db", "excavator", "Database file name")
	var logFile = flag.String("log", "logfile", "Log file name")
	var samples = flag.String("root", "~/Library/Audio/Sounds/Samples", "Root samples directory")
	var userArg = flag.String("user", "", "User name to launch with")
	var watch = flag.Bool("watch", false, "Watch for changes in the samples directory")
	flag.Parse()
	return &Flags{Data: core.ExpandHomeDir(*data), DbFileName: *dbFileName, LogFile: *logFile, Root: core.ExpandHomeDir(*samples), User: *userArg, Watch: *watch}
}

// Part of newServer constructor
func (s Server) HandleRootConstruction() (Server, error) {
	if s.User.Root == "" && s.Config.Root == "" {
		return Server{}, errors.New("no root found")
	} else if s.Config.Root == "" {
		s.Config.Root = s.User.Root
	} else if s.User.Root == "" {
		s.User.Root = s.Config.Root
		s.UpdateRootInDb(s.Config.Root)
	} else if s.User.Root != s.Config.Root {
		log.Println("launched with temporary root ", s.Config.Root)
		s.User.Root = s.Config.Root
	}
	return s, nil
}

// Construct the server
func NewServer(audioPlayer *audio.Player, flags *Flags) Server {
	config := core.NewConfig(flags.Data, flags.Root, flags.DbFileName)
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
		Config: config,
		Flags:  flags,
	}
	return s
}

func (s Server) AddUserAndRoot() (Server, error){
	log.Println("about to handle user arg")
	user, err := s.HandleUserArg(&s.Flags.User)
	if err != nil {
		log.Println("error in handle user arg")
		return s, err
	}
	s.User = user
	log.Println("about to handle root construction")
	s, err = s.HandleRootConstruction()
	if err != nil {
		return s, err
	}
	s.State = NewState(s.Config.Root, s.Config.Root, s.GetDirectoryTags)
	s.State.UpdateChoices()
	return s, nil
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
func (s *Server) UpdateTargetCollection(collection core.CollectionMetadata) {
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

func (s *Server) CreateExport(name string, outputDir string, concrete bool) int {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(err)
		}
	}
	if len(name) == 0 {
		return 0
	}
	res, err := s.Db.Exec("insert or ignore into Export (user_id, name, output_dir, concrete) values (?, ?, ?, ?)", s.User.Id, name, outputDir, concrete)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in createTagInDb: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

func (s *Server) GetExports() []core.SelectableListItem {
	statement := `select id, name, output_dir, concrete from Export where user_id = ? order by name desc`
	rows, err := s.Db.Query(statement, s.User.Id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getExports: %v", err)
	}
	defer rows.Close()
	exports := make([]core.SelectableListItem, 0)
	for rows.Next() {
		var id int
		var name string
		var outputDir string
		var concrete bool
		if err := rows.Scan(&id, &name, &outputDir, &concrete); err != nil {
			log.Fatalf("Failed to scan row in getExports: %v", err)
		}
		exports = append(exports, core.NewExport(id, name, outputDir, concrete))
	}
	return exports
}

func (s *Server) GetExport(id int) core.Export {
	statement := `select name, output_dir, concrete from Export where id = ?`
	row := s.Db.QueryRow(statement, id)
	var name string
	var outputDir string
	var concrete bool
	if err := row.Scan(&name, &outputDir, &concrete); err != nil {
		log.Fatalf("Failed to scan row in getExport: %v", err)
	}
	return core.NewExport(id, name, outputDir, concrete)
}

func (s *Server) ExportCollection(collectionId int, exportId int) {
	export := s.GetExport(exportId)
	tags := s.GetCollectionTags(collectionId)
	var copyFn func(source string, destination string) error
	if export.Description() == "concrete" {
		copyFn = os.Link
	} else {
		copyFn = os.Symlink
	}
	for _, tag := range tags {
		source := tag.FilePath
		_, err := os.Stat(source)
		if err != nil {
			log.Fatalf("Source doesn't exist: %v", err)
		}
		dir := path.Join(export.Path(), export.Name(), tag.CollectionName, tag.SubCollection)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
		destination := path.Join(dir, path.Base(tag.FilePath)) // Todo: use name field from collection tag
		if _, err := os.Stat(destination); os.IsNotExist(err) {
			if err := copyFn(source, destination); err != nil {
				log.Fatalf("Failed to create link: %v", err)
			}
		}
	}
}

// ////////////////////// DATABASE ENDPOINTS ////////////////////////
// Get collection tags associated with a directory
func (s *Server) GetCollectionTags(id int) []core.CollectionTag {
	statement := `select ct.id, t.file_path, col.name, ct.sub_collection,
ct.name
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where ct.id = ? order by ct.sub_collection asc, ct.name asc`
	rows, err := s.Db.Query(statement, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]core.CollectionTag, 0)
	for rows.Next() {
		var filePath, collectionName, subCollection, name string
		var id int
		if err := rows.Scan(&id, &filePath, &collectionName, &subCollection, &name); err != nil {
			log.Fatalf("Failed to scan row in getcollectiontags: %v", err)
		}
		tags = append(tags, core.NewCollectionTag(id, name, filePath, collectionName, subCollection))
	}
	return tags
}

// Get collection tags associated with a directory
func (s *Server) GetCollectionTagsAsListItem(id int) []core.SelectableListItem {
	statement := `select ct.id, t.file_path, col.name, ct.sub_collection,
ct.name
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where col.id = ? order by ct.sub_collection asc, ct.name asc`
	rows, err := s.Db.Query(statement, id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement: %v", err)
	}
	defer rows.Close()
	tags := make([]core.SelectableListItem, 0)
	for rows.Next() {
		var filePath, collectionName, subCollection, name string
		var id int
		if err := rows.Scan(&id, &filePath, &collectionName, &subCollection, &name); err != nil {
			log.Fatalf("Failed to scan row in getcollectiontags: %v", err)
		}
		tags = append(tags, core.NewCollectionTag(id, name, filePath, collectionName, subCollection))
	}
	return tags
}

// Get collection tags associated with a directory
func (s *Server) GetDirectoryTags(dir string) []core.CollectionTag {
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
			log.Fatalf("Failed to scan row in getcollectiontags: %v", err)
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
	statement := `select t.file_path, ct.id, col.name, ct.sub_collection
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
	for rows.Next() {
		var filePath, collectionName, subCollection string
		var id int
		if err := rows.Scan(&filePath, &id, &collectionName, &subCollection); err != nil {
			log.Fatalf("Failed to scan row in fuzzy find collection tags: %v", err)
		}
		log.Printf("filepath: %s, collection name: %s, subcollection: %s", filePath, collectionName, subCollection)
		tags = append(tags, core.NewCollectionTag(id, path.Base(filePath), filePath, collectionName, subCollection))
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
			log.Fatalf("Failed to scan row in search collection tags: %v", err)
		}
		log.Printf("filepath: %s, collection name: %s, subcollection: %s", filePath, collectionName, subCollection)
		tags = append(tags, core.CollectionTag{FilePath: filePath, CollectionName: collectionName, SubCollection: subCollection})
	}
	return tags
}

func (s *Server) GetUser(id int) core.User {
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
		log.Fatalf("Failed to scan row in getuser: %v", err)
	}
	var selectedCollection *core.CollectionMetadata
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
	log.Println("In get users")
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
			log.Fatalf("Failed to scan row in getusers: %v", err)
		}
		var selectedCollection *core.CollectionMetadata
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

func (s *Server) SetUserFromInput(user string) error {
	if len(user) == 0 {
		return errors.New("No user entered")
	}
	existing := s.GetUsers(&user)
	if len(existing) == 0 {
		id := s.CreateUser(user)
		s.User = s.GetUser(id)
	} else {
		s.User = existing[0]
	}
	return nil
}

func (s *Server) SetRootFromInput(root string) error {
	if len(root) == 0 {
		return errors.New("No root entered")
	}
	root = core.ExpandHomeDir(root)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return errors.New("Root does not exist")
	}
	s.State = NewState(root, root, s.GetDirectoryTags)
	s.State.UpdateChoices()
	return nil
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
	s.UpdateSelectedCollectionInDb(int(id))
	s.UpdateTargetCollection(core.NewCollection(int(id), name, description))
	return int(id)
}

// Get all collections for the current user
func (s *Server) GetCollections() []core.CollectionMetadata {
	statement := `select id, name, description from Collection where user_id = ?`
	rows, err := s.Db.Query(statement, s.User.Id)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getCollections: %v", err)
	}
	defer rows.Close()
	collections := make([]core.CollectionMetadata, 0)
	for rows.Next() {
		var id int
		var name string
		var description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			log.Fatalf("Failed to scan row in getcollections: %v", err)
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
	if collectionId == 0 {
		log.Fatal("Collection id is 0")
	}
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

type CollectionItem struct {
	id            int
	name          string
	path          string
	subCollection string
}

func NewCollectionItem(path string, subCollection string) CollectionItem {
	name := filepath.Join("%s%s", subCollection, strings.Split(filepath.Base(path), ".")[0])
	return CollectionItem{
		id:            0,
		name:          name,
		path:          path,
		subCollection: subCollection,
	}
}

func (c CollectionItem) Id() int {
	return c.id
}

func (c CollectionItem) Name() string {
	return c.name
}

func (c CollectionItem) Path() string {
	return c.path
}

func (c CollectionItem) Description() string {
	return ""
}

func (c CollectionItem) IsDir() bool {
	return false
}

func (c CollectionItem) IsFile() bool {
	return true
}

func (s *Server) GetCollection(collectionId int) []core.SelectableListItem {
	statement := `select ct.id, t.file_path, ct.sub_collection
from CollectionTag ct
left join Collection col
on ct.collection_id = col.id
left join Tag t on ct.tag_id = t.id
where ct.collection_id like ?
order by ct.sub_collection asc, t.file_path asc`
	rows, err := s.Db.Query(statement, collectionId)
	if err != nil {
		log.Fatalf("Failed to execute SQL statement in getCollection: %v", err)
	}
	defer rows.Close()
	tags := make([]core.SelectableListItem, 0)
	for rows.Next() {
		var id int
		var filePath, subCollection string
		if err := rows.Scan(&id, &filePath, &subCollection); err != nil {
			log.Fatalf("Failed to scan row in getcollection: %v", err)
		}
		tags = append(tags, NewCollectionItem(filePath, subCollection))
	}
	return tags
}
