package auth

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/poouo/NebulaPanel/internal/db"
)

// Lockout implements a lightweight "after N consecutive failures lock this
// account/IP for M minutes" policy. State is kept in-memory so there's no
// extra schema to migrate and no database dependency for the rate limit.
//
// Configuration is persisted in the `settings` table so admins can toggle it
// from the panel. The three relevant keys are:
//
//	lockout_enabled     "true" / "false" (default "false")
//	lockout_threshold   number of consecutive failures (default 5, min 1)
//	lockout_minutes     lock duration in minutes (default 10, min 1, max 1440)
//
// The lockout is keyed by "<username>|<ip>". Username alone could lock a
// legitimate user out by someone knowing their account, and IP alone would be
// bypassable via a shared NAT; the combined key is a pragmatic middle ground.

const (
	lockoutKeyEnabled   = "lockout_enabled"
	lockoutKeyThreshold = "lockout_threshold"
	lockoutKeyMinutes   = "lockout_minutes"

	lockoutDefaultThreshold = 5
	lockoutDefaultMinutes   = 10
	lockoutMaxMinutes       = 1440 // 24 hours
)

type lockState struct {
	failures  int
	firstFail time.Time
	lockedAt  time.Time
}

var (
	lkMu   sync.Mutex
	lkMap  = map[string]*lockState{}
	lkOnce sync.Once
)

// StartLockoutCleaner removes stale entries every minute so the in-memory
// state cannot grow unbounded.
func StartLockoutCleaner() {
	lkOnce.Do(func() {
		go func() {
			for {
				time.Sleep(time.Minute)
				cutoff := time.Now().Add(-2 * time.Hour)
				lkMu.Lock()
				for k, v := range lkMap {
					// Forget very old records (no recent failure and not currently locked).
					if v.firstFail.Before(cutoff) && v.lockedAt.Before(cutoff) {
						delete(lkMap, k)
					}
				}
				lkMu.Unlock()
			}
		}()
	})
}

func lockoutConfig() (enabled bool, threshold int, minutes int) {
	threshold = lockoutDefaultThreshold
	minutes = lockoutDefaultMinutes

	var v string
	db.DB.QueryRow("SELECT value FROM settings WHERE key=?", lockoutKeyEnabled).Scan(&v)
	enabled = strings.EqualFold(strings.TrimSpace(v), "true")

	v = ""
	db.DB.QueryRow("SELECT value FROM settings WHERE key=?", lockoutKeyThreshold).Scan(&v)
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
		threshold = n
	}
	v = ""
	db.DB.QueryRow("SELECT value FROM settings WHERE key=?", lockoutKeyMinutes).Scan(&v)
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
		if n > lockoutMaxMinutes {
			n = lockoutMaxMinutes
		}
		minutes = n
	}
	return
}

func lockoutKey(username, ip string) string {
	return strings.ToLower(strings.TrimSpace(username)) + "|" + ip
}

// CheckLocked returns (true, retryAfterSeconds) if this user/ip pair is
// currently within a lockout window. If the feature is disabled, it always
// returns (false, 0).
func CheckLocked(username, ip string) (bool, int) {
	enabled, _, minutes := lockoutConfig()
	if !enabled {
		return false, 0
	}
	lkMu.Lock()
	defer lkMu.Unlock()
	st, ok := lkMap[lockoutKey(username, ip)]
	if !ok || st.lockedAt.IsZero() {
		return false, 0
	}
	window := time.Duration(minutes) * time.Minute
	remain := time.Until(st.lockedAt.Add(window))
	if remain <= 0 {
		// Lock already expired; clear state.
		delete(lkMap, lockoutKey(username, ip))
		return false, 0
	}
	return true, int(remain.Seconds())
}

// RecordLoginFailure increments the failure counter for this user/ip pair
// and locks it if the threshold is reached. Returns whether the caller is now
// locked and how long to wait.
func RecordLoginFailure(username, ip string) (locked bool, retryAfterSec int) {
	enabled, threshold, minutes := lockoutConfig()
	if !enabled {
		return false, 0
	}
	key := lockoutKey(username, ip)
	lkMu.Lock()
	defer lkMu.Unlock()
	st, ok := lkMap[key]
	if !ok {
		st = &lockState{}
		lkMap[key] = st
	}
	// Reset the counter if the previous window is very old (>1 hour).
	if !st.firstFail.IsZero() && time.Since(st.firstFail) > time.Hour {
		st.failures = 0
		st.firstFail = time.Time{}
	}
	if st.failures == 0 {
		st.firstFail = time.Now()
	}
	st.failures++
	if st.failures >= threshold {
		st.lockedAt = time.Now()
		return true, minutes * 60
	}
	return false, 0
}

// RecordLoginSuccess clears any tracked failures for the user/ip pair.
func RecordLoginSuccess(username, ip string) {
	key := lockoutKey(username, ip)
	lkMu.Lock()
	delete(lkMap, key)
	lkMu.Unlock()
}
