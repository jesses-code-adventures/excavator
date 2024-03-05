# **excavator**
![Tests](https://github.com/jesses-code-adventures/excavator/actions/workflows/formatting.yml/badge.svg)

a sample-browsing tui. for music producers with large sample libraries who are comfortable in the terminal.

## **why**
most sample managers (i'm looking at you elektron) are painful to work with.

this serves as an intermediate program to manage your samples, allowing you to re-export your own sample packs on demand.

it can also export your sample packs as symlinks, so you can create new directories containing samples in your library without wasting disk space.

## **usage**
please see [usage.md](https://github.com/jesses-code-adventures/documentation/usage.md) for instructions.

## **goals**
minimum for v1.0:
- some way to create a collection out of samples used in a particular ableton session. (should be implmented allowing extensibility to other daws).
- ability to rename and move files in the app and keep local db in sync.
- ability to tag entire directories.
- improved performance when fuzzy finds return thousands of results.
