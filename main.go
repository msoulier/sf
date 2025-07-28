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
	"fmt"
	"path/filepath"
	"io/fs"
	"os"
	"strings"
	"regexp"
)

func clean_name(path string) string {
	fmt.Printf("clean_name: path is %s\n", path)
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	fmt.Printf("basename is %s\n", base)

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

	return filepath.Join(dir, base)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path/files>\n", os.Args[0])
		os.Exit(1)
	}
	for _, path := range os.Args[1:] {
		fmt.Printf("%s\n", path)
		filepath.Walk(path, func(fpath string, info fs.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Walk error: %s\n", err)
				return fs.SkipDir
			}
			fmt.Printf("fpath is %s\n", fpath)
			if info.IsDir() {
				fmt.Printf("%s is a directory\n", fpath)
				// FIXME switches to control directory renaming behaviour
			} else {
				newname := clean_name(fpath)
				fmt.Printf("newname is %s\n", newname)
				err := os.Rename(fpath, newname)
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR in rename from %s to %s: %s", fpath, newname, err)
					// For now, continue processing
				}
			}
			return nil
		})
	}
}
