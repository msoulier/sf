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
	"io/fs"
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
)

func init() {
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
	if base[len(base)-1] == '_' {
		base = base[:len(base)-1]
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

func main() {
	stdin_reader := bufio.NewReader(os.Stdin)
	stdout_writer := bufio.NewWriter(os.Stdout)

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
			stdout_writer.WriteString(newline + "\n")
			stdout_writer.Flush()
		}
	}

	for _, path := range args {
		log.Debugf("%s", path)
		rename_directories := make([]string, 0)
		filepath.Walk(path, func(fpath string, info fs.FileInfo, err error) error {
			if err != nil {
				log.Errorf("Walk error: %s", err)
				return fs.SkipDir
			}
			log.Debugf("fpath is %s", fpath)
			if info.IsDir() {
				log.Debugf("%s is a directory", fpath)
				if directories {
					_, changed := clean_name(fpath, false)
					if changed {
						rename_directories = append(rename_directories, fpath)
					}
				}
			} else {
				newname, changed := clean_name(fpath, false)
				log.Debugf("newname is %s", newname)
				if changed {
					log.Debugf("===> filename has changed, need to rename")
					if confirm {
						_, err := stdout_writer.WriteString(fmt.Sprintf("Plan to rename %s to %s. Ok? [y/N] ", fpath, newname))
						if err != nil {
							log.Errorf("%s", err)
							os.Exit(1)
						}
						stdout_writer.Flush()
						line, err := stdin_reader.ReadString('\n')
						if err != nil {
							log.Errorf("%s", err)
							os.Exit(1)
						}
						if line[0] != 'y' && line[0] != 'Y' {
							log.Debug("skipping based on user response")
							return fs.SkipDir
						}
					}
					err := os.Rename(fpath, newname)
					if err != nil {
						log.Errorf("ERROR in rename from %s to %s: %s", fpath, newname, err)
						// For now, continue processing
					}
				} else {
					log.Debugf("===> no change")
				}
			}
			return nil
		})
		if len(rename_directories) > 0 {
			log.Debugf("There are directories to rename")
			for _, dir := range rename_directories {
				newname, changed := clean_name(dir, false)
				if changed {
					log.Debugf("should change %s to %s", dir, newname)
				} else {
					panic("we should not be here")
				}
			}
		}
	}
}
