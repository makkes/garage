package features

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

const (
	SendLegacyDigestHeader = "SendLegacyDigestHeader"
)

var knownFeatures = []string{
	SendLegacyDigestHeader,
}

type Features struct {
	Flag *[]string
}

func (f *Features) BindFlags(fs *pflag.FlagSet) {
	f.Flag = fs.StringSlice("feature-gates", nil, fmt.Sprintf("A set of feature gate names to enable. Features are:\n%s", strings.Join(knownFeatures, "\n")))
}

func (f Features) Enabled(feat string) bool {
	if f.Flag == nil {
		return false
	}
	for _, flag := range *f.Flag {
		if flag == feat {
			return true
		}
	}
	return false
}
