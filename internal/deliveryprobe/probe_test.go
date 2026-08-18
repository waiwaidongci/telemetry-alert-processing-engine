package deliveryprobe

import "testing"

func TestMissingNotificationIsNotRetried(t *testing.T) {
	err := RepositoryError(ErrMissing)
	s := Status(err)
	if s != 404 || Attempts(s) != 1 {
		t.Fatalf("status=%d attempts=%d", s, Attempts(s))
	}
}
