package format // import "gotest.tools/internal/format"

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Message accepts a msgAndArgs varargs and formats it using fmt.Sprintf
func Message(msgAndArgs ...interface{}) string {
	switch len(msgAndArgs) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%v", msgAndArgs[0])
	default:
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
}

// WithCustomMessage accepts one or two messages and formats them appropriately
func WithCustomMessage(source string, msgAndArgs ...interface{}) string {
	custom := Message(msgAndArgs...)
	switch {
	case custom == "":
		return source
	case source == "":
		return custom
	}
	return fmt.Sprintf("%s: %s", source, custom)
}

// CleanTestNameForFilesystem converts path separators to dashes so that
// t.Name() can be used as a filename.
func CleanTestNameForFilesystem(name string) string {
	// windows requires both / and \ are replaced
	if runtime.GOOS == "windows" {
		name = strings.Replace(name, string(os.PathSeparator), "-", -1)
	}
	return strings.Replace(name, "/", "-", -1)
}
