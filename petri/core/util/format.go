package util

import "bytes"

// CleanDockerOutput removes trailing bytes from Docker's stdout pipe
func CleanDockerOutput(stdout string) string {
	cleanedLine := bytes.TrimFunc([]byte(stdout), func(r rune) bool {
		return !(r >= 0x20 && r <= 0x7E)
	})

	cleanedLine = bytes.TrimSpace(cleanedLine)
	cleanedLine = bytes.Trim(cleanedLine, "\n.")

	return string(cleanedLine)
}
