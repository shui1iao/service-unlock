package render

import _ "embed"

// ThirdPartyLicenses keeps the renderer's notices in standalone binaries.
//
//go:embed THIRD_PARTY_LICENSES.txt
var ThirdPartyLicenses string
