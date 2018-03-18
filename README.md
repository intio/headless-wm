# dewm (forked)

`dewm` is a pure Go autotiling window manager. You may find it
somewhat similar to [dwm][] or [wmii][], but has some ideas of its
own.

This `dewm` was forked from Dave MacFarlane's [dewm][original-dewm],
which was written in [literate style][literate-programming], using
[lmt][]. The fork dropped the original Markdown sources, heavy
refactoring and cleanup was done, bugs were fixed, some features
dropped, more added, arbitrary changes made.

[original-dewm]: https://github.com/driusan/dewm
[literate-programming]: https://en.wikipedia.org/wiki/Literate_programming
[lmt]: https://github.com/driusan/lmt
[dwm]: https://dwm.suckless.org/
[wmii]: https://code.google.com/archive/p/wmii/

## Basics

`dewm` arranges the screen into columns, and divides columns up
between windows that are in that column. Windows always spawn in the
first empty column, or the end of the last column if there are no
empty columns. All columns are equally sized, and each window in any
given column is equally sized.

## Keybindings

These keybindings are currently hardcoded, but may one day be configurable.

### Window Management

* `Alt-H/Alt-L` move the current window left or right 1 column.
* `Alt-J/Alt-K` move the current window up or down 1 window in current column
* `Alt-M` switch to monocle layout (maximize all windows)
* `Alt-T` switch to tile/columns layout
* `Alt-N` create a new column 
* `Alt-D` delete any empty columns

### Other

* `Alt-Enter` spawn an xterm
* `Alt-Q` close the current window
* `Alt-Shift-Q` destroy the current window
* `Ctrl-Alt-Shift-Q` quit dewm
