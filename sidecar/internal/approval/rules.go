package approval

import (
	"regexp"
	"strings"
)

// Rule flags a dangerous command. Match runs against the full command line.
type Rule struct {
	ID      string
	Reason  string
	pattern *regexp.Regexp
}

// Match reports whether the rule fires on a command line.
func (r Rule) Match(cmdline string) bool {
	return r.pattern.MatchString(cmdline)
}

// defaultRules is the built-in dangerous-command set. Kept conservative to
// avoid false positives on normal npm/git workflows (see tests).
var defaultRules = []Rule{
	{ID: "rm-rf", Reason: "recursive force delete", pattern: regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r|-rf|-fr)\b`)},
	{ID: "sudo", Reason: "privilege escalation", pattern: regexp.MustCompile(`\bsudo\b`)},
	{ID: "curl-pipe-sh", Reason: "pipe remote script to shell", pattern: regexp.MustCompile(`\b(curl|wget)\b[^|]*\|\s*(sudo\s+)?(ba)?sh\b`)},
	{ID: "dd", Reason: "raw disk write", pattern: regexp.MustCompile(`\bdd\s+.*\bof=`)},
	{ID: "mkfs", Reason: "format filesystem", pattern: regexp.MustCompile(`\bmkfs\b`)},
	{ID: "force-push", Reason: "force-push rewrites remote history", pattern: regexp.MustCompile(`\bgit\b.*\bpush\b.*(--force\b|--force-with-lease\b|\s-f\b)`)},
	{ID: "chmod-777", Reason: "world-writable permissions", pattern: regexp.MustCompile(`\bchmod\s+(-R\s+)?0?777\b`)},
}

// RuleSet evaluates command lines against the default rules plus any overrides.
type RuleSet struct {
	rules []Rule
}

func DefaultRuleSet() *RuleSet {
	return &RuleSet{rules: append([]Rule(nil), defaultRules...)}
}

// FirstMatch returns the first rule that fires on the command + args, if any.
func (rs *RuleSet) FirstMatch(command string, args []string) (Rule, bool) {
	cmdline := strings.TrimSpace(command + " " + strings.Join(args, " "))
	for _, r := range rs.rules {
		if r.Match(cmdline) {
			return r, true
		}
	}
	return Rule{}, false
}
