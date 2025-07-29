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
	args []string
)

func init() {
	flag.BoolVar(&debug, "d", false, "Debug logging")
	flag.BoolVar(&confirm, "c", false, "Confirm all moves")
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

	if len(args) < 1 {
		log.Errorf("Usage: %s <path/files>", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func clean_name(path string) (string, bool) {
	log.Debugf("clean_name: path is %s", path)
	changed := false
	base := filepath.Base(path)
	dir := filepath.Dir(path)
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

	if strings.Compare(origname, base) == 0 {
		changed = false
	} else {
		changed = true
	}

	return filepath.Join(dir, base), changed
}

func main() {
	stdin_reader := bufio.NewReader(os.Stdin)
	stdout_writer := bufio.NewWriter(os.Stdout)

	for _, path := range args {
		log.Debugf("%s", path)
		filepath.Walk(path, func(fpath string, info fs.FileInfo, err error) error {
			if err != nil {
				log.Errorf("Walk error: %s", err)
				return fs.SkipDir
			}
			log.Debugf("fpath is %s", fpath)
			if info.IsDir() {
				log.Debugf("%s is a directory", fpath)
				// FIXME switches to control directory renaming behaviour
			} else {
				newname, changed := clean_name(fpath)
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
	}
}
