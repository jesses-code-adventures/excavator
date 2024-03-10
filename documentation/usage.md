# **usage**

launch the program by calling excavator.

## first launch

on first launch you are prompted to set up your username and a root sample directory. after that, you will be prompted to create a collection.

a collection is a group of subcollections and tags. a subcollection does what it says on the tin. a tag belongs to a collection or a subcollection and is a link to a file on your hard drive.

once you've entered a name and a description for your first collection, you're ready to use excavator.

## general use

- navigate around your samples directory tagging samples you'd like to add to your collection.
- optionally include subcollections with your tags, which will be exported as subdirectories.
- create concrete exports, which fully copy the files in your collection to the export location (good for creating sample packs).
- create abstract exports, which create symlinks in the export location referencing the source files (good for organising your samples for a daw).
- collections and exports live in an sqlite database on your harddrive.
- at any point you can use run any export on any collection.

## cli flags

- **--data** _string allowing you to modify the location of your sqlite database and logfile. defaults to "~/.local/state/excavator-tui"._
- **--db** _string allowing you to set the filename of the sqlite .db file. defaults to "excavator"._
- **--logfile** _string allowing you to enter the name of the logfile (defaults to "logfile")._
- **--root** _string allowing you to launch with a temporary root samples directory, lasting until the session is closed._
- **--user** _creates a new user whose name is the argument. if the user exists, you launch as that user._
- **--watch** _can be used in a separate terminal window to watch live log outputs as the program runs._

## controls

- **q** _quit if you're in the home window, else go to the home window._
- **j** _move up_
- **k** _move down_
- **<ctrl>-d** _jump up_
- **<ctrl>-u** _jump down_
- **gg** _jump to top_
- **G** _jump to bottom_
- **r** _audition random sample._
- **c** _change the target collection._
- **C** _create a new collection._
- **t** _quick tag (use target collection & subcollection)._
- **T** _tag (enter alternative collection & subcollection)._
- **a** _audition selected sample._
- **A** _toggle auto-audition mode._
- **e** _run an export._
- **E** _create an export._
- **d** _clear target subdirectory._
- **D** _change target directory._
- **f** _recursively search filenames from current directory._
- **F** _recursively search filenames from the root directory._
- **b** _browse the target collection_
- **K** _toggle showing collection tags for all samples_
- **/** _search the current buffer and move the cursor to the next match_
- **n** _move to the next search result after executing a search_
- **p** _move to the previous search result after executing a search_
