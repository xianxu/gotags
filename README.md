This is a program to generate ctags for go. I'm using this with vim and ctrl-p.
As a result, it's not particularly fully implemented, just the portion of stuff I need.

The basic functionality provided is to construct a tags file under current running directory.
Several src directory can be passed in, all in the relative form of current directory. All go
source files in those directories will be recursively parsed and root level declarations
extracted. Currently private declarations are extracted as well, for easier reading of source
code.

So to use, do:
	gotags dir1 dir2 file1

For golang dev though, I think the golang plugin for IntelliJ's very good. With IdeaVim plugin,
you don't really need this vim+ctags setting. My 2c.

