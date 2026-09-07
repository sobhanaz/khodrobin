package tokens

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("یک رمز فارسی درست")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("یک رمز فارسی درست", hash)
	if err != nil || !ok {
		t.Fatalf("correct password rejected: ok=%v err=%v", ok, err)
	}
	ok, _ = VerifyPassword("رمز اشتباه", hash)
	if ok {
		t.Error("wrong password accepted")
	}
}

func TestEachHashHasItsOwnSalt(t *testing.T) {
	// Identical passwords must not produce identical hashes, or a dump reveals
	// which accounts share one.
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Error("two hashes of the same password are identical — salt is not random")
	}
}

func TestHashCarriesItsParameters(t *testing.T) {
	// Self-describing so cost can be raised later without locking anyone out.
	hash, _ := HashPassword("x")
	if !strings.HasPrefix(hash, "$argon2id$") || !strings.Contains(hash, "m=65536,t=1,p=4") {
		t.Errorf("hash does not carry its parameters: %s", hash)
	}
}

func TestMalformedHashIsRejectedNotPanicked(t *testing.T) {
	for _, bad := range []string{"", "plaintext", "$argon2id$broken", "$bcrypt$v=1$m=1,t=1,p=1$a$b"} {
		if _, err := VerifyPassword("x", bad); err == nil {
			t.Errorf("malformed hash %q was accepted", bad)
		}
	}
}

func TestSecretsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		s, err := Secret()
		if err != nil {
			t.Fatal(err)
		}
		if seen[s] {
			t.Fatal("Secret() repeated a value")
		}
		seen[s] = true
	}
}

func TestFingerprintIsStableAndNotReversible(t *testing.T) {
	s, _ := Secret()
	if Fingerprint(s) != Fingerprint(s) {
		t.Error("fingerprint is not stable")
	}
	if strings.Contains(Fingerprint(s), s) {
		t.Error("fingerprint contains the secret")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	iss := NewIssuer("test-secret-value", "khodrobin")
	now := time.Now()
	raw, err := iss.Access(42, "a@b.co", true, false, now)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := iss.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.Email != "a@b.co" || !claims.Verified || claims.IsAdmin {
		t.Errorf("claims round-tripped wrong: %+v", claims)
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	iss := NewIssuer("test-secret-value", "khodrobin")
	raw, _ := iss.Access(1, "a@b.co", true, false, time.Now().Add(-2*AccessTTL))
	if _, err := iss.Parse(raw); err == nil {
		t.Error("expired token accepted")
	}
}

func TestTokenSignedWithAnotherKeyIsRejected(t *testing.T) {
	raw, _ := NewIssuer("attacker-key", "khodrobin").Access(1, "a@b.co", true, true, time.Now())
	if _, err := NewIssuer("real-key", "khodrobin").Parse(raw); err == nil {
		t.Error("token signed with a different key was accepted")
	}
}

func TestAlgNoneForgeryIsRejected(t *testing.T) {
	// The classic JWT forgery: claim alg=none and drop the signature. Rejected
	// because Parse pins the accepted algorithms.
	forged := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
		"eyJ1aWQiOjEsImVtYWlsIjoiYUBiLmNvIiwiYWRtaW4iOnRydWUsImlzcyI6Imtob2Ryb2JpbiJ9."
	if _, err := NewIssuer("real-key", "khodrobin").Parse(forged); err == nil {
		t.Error("alg=none token was accepted")
	}
}

func TestTokenFromAnotherIssuerIsRejected(t *testing.T) {
	raw, _ := NewIssuer("shared-key", "someone-else").Access(1, "a@b.co", true, true, time.Now())
	if _, err := NewIssuer("shared-key", "khodrobin").Parse(raw); err == nil {
		t.Error("token from another issuer was accepted")
	}
}
