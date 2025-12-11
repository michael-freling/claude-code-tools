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
func isProtectedBranch(branch string) bool {
	branch = strings.TrimSpace(branch)
	return branch == "main" || branch == "master"
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
