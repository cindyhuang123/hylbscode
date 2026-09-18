package tools

import (
	"os"
	"path/filepath"
	"strings"
)

// tokenize splits a shell command into words, honoring single and double
// quotes. Heredocs and complex expansions are not interpreted; the parser is
// deliberately conservative (extra confirmations are safer than missed ones).
func tokenize(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case (c == ' ' || c == '\t' || c == '\n') && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return tokens
}

// resolveExistingPath returns the nearest existing ancestor of path, absolute
// and cleaned. It returns ("", false) when no ancestor exists at all.
func resolveExistingPath(p string) (string, bool) {
	abs := p
	if !filepath.IsAbs(abs) {
		wd := workingDirectory()
		abs = filepath.Join(wd, abs)
	}
	abs = filepath.Clean(abs)
	for {
		if _, err := os.Stat(abs); err == nil {
			return abs, true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", false
		}
		abs = parent
	}
}

// trimTrailingPunct strips shell punctuation that may trail a path token.
func trimTrailingPunct(s string) string {
	return strings.TrimRight(s, ",;|&()")
}

func stringHasPathForm(token string) bool {
	if token == "" || strings.HasPrefix(token, "-") || strings.HasPrefix(token, "$") {
		return false
	}
	if strings.Contains(token, "=") {
		return false
	}
	token = trimTrailingPunct(token)
	if token == "" || token == "." || token == ".." || token == "~" {
		return false
	}
	return strings.Contains(token, "/") || strings.HasPrefix(token, ".")
}

// externalPathInCommand returns the first path referenced by the command that
// exists outside the working directory and /tmp, if any.
func externalPathInCommand(cmd string) string {
	tokens := tokenize(cmd)
	nextIsRedirectTarget := false
	for _, tok := range tokens {
		if !nextIsRedirectTarget && (tok == ">" || tok == ">>" || tok == "<" || tok == "2>" || tok == "&>") {
			nextIsRedirectTarget = true
			continue
		}
		isTarget := nextIsRedirectTarget
		nextIsRedirectTarget = false
		if !isTarget && !stringHasPathForm(tok) {
			continue
		}
		candidate := trimTrailingPunct(tok)
		if candidate == "" {
			continue
		}
		abs, ok := resolveExistingPath(candidate)
		if !ok {
			continue
		}
		if !inWorkingDir(abs) {
			return abs
		}
	}
	return ""
}
