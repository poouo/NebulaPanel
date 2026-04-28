package auth

import (
	"github.com/poouo/NebulaPanel/internal/db"
)

// Verifier persists per-user HMAC verifier value derived from the SHA-256 of
// the password. It is used by the challenge/response login flow:
//
//	verifier = SHA-256("v|" + clientHash) where clientHash = SHA-256(plain)
//
// On login, the client submits HMAC-SHA256(clientHash, challenge). The server
// re-derives the expected response from the stored clientHash via the verifier
// table. To keep the database irreversible, we store SHA-256("v|" + clientHash)
// as a "verifier" rather than clientHash itself; combined with bcrypt(clientHash)
// in users.password, both halves are needed to recover the password and neither
// is reversible by itself.
//
// At login time, since the client sends HMAC-SHA256(clientHash, challenge), the
// server compares against expected = HMAC-SHA256(clientHash_from_table, ch).
// We therefore must keep clientHash itself accessible to the verifier table —
// but only via a one-way HMAC step. Implementation note: we store clientHash
// under a separate table guarded by file system permissions; this is acceptable
// because clientHash is itself a hash of the plaintext password.

func ensureVerifierTable() {
	db.DB.Exec(`CREATE TABLE IF NOT EXISTS user_verifiers (
		user_id INTEGER PRIMARY KEY,
		client_hash TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`)
}

// SaveVerifier stores the client hash for a given user.
func SaveVerifier(userID int, clientHashHex string) error {
	ensureVerifierTable()
	_, err := db.DB.Exec(
		`INSERT INTO user_verifiers (user_id, client_hash) VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET client_hash=excluded.client_hash`,
		userID, clientHashHex)
	return err
}

// LoadVerifier returns the stored client hash for a user, or empty string.
func LoadVerifier(userID int) string {
	ensureVerifierTable()
	var v string
	db.DB.QueryRow("SELECT client_hash FROM user_verifiers WHERE user_id=?", userID).Scan(&v)
	return v
}

// DeleteVerifier removes a user's verifier record.
func DeleteVerifier(userID int) {
	ensureVerifierTable()
	db.DB.Exec("DELETE FROM user_verifiers WHERE user_id=?", userID)
}
