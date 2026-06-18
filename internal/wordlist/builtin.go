package wordlist

import (
	"bufio"
	"bytes"
	_ "embed"
	"strings"
)

//go:embed data/common.txt
var commonRaw []byte

//go:embed data/directories.txt
var directoriesRaw []byte

//go:embed data/files.txt
var filesRaw []byte

// BuiltinCommon returns the embedded common wordlist.
func BuiltinCommon() []string { return parseWordlist(commonRaw) }

// BuiltinDirectories returns the embedded directory wordlist.
func BuiltinDirectories() []string { return parseWordlist(directoriesRaw) }

// BuiltinFiles returns the embedded file wordlist.
func BuiltinFiles() []string { return parseWordlist(filesRaw) }

func parseWordlist(raw []byte) []string {
	var entries []string
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries = append(entries, line)
	}
	return entries
}
