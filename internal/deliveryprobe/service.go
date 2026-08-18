package deliveryprobe

import (
	"errors"
	"fmt"
)

func Missing(err error) bool { return errors.Is(err, ErrMissing) }
func PublicError(err error) error {
	if Missing(err) {
		return fmt.Errorf("missing delivery: %w", err)
	}
	return err
}
