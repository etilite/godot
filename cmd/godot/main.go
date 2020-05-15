package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/tetafro/godot"
)

// version is the application version. Set to latest git tag on `make build`.
var version = "dev"

const usage = `Usage:
    godot [OPTION] [FILES]
Options:
	-a, --all       check all top-level comments (not only declarations)
	-h, --help      show this message
	-v, --version   show version`

type arguments struct {
	help    bool
	version bool
	all     bool
	files   []string
}

func main() {
	args, err := readArgs()
	if err != nil {
		fatalf("Error: %v", err)
	}

	if args.help {
		fmt.Println(usage)
		os.Exit(0)
	}
	if args.version {
		fmt.Println(version)
		os.Exit(0)
	}
	if len(args.files) == 0 {
		fatalf(usage)
	}

	var settings godot.Settings
	if args.all {
		settings.CheckAll = true
	}

	var files []*ast.File
	fset := token.NewFileSet()

	for _, path := range args.files {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fatalf("Path '%s' does not exist", path)
		}
		for f := range findFiles(path) {
			file, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
			if err != nil {
				fatalf("Failed to parse file '%s': %v", path, err)
			}
			files = append(files, file)
		}
	}

	for i := range files {
		issues := godot.Run(files[i], fset, settings)
		for _, iss := range issues {
			fmt.Printf("%s: %s\n", iss.Message, iss.Pos)
		}
	}
}

func readArgs() (args arguments, err error) {
	if len(os.Args) < 2 {
		return arguments{}, fmt.Errorf("not enough arguments")
	}

	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			args.files = append(args.files, arg)
			continue
		}

		switch arg {
		case "-h", "--help":
			args.help = true
		case "-v", "--version":
			args.version = true
		case "-a", "--all":
			args.all = true
		default:
			return arguments{}, fmt.Errorf("unknown flag '%s'", arg)
		}
	}
	return args, nil
}

func findFiles(root string) chan string {
	out := make(chan string)

	go func() {
		defer close(out)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			sep := string(filepath.Separator)
			if strings.HasPrefix(path, "vendor"+sep) || strings.Contains(path, sep+"vendor"+sep) {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
				out <- path
			}
			return nil
		})
		if err != nil {
			fatalf("Failed to get files from directory: %v", err)
		}
	}()

	return out
}

func fatalf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
	os.Exit(1)
}
