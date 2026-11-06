package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"sync"
	"time"
)

type LoginLimits struct {
	attempts *attemptWindow
	checking chan struct{}
}

func NewLoginLimits() *LoginLimits {
	return &LoginLimits{newAttemptWindow(), make(chan struct{}, 8)}
}
func (l *LoginLimits) AllowSource(peer string) bool {
	source, _, err := net.SplitHostPort(peer)
	if err != nil {
		source = peer
	}
	return l.attempts.allow("ip:"+source, 30, time.Now())
}
func (l *LoginLimits) AcquireAccount(school, email string) (release func(), ok bool) {
	key := sha256.Sum256([]byte(school + "\x00" + strings.ToLower(strings.TrimSpace(email))))
	if !l.attempts.allow("account:"+hex.EncodeToString(key[:]), 8, time.Now()) {
		return nil, false
	}
	select {
	case l.checking <- struct{}{}:
		return func() { <-l.checking }, true
	default:
		return nil, false
	}
}

type windowEntry struct {
	n       int
	expires time.Time
}
type attemptWindow struct {
	mu         sync.Mutex
	entries    map[string]windowEntry
	cleanAfter time.Time
}

func newAttemptWindow() *attemptWindow { return &attemptWindow{entries: make(map[string]windowEntry)} }

// Bound state even with attacker-selected account names or many peer addresses.
// When at capacity, refuse new keys until eligible entries expire.
func (a *attemptWindow) allow(key string, limit int, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !now.Before(a.cleanAfter) {
		for k, v := range a.entries {
			if !now.Before(v.expires) {
				delete(a.entries, k)
			}
		}
		a.cleanAfter = now.Add(5 * time.Second)
	}
	v, ok := a.entries[key]
	if !ok || !now.Before(v.expires) {
		if !ok && len(a.entries) >= 8192 {
			return false
		}
		v = windowEntry{expires: now.Add(time.Minute)}
	}
	if v.n >= limit {
		return false
	}
	v.n++
	a.entries[key] = v
	return true
}
