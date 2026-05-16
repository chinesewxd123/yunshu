package logpath

import (
	"regexp"
	"strings"
)

// GlobToRegExp converts a simple glob (supports * and **) to a regex pattern string.
func GlobToRegExp(pattern string) (string, bool) {
	src := strings.TrimSpace(pattern)
	if src == "" {
		return "", false
	}
	var re strings.Builder
	re.WriteString("^")
	i := 0
	for i < len(src) {
		ch := src[i]
		if ch == '*' {
			if i+1 < len(src) && src[i+1] == '*' {
				re.WriteString(".*")
				i += 2
			} else {
				re.WriteString("[^/\\\\]*")
				i++
			}
			continue
		}
		if ch == '?' {
			re.WriteString(".")
			i++
			continue
		}
		if ch == '.' {
			re.WriteString(`\.`)
			i++
			continue
		}
		if strings.ContainsRune(`+^$(){}|[]\`, rune(ch)) {
			re.WriteString(`\`)
		}
		re.WriteByte(ch)
		i++
	}
	re.WriteString("$")
	return re.String(), true
}

// PathMatchesSource returns true when filePath matches log source path (exact or glob).
func PathMatchesSource(filePath, sourcePath string) bool {
	file := strings.TrimSpace(filePath)
	src := strings.TrimSpace(sourcePath)
	if file == "" || src == "" {
		return false
	}
	if !strings.ContainsAny(src, "*?[") {
		return file == src
	}
	reSrc, ok := GlobToRegExp(src)
	if !ok {
		return false
	}
	re, err := regexp.Compile(reSrc)
	if err != nil {
		return false
	}
	return re.MatchString(file)
}
