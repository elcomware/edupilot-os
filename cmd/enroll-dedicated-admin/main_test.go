package main

import (
	"errors"
	"strings"
	"testing"

	auth "github.com/elcomware/edupilot/modules/auth/app/services"
)

func TestSafeMessageRedactsRawErrors(t *testing.T) {
	secret := "synthetic-private-credential"
	if strings.Contains(safeMessage(errors.New("database failed password="+secret)), secret) {
		t.Fatal("raw database error exposed")
	}
	if got := safeMessage(auth.ErrConflict); got != auth.ErrConflict.Error() {
		t.Fatal("known error changed", got)
	}
}
