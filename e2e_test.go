package algeneva

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizationAllStrategies(t *testing.T) {
	for country, strategy := range Strategies {
		for i, s := range strategy {
			results, pass, err := TestStrategyNormalization(s)
			if !assert.NoError(t, err, "%s[%d]: ", country, i) || pass {
				continue
			}

			var msg []string
			for _, r := range results {
				if !r.Pass {
					msg = append(msg, fmt.Sprintf("%s: %s", r.Name, r.Msg))
				}
			}

			// Even though we know the test failed by this point, we still use assert for
			// formatting
			assert.True(t, pass, "%s[%d] failed:\n\t%s", country, i, strings.Join(msg, "\n\t"))
		}
	}
}
