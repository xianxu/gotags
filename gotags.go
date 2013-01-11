/*
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
 */
package main

import (
	"fmt"
    "flag"
	"os"
	"bufio"
	"strings"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"bytes"
	"path/filepath"
)

const (
	sep = "."
	header = `!_TAG_FILE_FORMAT	2	/extended format; --format=1 will not append ;" to lines/
!_TAG_FILE_SORTED	0	/0=unsorted, 1=sorted, 2=foldcase/
!_TAG_PROGRAM_AUTHOR	Xian Xu	/email/
!_TAG_PROGRAM_NAME	gotags	//
!_TAG_PROGRAM_URL	http://github.com/xianxu/gotags	/official site/
!_TAG_PROGRAM_VERSION	1.0	//
`
)

func main() {
    flag.Parse()
	paths := flag.Args()

	// a channel of tags discovered.
	tags := make(chan string, 10)
	done := make(chan int)
	// start simple consumer of tag stream
	go consume(tags, done)

	// parse files and send tags to a channel
	for _, p := range paths {
		if info, err := os.Lstat(p); err == nil {
			if info.IsDir() {
				filepath.Walk(p, func(path string, info os.FileInfo, err error)error {
						if info.IsDir() {
							fset := token.NewFileSet()
							if trees, err := parser.ParseDir(fset, path, func(fi os.FileInfo) bool {
									return strings.HasSuffix(fi.Name(), ".go")
								}, 0); err == nil {
								for i, v := range trees {
									for _, file := range v.Files {
										parseGo(i + sep, fset, file, tags)
									}
								}
							}
						}
						return nil
					})
			} else {
				fset := token.NewFileSet()
				if tree, err := parser.ParseFile(fset, p, nil, 0); err == nil {
					parseGo(tree.Name.Name + sep, fset, tree, tags)
				}
			}
		}
	}
	close(tags)
	<-done
}

func consume(tags chan string, done chan int) {
	tf, err := os.Create("tags")
	if err != nil {
		fmt.Printf("%v", err)
		panic("Can't open tags file for write.")
	}
	writer := bufio.NewWriter(tf)
	writer.WriteString(header)
	for tag := range tags {
		writer.WriteString(tag + "\n")
	}
	writer.Flush()
	tf.Close()

	done <- 1
}

func isExported(s string)bool {
	return s[0] >= 'A' && s[0] <= 'Z'
}

// convert an ast node to a string. Only use this for simple stuff
func nodeToString(fset *token.FileSet, n interface{}) string {
	if n == nil {
		return ""
	}
	var buf bytes.Buffer
	err := printer.Fprint(&buf, fset, n)
	if err != nil {
		fmt.Println("Error : " + err.Error())
	}
	return buf.String()
}
// get a better function signature
func funcToString(fset *token.FileSet, n interface{}) string {
	if n == nil {
		return ""
	}
	f := n.(*ast.FuncDecl)
	var buf bytes.Buffer
	if f.Recv == nil {
		buf.WriteString("func" + sep)
	} else {
		buf.WriteString("method" + sep + fieldsToString(fset, f.Recv) + sep)
	}
	buf.WriteString(f.Name.Name)
	buf.WriteString("("+fieldsToString(fset, f.Type.Params)+")")
	buf.WriteString("("+fieldsToString(fset, f.Type.Results)+")")
	return buf.String()
}
func fieldsToString(fset *token.FileSet, fl *ast.FieldList) string {
	if fl == nil || fl.List == nil {
		return ""
	}
	var buf bytes.Buffer
	for _, f := range fl.List {
		buf.WriteString(nodeToString(fset, f.Type) + ",")
	}
	r := buf.String()
	if len(r) <= 0 {
		return r
	}
	return r[0:len(r)-1]
}

// convert an ast node to a string representing the location of that string in source file.
// This is strictly used for ctags, so the format's basically
// filename<tab>line_no;
func nodeToLoc(fset *token.FileSet, n interface{}) string {
	start := fset.Position(n.(ast.Node).Pos())
	return fmt.Sprintf("%v\t%v;", start.Filename, start.Line)
}
// parse go source file and output ctags format.
// ctags format is documented here: http://ctags.sourceforge.net/FORMAT
// I'm using a variety that's supported by ctrl-p of vim. Notibly the tag itself may contain
// whitespace.
func parseGo(prefix string, fset *token.FileSet, tree *ast.File, tags chan string) {
	for _, node := range tree.Decls {
		switch n := node.(type) {
		case *ast.FuncDecl:
			if !isExported(n.Name.Name) {
				continue
			}
			tags <- prefix + funcToString(fset, n) + "\t" + nodeToLoc(fset, n)
		case *ast.GenDecl:
			// n.Tok for type
			for _, v := range n.Specs {
				switch m := v.(type) {
				case *ast.TypeSpec:
					if !isExported(m.Name.Name) {
						continue
					}
					if _, ok := m.Type.(*ast.StructType); ok {
						tags <- prefix + "struct" + sep + m.Name.Name + "\t" + nodeToLoc(fset, m)
					} else if _, ok := m.Type.(*ast.InterfaceType); ok {
						tags <- prefix + "interface" + sep + m.Name.Name + "\t" + nodeToLoc(fset, m)
					} else if id, ok := m.Type.(*ast.Ident); ok {
						tags <- prefix + "struct" + sep + m.Name.Name + " " + id.Name + "\t" + nodeToLoc(fset, m)
					}
				case *ast.ValueSpec:
					for _, v := range m.Names {
						if !isExported(v.Name) {
							continue
						}
						ts := nodeToString(fset, m.Type)
						if len(ts) > 0 {
							ts = " " + ts
						}
						if n.Tok == token.CONST {
							tags <- prefix + "const" + sep + v.Name + ts + "\t" + nodeToLoc(fset, v)
						} else if n.Tok == token.VAR {
							tags <- prefix + "var" + sep + v.Name + ts + "\t" + nodeToLoc(fset, v)
						}
					}
				}
			}
		}
	}
}

