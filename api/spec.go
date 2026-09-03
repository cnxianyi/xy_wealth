// Package api contains the embedded OpenAPI contract and documentation UI.
package api

import (
	"bytes"
	_ "embed"
)

//go:embed openapi.yaml
var specification []byte

//go:embed docs.html
var documentation []byte

// Specification returns a copy of the OpenAPI document.
func Specification() []byte {
	return bytes.Clone(specification)
}

// Documentation returns a copy of the interactive documentation page.
func Documentation() []byte {
	return bytes.Clone(documentation)
}
