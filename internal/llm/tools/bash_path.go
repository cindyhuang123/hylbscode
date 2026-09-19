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

// scriptInterpreters are non-bash executables that run a script file whose
// contents bash_path cannot audit. Executing a script through any of these
// requires confirmation even when the script lives inside the working
// directory, because the script body is opaque to path analysis.
var scriptInterpreters = map[string]bool{
	"python": true, "python2": true, "python3": true, "pypy": true,
	"node": true, "nodejs": true, "deno": true, "bun": true,
	"ruby": true, "perl": true, "php": true, "lua": true, "luajit": true,
	"Rscript": true, "R": true,
	"sh": true, "zsh": true, "dash": true, "ksh": true, "ash": true,
	"fish": true, "tcsh": true, "csh": true,
	"awk": true, "gawk": true, "sed": true,
	"source": true,
}

// directScriptExt lists extensions (lowercase) indicating a token is executed
// as a script file rather than referenced as data.
var directScriptExt = map[string]bool{
	".sh": true, ".py": true, ".py3": true, ".js": true, ".mjs": true,
	".pl": true, ".pm": true, ".rb": true, ".php": true, ".lua": true,
	".bash": true, ".zsh": true, ".ksh": true, ".dash": true, ".csh": true,
	".fish": true, ".r": true, ".awk": true,
}

// analyzeScriptExecution returns the script file the command executes plus
// whether confirmation is required. Any non-bash executable used to run a
// script forces confirmation, so the whitelist root is the script file itself
// (file-level, not directory-tree): confirming the script does not open its
// sibling scripts. bash is trusted: script execution through bash falls back
// to the regular command-path analysis.
func analyzeScriptExecution(cmd string) (string, bool) {
	tokens := tokenize(cmd)
	if len(tokens) == 0 {
		return "", false
	}
	base := strings.ToLower(trimTrailingPunct(tokens[0]))
	if base == "" {
		return "", false
	}
	if base == "bash" {
		return "", false
	}
	isInterp := scriptInterpreters[base]
	isDirect := stringHasPathForm(tokens[0]) && isScriptExtToken(tokens[0])
	if !isInterp && !isDirect {
		return "", false
	}

	// Direct execution ./things.sh: the base token is the script itself.
	if isDirect {
		candidate := trimTrailingPunct(tokens[0])
		if strings.HasPrefix(candidate, "~") || strings.HasPrefix(candidate, "$") {
			expanded, ok := expandHome(candidate)
			if !ok {
				return "", true
			}
			candidate = expanded
		}
		abs, ok := resolveExistingPath(candidate)
		if !ok {
			return "", true
		}
		return abs, true
	}

	// Interpreter execution: scan for the script file argument; inline -c/-e
	// code cannot be audited either, so it forces confirmation.
	nextIsRedirectTarget := false
	for _, tok := range tokens[1:] {
		if !nextIsRedirectTarget && isRedirectOp(tok) {
			nextIsRedirectTarget = true
			continue
		}
		isTarget := nextIsRedirectTarget
		nextIsRedirectTarget = false
		if isTarget {
			continue
		}
		if tok == "-c" || tok == "-e" || tok == "-m" || tok == "--eval" ||
			tok == "-p" || tok == "-i" || tok == "-r" {
			return "", true
		}
		if strings.HasPrefix(tok, "-") {
			continue
		}
		if !stringHasPathForm(tok) {
			// A bare word that's not the script (e.g. program name) — ignore.
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
		if abs, ok := resolveExistingPath(candidate); ok {
			return abs, true
		}
		return "", true
	}

	// No script argument found yet the base is a known interpreter: script is
	// opaque relative to the process (WD/server). Confirm conservatively.
	if isInterp {
		return "", true
	}
	return "", false
}

func isScriptExtToken(tok string) bool {
	tok = strings.ToLower(trimTrailingPunct(tok))
	for ext := range directScriptExt {
		if strings.HasSuffix(tok, ext) {
			return true
		}
	}
	return false
}
