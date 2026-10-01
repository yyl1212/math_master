package auth

// DecodeContentProof only decodes proof material; identity always comes from the database.
func DecodeContentProof(cookies Cookies, csrf string, write bool) (SessionProof, error) {
	var p SessionProof
	if cookies.Session == "" {
		return p, ErrAuthenticationRequired
	}
	token, err := DecodeSecret(cookies.Session)
	if err != nil {
		return p, ErrInvalidCookie
	}
	p.TokenHash = TokenDigest(token)
	if write {
		p.CSRF, err = DecodeSecret(csrf)
		if err != nil {
			return SessionProof{}, ErrCSRF
		}
	}
	return p, nil
}
