package deliveryprobe

import (
	"errors"
	"fmt"
)

var ErrMissing = errors.New("notification missing")

func RepositoryError(err error) error { return fmt.Errorf("load notification: %v", err) }
