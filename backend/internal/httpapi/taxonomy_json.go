package httpapi

import "io"

// 复用既有严格词法检查：重复/大小写键、未知字段、缺项、NUL、UTF-8和深度。
func decodeTaxonomyJSON(reader io.Reader, dst any) error { return decodeContentJSON(reader, 8192, dst) }
