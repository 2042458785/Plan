package service

import (
	"Boutique/src/config"
	"context"
	"github.com/golang-jwt/jwt/v4"
	"testing"
	"time"
)

func TestInvalidJWTsRejectedBeforeDataAccess(t *testing.T) {
	cfg := config.JWTConfig{Secret: "unit-test-secret-with-32-characters", Issuer: "unit"}
	auth := AuthService{Config: cfg}
	base := Claims{AdminID: 1, Role: "admin", RegisteredClaims: jwt.RegisteredClaims{Issuer: cfg.Issuer, Audience: jwt.ClaimStrings{"merchant"}, ID: "test-session", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	cases := []struct {
		name   string
		modify func(*Claims)
		secret string
	}{{"forged signature", func(c *Claims) {}, "other-key"}, {"wrong role", func(c *Claims) { c.Role = "customer" }, cfg.Secret}, {"expired", func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) }, cfg.Secret}, {"wrong issuer", func(c *Claims) { c.Issuer = "another-app" }, cfg.Secret}, {"wrong audience", func(c *Claims) { c.Audience = jwt.ClaimStrings{"customer"} }, cfg.Secret}}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			claims := base
			test.modify(&claims)
			raw, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(test.secret))
			if e != nil {
				t.Fatal(e)
			}
			if _, e = auth.Validate(context.Background(), raw); e == nil {
				t.Fatal("非法JWT被接受")
			}
		})
	}
}
