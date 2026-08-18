package deliveryprobe

import "errors"

func Missing(err error) bool { return errors.Is(err, ErrMissing) }
