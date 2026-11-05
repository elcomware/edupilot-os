package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/contracts/schedule"
)

// ExpireSessions removes expired credentials from the control database.
// Multiple scheduled invocations or redeliveries are safe.
type ExpireSessions struct{}

func (ExpireSessions) Signature() string { return "edupilot:expire-sessions" }
func (ExpireSessions) Handle(...any) error {
	pools, err := database.OpenConfigured()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = pools.Control.DB.ExecContext(ctx, pools.Control.Bind("DELETE FROM identity_sessions WHERE expires_at<=?"), time.Now().Unix())
	return err
}
func Jobs() []queue.Job { return []queue.Job{ExpireSessions{}} }
func Schedule() []schedule.Event {
	return []schedule.Event{facades.Schedule().Call(func() {
		if err := facades.Queue().Job(ExpireSessions{}).Dispatch(); err != nil {
			slog.Error("schedule session cleanup", "error", err)
		}
	}).DailyAt("02:00")}
}
