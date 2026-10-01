package content

import (
	"context"
	"path/filepath"
	"strings"
)

type AssetReader func(context.Context, Asset) ([]byte, error)

func ValidAssetPath(p string) bool {
	if p == "" || filepath.IsAbs(p) || strings.ContainsAny(p, "\\\x00") || !strings.HasSuffix(p, ".svg") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}
func FileAssetReader(root string) AssetReader {
	return func(ctx context.Context, a Asset) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b, err := readAsset(root, a)
		if err != nil {
			return nil, err
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		return b, nil
	}
}
func ValidateSVG(b []byte) error {
	if len(b) > 1<<20 {
		return ErrValidation
	}
	return validateSVG(b)
}
