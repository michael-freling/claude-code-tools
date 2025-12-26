// Package hooks provides command parsing and validation utilities.
// This file contains shared helpers for parsing bash commands and extracting arguments.
package hooks

import (
	"strings"
)

// isGhApiCommand checks if the command starts with "gh api".
func isGhApiCommand(command string) bool {
	tokens := strings.Fields(command)
	if len(tokens) < 2 {
		return false
	}
	return tokens[0] == "gh" && tokens[1] == "api"
}

// extractHTTPMethod extracts the HTTP method from a gh api command.
// Returns empty string if no method is specified (defaults to GET).
func extractHTTPMethod(command string) string {
	tokens := strings.Fields(command)

	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "-X" || tokens[i] == "--method" {
			if i+1 < len(tokens) {
				return strings.ToUpper(tokens[i+1])
			}
		}
	}

	return ""
}

// isProtectedBranch checks if a branch name is main or master.
// Also handles full ref paths like refs/heads/main or origin/main.
func isProtectedBranch(branch string) bool {
	branch = strings.TrimSpace(branch)
	// Check exact match
	if branch == "main" || branch == "master" {
		return true
	}
	// Check full ref path (refs/heads/main, refs/heads/master, origin/main, etc.)
	if strings.HasSuffix(branch, "/main") || strings.HasSuffix(branch, "/master") {
		return true
	}
	return false
}

// parseCommandTokens parses a command string into tokens, respecting quoted strings.
// Quotes are included in the returned tokens to preserve the original token structure.
func parseCommandTokens(command string) []string {
	return parseTokens(command, true)
}

// parseTokensStripQuotes parses a command string into tokens, stripping quotes.
// Single and double quotes are removed from the returned tokens.
func parseTokensStripQuotes(command string) []string {
	return parseTokens(command, false)
}

// parseTokens parses a command string into tokens, respecting quoted strings.
// If keepQuotes is true, quotes are included in tokens; otherwise they are stripped.
func parseTokens(command string, keepQuotes bool) []string {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(command); i++ {
		ch := command[i]

		switch ch {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
				if keepQuotes {
					current.WriteByte(ch)
				}
			} else {
				current.WriteByte(ch)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
				if keepQuotes {
					current.WriteByte(ch)
				}
			} else {
				current.WriteByte(ch)
			}
		case ' ', '\t', '\n', '\r':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// findNonFlagArgs filters out flags and their values from an argument list.
// It returns only the non-flag arguments starting from startIndex.
// flagsWithValues is a list of flags that take a value (e.g., "--repo", "--exec").
func findNonFlagArgs(args []string, startIndex int, flagsWithValues []string) []string {
	var nonFlagArgs []string
	skipNext := false

	for i := startIndex; i < len(args); i++ {
		arg := args[i]

		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(arg, "-") {
			for _, flag := range flagsWithValues {
				if arg == flag {
					skipNext = true
					break
				}
			}
			continue
		}

		nonFlagArgs = append(nonFlagArgs, arg)
	}

	return nonFlagArgs
}

// extractTargetFromRefspec extracts the target branch from a refspec.
// Handles formats: "+src:dst" -> "dst", "+branch" -> "branch", ":branch" -> "branch", "src:dst" -> "dst"
func extractTargetFromRefspec(refspec string) string {
	// Strip force push prefix
	refspec = strings.TrimPrefix(refspec, "+")

	// Check for colon separator (src:dst format)
	if idx := strings.Index(refspec, ":"); idx >= 0 {
		return refspec[idx+1:]
	}

	// No colon, the refspec is the branch name itself
	return refspec
}

// containsPushAllFlag checks if args contain --all or --mirror flags
func containsPushAllFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--all" || arg == "--mirror" {
			return true
		}
	}
	return false
}

// containsDeleteFlag checks if args contain --delete or -d flag
func containsDeleteFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--delete" || arg == "-d" {
			return true
		}
	}
	return false
}

// isDeleteRefspec checks if the refspec is a delete operation (starts with ":")
func isDeleteRefspec(refspec string) bool {
	return strings.HasPrefix(refspec, ":")
}

// isForcePushRefspec checks if the refspec is a force push (starts with "+")
func isForcePushRefspec(refspec string) bool {
	return strings.HasPrefix(refspec, "+")
}

// splitShellCommands splits a command string on shell operators and returns individual sub-commands.
// Handles: &&, ||, ;, |, &
// Also strips subshell parentheses from sub-commands.
// Respects quoted strings - operators inside quotes are not treated as separators.
func splitShellCommands(command string) []string {
	var commands []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	i := 0

	for i < len(command) {
		ch := command[i]

		// Handle quotes
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			i++
			continue
		}
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			i++
			continue
		}

		// If inside quotes, just add the character
		if inSingleQuote || inDoubleQuote {
			current.WriteByte(ch)
			i++
			continue
		}

		// Check for redirection patterns like >&1, >&2, 2>&1, 1>&2 - don't split on & in these
		if ch == '>' && i+2 < len(command) && command[i+1] == '&' {
			// This is a redirection like >&1 or >&2
			current.WriteByte(ch)
			current.WriteByte(command[i+1])
			current.WriteByte(command[i+2])
			i += 3
			continue
		}
		if ch >= '0' && ch <= '9' && i+3 < len(command) && command[i+1] == '>' && command[i+2] == '&' {
			// This is a redirection like 2>&1
			current.WriteByte(ch)
			current.WriteByte(command[i+1])
			current.WriteByte(command[i+2])
			current.WriteByte(command[i+3])
			i += 4
			continue
		}

		// Check for two-character operators (&&, ||)
		if i+1 < len(command) {
			twoChar := command[i : i+2]
			if twoChar == "&&" || twoChar == "||" {
				if current.Len() > 0 {
					commands = append(commands, cleanSubCommand(current.String()))
					current.Reset()
				}
				i += 2
				continue
			}
		}

		// Check for single-character operators (;, |, &)
		if ch == ';' || ch == '|' || ch == '&' {
			if current.Len() > 0 {
				commands = append(commands, cleanSubCommand(current.String()))
				current.Reset()
			}
			i++
			continue
		}

		current.WriteByte(ch)
		i++
	}

	// Add the last command
	if current.Len() > 0 {
		commands = append(commands, cleanSubCommand(current.String()))
	}

	// Filter out empty commands
	var result []string
	for _, cmd := range commands {
		if cmd != "" {
			result = append(result, cmd)
		}
	}

	return result
}

// cleanSubCommand cleans up a sub-command by trimming whitespace and removing subshell parentheses.
// Also handles redirections by removing them from the command.
func cleanSubCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)

	// Strip leading and trailing parentheses (subshells)
	for strings.HasPrefix(cmd, "(") && strings.HasSuffix(cmd, ")") {
		cmd = strings.TrimPrefix(cmd, "(")
		cmd = strings.TrimSuffix(cmd, ")")
		cmd = strings.TrimSpace(cmd)
	}

	// Strip just leading ( for cases like "(git push ...)" without matching closing
	cmd = strings.TrimPrefix(cmd, "(")
	cmd = strings.TrimSpace(cmd)

	// Strip just trailing ) for cases where we might have unmatched parens
	cmd = strings.TrimSuffix(cmd, ")")
	cmd = strings.TrimSpace(cmd)

	// Remove common redirections (2>&1, >/dev/null, etc.)
	// This is a simple approach - we just remove common redirection patterns
	cmd = removeRedirections(cmd)

	return cmd
}

// removeRedirections removes common redirection patterns from a command.
func removeRedirections(cmd string) string {
	var result strings.Builder
	skipNext := false

	tokens := parseTokensStripQuotes(cmd)
	for _, t := range tokens {
		// Skip redirection targets (the file after > or 2>)
		if skipNext {
			skipNext = false
			continue
		}

		// Skip redirection operators and their targets
		if isRedirectionOperator(t) {
			skipNext = true
			continue
		}

		// Skip standalone redirection patterns like 2>&1
		if isRedirectionPattern(t) {
			continue
		}

		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(t)
	}

	return result.String()
}

// isRedirectionOperator checks if a token is a redirection operator.
func isRedirectionOperator(token string) bool {
	redirectOps := []string{">", ">>", "<", "<<", "2>", "2>>", "1>", "1>>", "&>", "&>>"}
	for _, op := range redirectOps {
		if token == op {
			return true
		}
	}
	return false
}

// isRedirectionPattern checks if a token is a redirection pattern like 2>&1 or >/dev/null.
func isRedirectionPattern(token string) bool {
	patterns := []string{"2>&1", "1>&2", ">&2", ">&1"}
	for _, p := range patterns {
		if token == p {
			return true
		}
	}
	// Check for patterns like 2>/dev/null, >/dev/null
	if strings.HasPrefix(token, ">") || strings.HasPrefix(token, "2>") || strings.HasPrefix(token, "1>") || strings.HasPrefix(token, "&>") {
		return true
	}
	return false
}
