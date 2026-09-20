package auth

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrTraversalAttack = errors.New("directory traversal attack detected")

func validateArgument(mask, input string) error {
	cleanedInput := filepath.Clean(input)
	if cleanedInput != input && strings.Contains(input, "..") {
		return ErrTraversalAttack
	}
	// match logic...
	return nil
}