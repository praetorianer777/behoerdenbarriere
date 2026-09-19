// Package seeddata carries the list of authorities to check. It is embedded so that
// the binary in the container brings its own list and no volume has to be mounted.
package seeddata

import _ "embed"

//go:embed seeds.yaml
var SeedsYAML []byte
