# excavator
a sample-browsing tui. for music producers with large sample libraries who are comfortable in the terminal.

## goal
i find managing samples frustrating, especially when it comes to keeping sample libraries synced across samplers.

this tool aims to create a natural experience for exploring and categorising your sample library for easy re-exporting when you get new devices, or re-organising the same set of samples repeatedly in your daw.

## experience
the user should run the app from their cli by calling excavator run. you should also be able to cat a collection from the CLI directly.

users that have already configured their root sample library should find themselves directly in the sample browser after running excavator run.

the standard workflow should be like browsing a file directory in netrw but you can auto-audition samples and you can randomly audition samples in the directory.

while browsing samples, you should just be able to "write a tag" on a sample and have the sample linked to a sub-directory of your collection.

## terminology
- **the root:** this is the user-defined root sample library where their sample browsing experience will start when they open the app
- **tag:** a path to a sample in the root.
- **collection tag:** links a tag to a collection. name can be customized and a child directory ("subcollection") exists as part of the object to be used in exports.
- **collection:** a group of collection tags with a name and a description - these will be fed to exports.
- **export:** taking a collection and copying all the files pointed to by the tags to a given directory. can be done in symlink or copy mode (default symlink).

## features
- [ ] first launch should prompt user to create a "root sample library".
- [ ] if the application is launched with no root directory and a directory flag, the user should be asked whether they'd like to make the given directory the root.
- [x] launching with a --data path sets the directory where long lived data, including the db file and the logfile is stored.
- [x] launching with a --root launches the program with a temporary root, lasting until the session is closed.
- [x] launching with a --user flag creates a new user whose name is the argument.
- [x] launching with a --db flag sets the filename of the sqlite .db file. the extension does not need to be included with the argument.
- [x] create collections, which are ephemeral directories stored in (maybe apple pickle?) data types designed to be able to collection to a concrete directory you can drag-and-drop elsewhere.
- [x] create tags that assign the samples to sub-directory in a collection. these should not copy the file, but create a reference in the locally stored file.
- [x] browse samples using J/up arrow and K/down arrow.
- [x] samples should play asynchronously so the user can continue browsing while a sample plays
- [x] support for ctrl-D, ctrl-U, G and gg vim functions should exist.
- [x] press q to quit.
- [x] press a to audition a sample you're hovering over.
- [x] press r to jump to and audition a random sample from the current directory.
- [x] press t to create a tag in the current target collection and target directory.
- [x] press shift-T to create a tag in the current target collection where the tag name and directory is editable.
- [x] press shift-A to toggle auto-audition mode.
- [x] press shift-N to create a new collection.
- [x] press shift-C to change the target collection.
- [x] press shift-D to change the target directory in the current collection. this should be a searchable list, and if enter is pressed with no matches it should create a new one.
- [ ] press shift-F to fuzzy find over the entire sample library when browsing samples.
- [ ] press shift-E to export the current target collection using an export (should just replace the whole directory destructively at first).
- [ ] press shift-K to toggle showing collection tags for all samples

### further extensions
- [ ] ability to read in a session and create a collection of every sample that's referenced in the session.

## implementation
- written in golang
- bubbletea for tui functionality (import tea "github.com/charmbracelet/bubbletea")
- beep for audio playback (import "https://github.com/gopxl/beep")
- go-sqlite3 for data storage (go install github.com/mattn/go-sqlite3)

### architecture
- [x] app state should be stored in "~/.excavator-tui"
- [x] the directory should contain "~/.excavator-tui/excavator.db" which will be an sqlite database.
- [ ] the database should be loaded into memory on launch and dumped back to disk on writes (maybe periodically instead) and on exit.

### db model
- **User:** id int auto_increment, name varchar(35) unique
- **Collection:** id int auto_increment, user_id int not null, name varchar(35) not null, description
- **Tag:** id int auto_increment, file_path text unique
- **CollectionTag:** id int auto_increment, tag_id int not null, collection_id int not null, name varchar(35) not null, sub_collection varchar(250)
- **Export:** id int auto_increment, user_id int not null, name varchar(35) not null, output_dir text
- **ExportTag:** id int auto_increment, tag_id int not null, export_id int not null
