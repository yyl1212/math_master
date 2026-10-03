package httpapi

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"io"
)

type NotificationReadInput struct{}

func DecodeNotificationInput(raw []byte) (NotificationReadInput, error) {
	var in NotificationReadInput
	e := correctionDecode(raw, &in)
	return in, e
}
func readNotificationInput(r io.Reader) (NotificationReadInput, error) {
	raw, e := io.ReadAll(io.LimitReader(r, correction.MaxRequestBytes+1))
	if e != nil {
		return NotificationReadInput{}, auth.ErrInvalidInput
	}
	return DecodeNotificationInput(raw)
}
