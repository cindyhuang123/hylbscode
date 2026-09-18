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

// expandHome resolves ~, ~/x, $HOME, ${HOME} and $HOME/x tokens to absolute
// paths. Tokens it cannot resolve report ok=false.
func expandHome(token string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	switch {
	case token == "~" || token == "$HOME" || token == "${HOME}":
		return home, true
	case strings.HasPrefix(token, "~/"):
		return filepath.Join(home, token[len("~/"):]), true
	case strings.HasPrefix(token, "$HOME/"):
		return filepath.Join(home, token[len("$HOME/"):]), true
	case strings.HasPrefix(token, "${HOME}/"):
		return filepath.Join(home, token[len("${HOME}/"):]), true
	}
	return "", false
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

func isRedirectOp(token string) bool {
	switch token {
	case ">", ">>", "<", "2>", "2>>", "&>", "&>>":
		return true
	}
	return false
}

func stringHasPathForm(token string) bool {
	if token == "" || strings.HasPrefix(token, "-") {
		return false
	}
	if strings.Contains(token, "=") {
		return false
	}
	token = trimTrailingPunct(token)
	if token == "" || token == "." || token == ".." {
		return false
	}
	if strings.HasPrefix(token, "~") {
		return true
	}
	if strings.HasPrefix(token, "$") {
		return strings.Contains(token, "/") || token == "$HOME" || token == "${HOME}"
	}
	return strings.Contains(token, "/") || strings.HasPrefix(token, ".")
}

// analyzeCommandPaths returns the first path referenced by the command that
// exists outside the working directory and /tmp, plus whether the command
// requires confirmation. Unresolvable variable paths force confirmation.
func analyzeCommandPaths(cmd string) (string, bool) {
	tokens := tokenize(cmd)
	nextIsRedirectTarget := false
	for _, tok := range tokens {
		if !nextIsRedirectTarget && isRedirectOp(tok) {
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
		if strings.HasPrefix(candidate, "~") || strings.HasPrefix(candidate, "$") {
			expanded, ok := expandHome(candidate)
			if !ok {
				return "", true
			}
			candidate = expanded
		}
		abs, ok := resolveExistingPath(candidate)
		if !ok {
			continue
		}
		if !inWorkingDir(abs) {
			return abs, true
		}
	}
	return "", false
}
