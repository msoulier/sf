package main

// Adapted from my original sane-filenames.py script.
// Rules
// All lower-case
// One or more spaces to a single underscore
// hyphens to underscores
// ampersand to "and"
// shell metachars to underscores
// when all done, translate multiple, adjacent underscores to a single one
import (
	"path/filepath"
	"io"
	"os"
	"strings"
	"regexp"
	"github.com/op/go-logging"
	"flag"
	"bufio"
	"fmt"
)

var (
	log *logging.Logger = nil
	debug = false
	confirm = false
	directories = false
	string_input = false
	args []string
	errors []string
)

func init() {
	errors = make([]string, 0)
	flag.BoolVar(&debug, "d", false, "Debug logging")
	flag.BoolVar(&confirm, "c", false, "Confirm all moves")
	flag.BoolVar(&directories, "D", false, "Rename directories too")
	flag.BoolVar(&string_input, "s", false, "Apply clean algorithm to stdin")
	flag.Parse()
	args = flag.Args()

	format := logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05.000-0700} %{level} [%{shortfile}] %{message}`,
	)
	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, format)
	stderrBackendLevelled := logging.AddModuleLevel(stderrFormatter)
	logging.SetBackend(stderrBackendLevelled)
	if debug {
			stderrBackendLevelled.SetLevel(logging.DEBUG, "sf")
	} else {
			stderrBackendLevelled.SetLevel(logging.INFO, "sf")
	}
	log = logging.MustGetLogger("sf")

	if len(args) < 1 && ! string_input {
		log.Errorf("Usage: %s <path/files>", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func clean_name(path string, simple bool) (string, bool) {
	log.Debugf("clean_name: path is %s", path)
	changed := false
	var base, dir string
	if simple {
		base = path
		dir = ""
	} else {
		base = filepath.Base(path)
		dir = filepath.Dir(path)
	}
	log.Debugf("basename is %s", base)
	origname := base

	// All lower-case.
	base = strings.ToLower(base)
	// One or more spaces to a single underscore.
	base = strings.ReplaceAll(base, " ", "_")
	// hyphens to underscores
	base = strings.ReplaceAll(base, "-", "_")
	// ampersand to "and"
	base = strings.ReplaceAll(base, "&", "and")
	// shell metachars to underscores
	shell_metachar_pat := regexp.MustCompile("[^_a-z0-9.]")
	base = shell_metachar_pat.ReplaceAllLiteralString(base, "_")
	// when all done, translate multiple, adjacent underscores to a single one
	mult_underscores_pat := regexp.MustCompile("_+")
	base = mult_underscores_pat.ReplaceAllLiteralString(base, "_")
	// Remove any leading or trailing underscores.
	if base[0] == '_' {
		base = base[1:]
	}
	if len(base) > 1 {
		if base[len(base)-1] == '_' {
			base = base[:len(base)-1]
		}
	}

	if strings.Compare(origname, base) == 0 {
		changed = false
	} else {
		changed = true
	}

	log.Debugf("base is now '%s', changed is %v", base, changed)
	if simple {
		return base, changed
	} else {
		return filepath.Join(dir, base), changed
	}
}

func prompt_confirmation(prompt string) (bool, error) {
	stdin_reader := bufio.NewReader(os.Stdin)
	stdout_writer := bufio.NewWriter(os.Stdout)
	_, err := stdout_writer.WriteString(prompt)
	if err != nil {
		log.Errorf("%s", err)
		return false, err
	}
	stdout_writer.Flush()
	line, err := stdin_reader.ReadString('\n')
	if err != nil {
		log.Errorf("%s", err)
		return false, err
	}
	// Default to false - make this a param
	if len(line) == 0 {
		return false, nil
	}
	if line[0] != 'y' && line[0] != 'Y' {
		return false, nil
	} else {
		return true, nil
	}
}

func walk(path string) {
	outputDirRead, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	outputDirFiles, err := outputDirRead.Readdir(0)
	for i, f := range outputDirFiles {
		log.Debugf("dir %s: %d %s\n", path, i, f.Name())
		subpath := filepath.Join(path, f.Name())
		// Visit the path. For directories this will recursively
		// end up in walk again.
		visit(subpath)
	}
}

func visit(path string) {
	finfo, err := os.Lstat(path)
	if err != nil {
		panic(err)
	}
	if finfo.IsDir() {
		// DFS search
		walk(path)
		// Do not continue unless the user wants to rename directories too.
		if ! directories {
			log.Debugf("skipping directory %s, directory option is false", path)
		}
	}
	// At this point we're back from all subdirectories.
	newname, changed := clean_name(path, false)
	if changed {
		rename := false
		if confirm {
			msg := fmt.Sprintf("Rename\n\t%s\n\tto\n\t%s? [y/N] ", path, newname)
			rename, err = prompt_confirmation(msg)
			if err != nil {
				panic(err)
			}
		} else {
			rename = true
		}
		if rename {
			log.Infof("renaming %s to %s", path, newname)
			err := os.Rename(path, newname)
			if err != nil {
				errmsg := fmt.Sprintf("failed to rename %s to %s: %s", path, newname, err)
				log.Errorf(errmsg)
				errors = append(errors, errmsg)
			}
		}
	}
}

func main() {
	stdin_reader := bufio.NewReader(os.Stdin)

	if string_input {
		// Read from stdin, clean and print to stdout.
		for {
			line, err := stdin_reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					os.Exit(0)
				} else {
					panic(err)
				}
			}
			newline, _ := clean_name(line, true)
			fmt.Printf("%s\n", newline)
		}
	}

	for _, path := range args {
		log.Debugf("%s", path)
		visit(path)
	}

	if len(errors) > 0 {
		fmt.Printf("There were errors:\n")
		for _, err := range errors {
			fmt.Printf("   ==> %s\n", err)
		}
	}
}
