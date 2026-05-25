package activity

import "regexp"

// Secret patterns scrubbed from activity summaries/details before they are
// persisted or sent to the UI. Conservative: better to over-redact a log line
// than leak a credential into a shareable replay export.
var secretPatterns = []*regexp.Regexp{
	// key=value / key: value style secrets.
	regexp.MustCompile(`(?i)(token|secret|password|passwd|api[_-]?key|access[_-]?key|client[_-]?secret)(["'\s:=]+)([^\s"']{6,})`),
	// AWS access key ids.
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	// Bearer tokens in Authorization headers.
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]{12,}`),
	// GitHub-style tokens.
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),
	// JWTs.
	regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}`),
}

const redactMask = "‹redacted›"

// Redact replaces detected secrets in s with a mask. Key=value matches keep the
// key (so context survives) and mask only the value.
func Redact(s string) string {
	out := s
	// First pattern preserves the key group; the rest mask wholesale.
	out = secretPatterns[0].ReplaceAllString(out, "$1$2"+redactMask)
	for _, re := range secretPatterns[1:] {
		out = re.ReplaceAllString(out, redactMask)
	}
	return out
}
