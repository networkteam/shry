package template

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// VariablePattern matches {{variableName}} in text, where variableName can only contain
	// alphanumeric characters, hyphens, and underscores. Whitespace around the variable name
	// is allowed but not newlines.
	VariablePattern = regexp.MustCompile(`{{[ \t]*([a-zA-Z0-9_-]+)[ \t]*}}`)
)

// Resolve resolves variables in the given text using the provided variables map
func Resolve(text string, variables map[string]any) (string, error) {
	// Find all variables in the text
	matches := VariablePattern.FindAllStringSubmatch(text, -1)
	if matches == nil {
		return text, nil
	}

	// Replace each variable
	result := text
	for _, match := range matches {
		placeholder := match[0] // e.g. {{variableName}}
		varName := match[1]     // e.g. variableName
		varValue, exists := variables[varName]
		if !exists {
			return "", fmt.Errorf("variable %s not defined", varName)
		}
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(varValue))
	}

	return result, nil
}

// FindVariables returns all variable names found in the given text
func FindVariables(text string) []string {
	matches := VariablePattern.FindAllStringSubmatch(text, -1)
	if matches == nil {
		return nil
	}

	// Extract unique variable names
	seen := make(map[string]bool)
	var variables []string
	for _, match := range matches {
		varName := match[1]
		if !seen[varName] {
			seen[varName] = true
			variables = append(variables, varName)
		}
	}

	return variables
}
