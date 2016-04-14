package units

import "github.com/kopeio/kope/pkg/fi"

type HasAddress interface {
	// Finds the IP address managed by this unit
	// if not found (but not an error), returns nil, nil
	FindAddress(c fi.Cloud) (*string, error)
}
