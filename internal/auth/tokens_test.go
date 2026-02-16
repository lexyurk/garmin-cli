package auth

import (
	"testing"
	"time"
)

func TestOAuth2Token_Expired_WithSkew(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	tok := OAuth2Token{ExpiresAt: now.Add(20 * time.Second).Unix()}
	if tok.Expired(now) != true {
		t.Fatalf("expected token to be expired due to skew")
	}

	tok2 := OAuth2Token{ExpiresAt: now.Add(2 * time.Minute).Unix()}
	if tok2.Expired(now) != false {
		t.Fatalf("expected token to be not expired")
	}
}
