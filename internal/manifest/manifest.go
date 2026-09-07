package manifest

import (
	"strings"

	"github.com/ju4n97/hclapi/internal/problem"
)

// ProblemConfig holds global RFC 9457 Problem Details configuration.
type ProblemConfig struct {
	TypePrefix string
}

// ProblemType returns the error URI using TypePrefix or the default URN prefix.
func (p ProblemConfig) ProblemType(slug string) string {
	if p.TypePrefix != "" {
		prefix := p.TypePrefix
		if strings.HasPrefix(prefix, "http://") || strings.HasPrefix(prefix, "https://") {
			return strings.TrimSuffix(prefix, "/") + "/" + slug
		}
		return prefix + slug
	}
	return problem.TypeURI(slug)
}
