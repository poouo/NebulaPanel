package captcha

import (
	"sync"
	"time"
)

const (
	// 密码错误多少次后要求验证码
	FailThreshold = 2
	// 失败记录保留时间
	FailWindow = 30 * time.Minute
)

type failRecord struct {
	Count     int
	FirstFail time.Time
}

var (
	failStore = make(map[string]*failRecord)
	failMu    sync.RWMutex
)

// RecordFail 记录一次登录失败
func RecordFail(ip string) {
	failMu.Lock()
	defer failMu.Unlock()

	rec, ok := failStore[ip]
	if !ok || time.Since(rec.FirstFail) > FailWindow {
		failStore[ip] = &failRecord{Count: 1, FirstFail: time.Now()}
		return
	}
	rec.Count++
}

// NeedCaptcha 检查该 IP 是否需要验证码
func NeedCaptcha(ip string) bool {
	failMu.RLock()
	defer failMu.RUnlock()

	rec, ok := failStore[ip]
	if !ok {
		return false
	}
	if time.Since(rec.FirstFail) > FailWindow {
		return false
	}
	return rec.Count >= FailThreshold
}

// ClearFail 登录成功后清除失败记录
func ClearFail(ip string) {
	failMu.Lock()
	defer failMu.Unlock()
	delete(failStore, ip)
}

// StartFailCleaner 定期清理过期的失败记录
func StartFailCleaner() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			failMu.Lock()
			now := time.Now()
			for k, v := range failStore {
				if now.Sub(v.FirstFail) > FailWindow {
					delete(failStore, k)
				}
			}
			failMu.Unlock()
		}
	}()
}
