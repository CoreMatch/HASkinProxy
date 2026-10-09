package main

import "embed"

// sdkSource embeds the compile-time SDK package source tree (manifest.json
// and the React page) into the binary. At startup the proxy packs it into a
// tar.gz archive and uploads it to HRPAuth, so the WebUI SDK handler can
// aggregate it into the frontend build. No sdk/ directory needs to be
// shipped beside the executable (see HA-Contract sdk-package.md).
//
//go:embed sdk
var sdkSource embed.FS
