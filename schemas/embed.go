package schemas

import "embed"

// Files embeds the canonical contracts; deployment does not need source paths.
//go:embed *.schema.json msc2020-classification-codes.json
var Files embed.FS
