This program generates ctags' tags file, for the intention to be used in vim and ctrl-p. To use,
simply pass in a list of directories and go source files. Directories are processed recursively.

Examples:

  gotags .

  gotags src ../go/src/pkg

Tag generate contain some additional information. E.g. os.func.Exit would be one tag. Typically
I use gotags src/ ../go/src/pkg to generate tags that contains both src files of my project and
source files of sdk.
