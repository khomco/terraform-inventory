package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var version = flag.Bool("version", false, "print version information and exit")
var list = flag.Bool("list", false, "list mode")
var host = flag.String("host", "", "host mode")
var inventory = flag.Bool("inventory", false, "inventory mode")

func main() {
	flag.Parse()
	file := flag.Arg(0)
	var files []string

	if *version == true {
		fmt.Printf("%s version %s\n", os.Args[0], versionInfo())
		return
	}

	// not given on the command line? try ENV.
	if file == "" {
		file = os.Getenv("TF_STATE")
	}

	// also try the old ENV name.
	if file == "" {
		file = os.Getenv("TI_TFSTATE")
	}

	// check for a file named terraform.tfstate in the pwd
	if file == "" {
		files = findFiles()
	} else {
		files = append(files, file)
	}

	if file == "" && len(files) == 0 {
		fmt.Printf("Usage: %s [options] path\n", os.Args[0])
		os.Exit(1)
	}

	if !*list && *host == "" && !*inventory {
		fmt.Fprint(os.Stderr, "Either --host or --list must be specified")
		os.Exit(1)
	}

	var states []*state
	for _, tfFile := range files {
		path, err := filepath.Abs(tfFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid file: %s\n", err)
			os.Exit(1)
		}

		stateFile, err := os.Open(path)
		defer stateFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening tfstate file: %s\n", err)
			os.Exit(1)
		}

		var s state
		err = s.read(stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading tfstate file: %s\n", err)
			os.Exit(1)
		}
		states = append(states, &s)
	}

	if *list {
		os.Exit(cmdList(os.Stdout, os.Stderr, states))

	} else if *inventory {
		os.Exit(cmdInventory(os.Stdout, os.Stderr, states))

	} else if *host != "" {
		os.Exit(cmdHost(os.Stdout, os.Stderr, states, *host))
	}
}

func findFiles() []string {
		var tfstateFiles []string
		dirname := "." + string(filepath.Separator)

    d, err := os.Open(dirname)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    defer d.Close()

    files, err := d.Readdir(-1)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    for _, file := range files {
        if file.Mode().IsRegular() {
            if filepath.Ext(file.Name()) == ".tfstate" {
							tfstateFiles = append(tfstateFiles, file.Name())
            }
        }
    }
		return tfstateFiles
}
