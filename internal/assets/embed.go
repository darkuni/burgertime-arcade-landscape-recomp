//go:build embedrom

package assets

import _ "embed"

//go:embed btimem.zip
var embeddedZip []byte

func init() { Embedded = embeddedZip }
