package auth

// HashClientHash converts the client-side SHA-256 hex of the user's password
// into the value stored in the database (bcrypt(SHA-256(plain))). This way:
//   - Plaintext password never leaves the browser.
//   - Database holds only an irreversible bcrypt hash.
//   - Both registration and password change use the same hashing chain.
func HashClientHash(clientHashHex string) (string, error) {
	return HashPassword(clientHashHex)
}

// CheckClientHash verifies a SHA-256(plain) hex against the stored bcrypt hash.
// For backward compatibility, callers may also fall back to CheckPassword on
// the original plaintext (used only by legacy clients during one transition).
func CheckClientHash(clientHashHex, storedHash string) bool {
	return CheckPassword(clientHashHex, storedHash)
}
