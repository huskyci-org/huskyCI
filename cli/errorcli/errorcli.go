package errorcli

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrInvalidExtension occurs when an extension is a image and video one
	ErrInvalidExtension = errors.New("invalid extension")
)

// Handle prints the error message in the cli format
func Handle(errorFound error) {
	fmt.Fprintf(os.Stderr, "\n[HUSKYCI] ❌ Error: %s\n\n", errorFound)
	fmt.Fprintf(os.Stderr, "Tip: Use 'huskyci --help' for more information about available commands.\n")
	fmt.Fprintf(os.Stderr, "For troubleshooting, visit: https://github.com/huskyci-org/huskyCI/wiki\n\n")
	os.Exit(1)
}
