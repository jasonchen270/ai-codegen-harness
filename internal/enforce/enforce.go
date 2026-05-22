// Package enforce judges a run's changeset against the allow-list.
//
// One rule, default-deny: the whole project is frozen except the paths the run
// explicitly allows. Any change (create, modify, or delete) to a path outside
// the allow-list is a violation. Enforced by the tool after the agent runs, not
// trusted to the model.
package enforce

import (
	"fmt"
	"path"
	"strings"

	"github.com/jasonchen270/ai-codegen-harness/internal/git"
)

// isAllowed reports whether rel falls under any allowed path. An allow entry
// ending in "/" is a directory prefix; otherwise it is an exact file match.
func isAllowed(rel string, allow []string) bool {
	rel = path.Clean(rel)
	for _, entry := range allow {
		if strings.HasSuffix(entry, "/") {
			dir := strings.TrimSuffix(entry, "/")
			if rel == dir || strings.HasPrefix(rel, dir+"/") {
				return true
			}
		} else if rel == path.Clean(entry) {
			return true
		}
	}
	return false
}

// Violation is a single change that touched the frozen region.
type Violation struct {
	Path   string
	Detail string
}

// Evaluate splits a changeset into the in-bounds paths and the violations.
func Evaluate(changes []git.Change, allow []string) (allowed []string, violations []Violation) {
	for _, ch := range changes {
		rel := ch.Path
		if isAllowed(rel, allow) {
			allowed = append(allowed, rel)
			continue
		}
		verb := "created"
		switch {
		case ch.IsDelete():
			verb = "deleted"
		case ch.IsModify():
			verb = "modified"
		}
		violations = append(violations, Violation{
			Path:   rel,
			Detail: fmt.Sprintf("%s a file outside the allowed paths %v", verb, allow),
		})
	}
	return allowed, violations
}
