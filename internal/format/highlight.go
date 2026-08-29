package format

import "regexp"

// badPattern marks log lines that often point at the failure.
var badPattern = regexp.MustCompile(`(?i)\b(error|panic|fatal|exception|oom|oomkilled|timeout|timed out|deadline exceeded|refused|denied|crash)\b`)

const ansiRed = "\x1b[31m"

// IsBad reports whether a log line looks like a failure.
func IsBad(text string) bool {
	return badPattern.MatchString(text)
}
