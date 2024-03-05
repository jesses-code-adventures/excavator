# **excavator**
![Tests](https://github.com/jesses-code-adventures/excavator/actions/workflows/formatting.yml/badge.svg)

a sample-browsing tui. for music producers with large sample libraries who are comfortable in the terminal.

## **why**
most sample managers (i'm looking at you elektron) are painful to work with.

this serves as an intermediate program to manage your samples, allowing you to re-export your own sample packs on demand.

it can also export your sample packs as symlinks, so you can create new directories containing samples in your library without wasting disk space.

## **usage**
launch the program by calling excavator.

### first launch
on first launch you are prompted to set up your username and a root sample directory. after that, you will be prompted to create a collection.

a collection is a group of subcollections and tags. a subcollection does what it says on the tin. a tag belongs to a collection or a subcollection and is a link to a file on your hard drive.

once you've entered a name and a description for your first collection, you're ready to use excavator.

### general use
- navigate around your samples directory tagging samples you'd like to add to your collection.
- optionally include subcollections with your tags, which will be exported as subdirectories.
- create concrete exports, which fully copy the files in your collection to the export location (good for creating sample packs).
- create abstract exports, which create symlinks in the export location referencing the source files (good for organising your samples for a daw).
- collections and exports live in an sqlite database on your harddrive.
- at any point you can use run any export on any collection.

### cli flags
- **--data**    *string allowing you to modify the location of your sqlite database and logfile. defaults to "~/.excavator-tui".*
- **--db**      *string allowing you to set the filename of the sqlite .db file. defaults to "excavator".*
- **--logfile** *string allowing you to enter the name of the logfile (defaults to "logfile").*
- **--root**    *string allowing you to launch with a temporary root samples directory, lasting until the session is closed.*
- **--user**    *creates a new user whose name is the argument. if the user exists, you launch as that user.*
- **--watch**   *can be used in a separate terminal window to watch live log outputs as the program runs.*

### controls
- **q**         *quit if you're in the home window, else go to the home window.*
- **j**         *move up*
- **k**         *move down*
- **<ctrl>-d**  *jump up*
- **<ctrl>-u**  *jump down*
- **gg**        *jump to top*
- **G**         *jump to bottom*
- **r**         *audition random sample.*
- **c**         *change the target collection.*
- **C**         *create a new collection.*
- **t**         *quick tag (use target collection & subcollection).*
- **T**         *tag (enter alternative collection & subcollection).*
- **a**         *audition selected sample.*
- **A**         *toggle auto-audition mode.*
- **e**         *run an export.*
- **E**         *create an export.*
- **d**         *clear target subdirectory.*
- **D**         *change target directory.*
- **f**         *recursively search filenames from current directory.*
- **F**         *recursively search filenames from the root directory.*
- **b**         *browse the target collection*
- **K**         *toggle showing collection tags for all samples*
- **/**         *search the current buffer and move the cursor to the next match*
- **n**         *move to the next search result after executing a search*
- **p**         *move to the previous search result after executing a search*

## **goals**
minimum for v1.0:
- some way to create a collection out of samples used in a particular ableton session. (should be implmented allowing extensibility to other daws).
- ability to rename and move files in the app and keep local db in sync.
- ability to tag entire directories.
- improved performance when fuzzy finds return thousands of results.
