package workspace

import (
	"context"
	"errors"
	"testing"
)

func TestRejectsUnscopedOrInvalidEntriesBeforePersistence(t *testing.T) {
	svc := Service{}
	for _, tc := range []struct{ school, title string }{{"", "Follow up"}, {"school-a", "   "}} {
		_, err := svc.Create(context.Background(), tc.school, tc.title)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("school=%q title=%q: got %v", tc.school, tc.title, err)
		}
	}
}
