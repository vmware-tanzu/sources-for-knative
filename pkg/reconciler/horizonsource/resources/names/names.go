package names

import (
	"knative.dev/pkg/kmeta"
)

// (@mgasch) not using source prefixes for now
// const prefix = "horizon-source"

func NewAdapterName(source string) string {
	return kmeta.ChildName(source, "-adapter")
}
