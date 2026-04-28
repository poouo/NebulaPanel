package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// challengeStore is an in-memory map of issued login challenges.
// Each challenge is consumed on first verification or expires automatically.
const challengeTTL = 2 * time.Minute

type challenge struct {
	expires time.Time
}

var (
	chMu  sync.Mutex
	chMap = make(map[string]challenge)
)

// IssueChallenge generates a new random hex challenge string for the client.
func IssueChallenge() string {
	cleanupChallenges()
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	c := hex.EncodeToString(b)
	chMu.Lock()
	chMap[c] = challenge{expires: time.Now().Add(challengeTTL)}
	chMu.Unlock()
	return c
}

// ConsumeChallenge returns true and removes the challenge if it exists and is valid.
func ConsumeChallenge(c string) bool {
	if c == "" {
		return false
	}
	chMu.Lock()
	defer chMu.Unlock()
	v, ok := chMap[c]
	if !ok {
		return false
	}
	delete(chMap, c)
	return time.Now().Before(v.expires)
}

func cleanupChallenges() {
	chMu.Lock()
	defer chMu.Unlock()
	now := time.Now()
	for k, v := range chMap {
		if now.After(v.expires) {
			delete(chMap, k)
		}
	}
}

// HMACSHA256Hex returns hex-encoded HMAC-SHA256(key, data).
func HMACSHA256Hex(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// SHA256Hex returns hex-encoded SHA-256 digest of input.
func SHA256Hex(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// VerifyChallengeResponse validates an HMAC-SHA256(passhash, challenge) response.
//
// passhash is what the client sees as its "stable secret": the SHA-256 of the
// user's plaintext password. The server stores bcrypt(passhash). To validate
// without ever seeing the plaintext or passhash, the server keeps a parallel
// HMAC verifier (hex(HMAC-SHA256(passhash, "verifier"))) which is compared
// indirectly via the client-supplied response. Implementations using bcrypt
// only must therefore accept either:
//   1) response == HMAC-SHA256(SHA256(plain), challenge) — preferred
//   2) sha256(plain) compared against bcrypt(stored) — backward compatibility
// CheckChallengeResponse returns true if the supplied response matches.
func VerifyHMAC(passhashHex, challenge, response string) bool {
	expect := HMACSHA256Hex(passhashHex, challenge)
	return strings.EqualFold(expect, response)
}
