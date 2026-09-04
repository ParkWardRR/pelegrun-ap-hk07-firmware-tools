package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// canonicalHash returns the lowercase hex SHA-256 of v's canonical JSON
// encoding. encoding/json emits struct fields in declaration order and sorts
// map keys, so the digest is deterministic for equal values — which is what
// lets a plan, policy, or identity be pinned by hash and revalidated later.
func canonicalHash(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// mustHash is canonicalHash for values that cannot fail to marshal (the fleet
// schema types). A marshal error here is a programming error, not input.
func mustHash(v any) string {
	h, err := canonicalHash(v)
	if err != nil {
		panic("fleet: canonical hash failed: " + err.Error())
	}
	return h
}
