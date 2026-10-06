package httpapi

import "io"

func decodeStudyJSON(reader io.Reader, limit int64, dst any) error {
	return decodeContentJSON(reader, limit, dst)
}
