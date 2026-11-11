package service

import (
	"Boutique/src/api"
	"Boutique/src/config"
	"Boutique/src/dao"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

type Claims struct {
	AdminID uint   `json:"adminId"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}
type AuthService struct {
	Dao       *dao.AuthDao
	Config    config.JWTConfig
	Redis     *redis.Client
	Namespace string
}

func (s *AuthService) key(id string) string { return "suxu:" + s.Namespace + ":admin-session:" + id }
func (s *AuthService) Login(ctx context.Context, in m.LoginInput, ip string) (map[string]any, error) {
	if len(in.Username) > 60 || len(in.Password) > 200 {
		return nil, api.Problem(400, "账号或密码格式不正确")
	}
	user, e := s.Dao.Authenticate(ctx, in, ip)
	if e != nil {
		return nil, e
	}

	id := utils.RandomID()
	expires := time.Now().Add(time.Duration(s.Config.Expire) * time.Hour)
	claims := Claims{AdminID: user.ID, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{Issuer: s.Config.Issuer, Subject: strconv.Itoa(int(user.ID)), Audience: jwt.ClaimStrings{"merchant"}, ExpiresAt: jwt.NewNumericDate(expires), IssuedAt: jwt.NewNumericDate(time.Now()), ID: id}}
	token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Config.Secret))
	if e != nil {
		return nil, e
	}
	if e = s.Dao.SaveSession(ctx, m.AdminSession{ID: id, AdminID: user.ID, ExpiresAt: expires}); e != nil {
		return nil, e
	}
	if s.Redis != nil {
		if e = s.Redis.Set(ctx, s.key(id), utils.Fingerprint(token), time.Until(expires)).Err(); e != nil {
			_ = s.Dao.Revoke(ctx, id)
			return nil, e
		}
	}
	return map[string]any{"token": token, "expiresAt": expires, "user": map[string]any{"id": user.ID, "username": user.Username, "role": user.Role}}, nil
}
func (s *AuthService) Validate(ctx context.Context, raw string) (*Claims, error) {
	claims := &Claims{}
	token, e := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("无效签名算法")
		}
		return []byte(s.Config.Secret), nil
	})
	if e != nil || !token.Valid || claims.AdminID == 0 || claims.Role != "admin" || claims.ID == "" || claims.Issuer != s.Config.Issuer || !claims.VerifyAudience("merchant", true) {
		return nil, api.Problem(401, "登录凭证无效或已过期")
	}
	ok, e := s.Dao.ValidSession(ctx, claims.ID, claims.AdminID)
	if e != nil {
		return nil, e
	}
	if !ok {
		return nil, api.Problem(401, "登录已失效，请重新登录")
	}
	if s.Redis != nil {
		stored, e := s.Redis.Get(ctx, s.key(claims.ID)).Result()
		if e != nil || stored != utils.Fingerprint(raw) {
			return nil, api.Problem(401, "登录会话已失效")
		}
	}
	return claims, nil
}
func (s *AuthService) Logout(ctx context.Context, id string) error {
	if e := s.Dao.Revoke(ctx, id); e != nil {
		return e
	}
	if s.Redis != nil {
		return s.Redis.Del(ctx, s.key(id)).Err()
	}
	return nil
}
