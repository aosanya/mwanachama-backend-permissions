package mwanachamapermissions

import "regexp"

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

var actionPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

var scopePattern = regexp.MustCompile(`^\*$|^[a-z][a-z0-9_]*:[A-Za-z0-9_.:/-]+$`)

func isCode(s string) bool { return codePattern.MatchString(s) }

func isAction(s string) bool { return actionPattern.MatchString(s) }

func isScope(s string) bool { return scopePattern.MatchString(s) }

var patterns = map[string]func(string) bool{
	"code":   isCode,
	"action": isAction,
	"scope":  isScope,
}
