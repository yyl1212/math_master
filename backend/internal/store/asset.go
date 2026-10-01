package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
)

var assetDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// GetPublishedAsset applies the same effective publication and immutable unit
// binding rules as knowledge reads; an administrative export is not sufficient.
func (s *Store) GetPublishedAsset(ctx context.Context, digest string) (data []byte, err error) {
	if !assetDigestPattern.MatchString(digest) {
		return nil, ErrNotFound
	}
	err = s.withPublication(ctx, func(st *publicationState) error {
		allowed := false
		for id := range st.knowledge {
			view, ok := st.knowledgeView(id)
			if !ok {
				continue
			}
			for _, u := range view.Units {
				for _, assetID := range u.AssetIDs {
					a, ok := st.assets[assetID]
					if ok && a.SHA256 == digest {
						allowed = true
						break
					}
				}
				if allowed {
					break
				}
			}
			if allowed {
				break
			}
		}
		if !allowed {
			return ErrNotFound
		}
		e := st.tx.QueryRowContext(st.ctx, "SELECT bytes FROM assets WHERE sha256=$1 AND octet_length(bytes)<=1048576", digest).Scan(&data)
		if errors.Is(e, sql.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			return errors.New("stored asset integrity failure")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}
