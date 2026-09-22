package mask

import (
	"regexp"
	"strings"
)

var bearerPattern = regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-_\.]+`)

type Masker struct {
	secrets []string
}

func New(secrets []string) *Masker {
	filtered := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if strings.TrimSpace(s) != "" {
			filtered = append(filtered, s)
		}
	}
	return &Masker{secrets: filtered}
}

func (m *Masker) Mask(input string) string {
	output := input
	for _, secret := range m.secrets {
		output = strings.ReplaceAll(output, secret, "***")
	}
	output = bearerPattern.ReplaceAllString(output, "Bearer ***")
	return output
}
