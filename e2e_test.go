package algeneva

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizationAllStrategies(t *testing.T) {
	for country, strategy := range Strategies {
		for i, s := range strategy {
			results, pass, err := TestStrategyNormalization(s)
			if !assert.NoError(t, err, "%s[%d]: failed", country, i) || pass {
				continue
			}

			for _, r := range results {
				if !r.Pass {
					assert.Fail(t, fmt.Sprintf("%s[%d]: %s: %s", country, i, r.Name, r.Msg))
				}
			}
		}
	}
}
