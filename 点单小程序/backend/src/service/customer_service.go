package service

import (
	"Boutique/src/api"
	"Boutique/src/dao"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CustomerService struct{ Dao *dao.CustomerDao }

var phoneNumber = regexp.MustCompile(`^1[3-9][0-9]{9}$`)
var wechatID = regexp.MustCompile(`^[a-z][a-z0-9_-]{5,19}$`)
var customerToken = regexp.MustCompile(`^[a-f0-9]{48}$`)

func customerContact(raw string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "+86")
}

func customerNameValid(name string) bool {
	return name != "" && utf8.RuneCountInString(name) <= 30 && len(name) <= 100
}

func newCustomerSession(id string) (string, m.CustomerSession) {
	token := utils.RandomID()
	return token, m.CustomerSession{TokenHash: utils.Fingerprint(token), CustomerID: id, ExpiresAt: time.Now().Add(30 * 24 * time.Hour)}
}

func (s *CustomerService) Register(ctx context.Context, in m.CustomerCredentials) (m.CustomerAccount, string, error) {
	contact, name := customerContact(in.Contact), strings.TrimSpace(in.Name)
	if !phoneNumber.MatchString(contact) && !wechatID.MatchString(contact) {
		return m.CustomerAccount{}, "", api.Problem(400, "请填写中国大陆手机号或微信号")
	}
	if !customerNameValid(name) || len(in.Password) < 10 || len(in.Password) > 72 || strings.TrimSpace(in.Password) == "" {
		return m.CustomerAccount{}, "", api.Problem(400, "请填写称呼和 10 至 72 字节的密码")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return m.CustomerAccount{}, "", err
	}
	account := m.CustomerAccount{ID: "SU-" + strings.ToUpper(utils.RandomID()[:16]), Identifier: contact, Name: name, PasswordHash: string(hash)}
	token, session := newCustomerSession(account.ID)
	return account, token, s.Dao.Register(ctx, account, session)
}

func (s *CustomerService) Login(ctx context.Context, in m.CustomerCredentials) (m.CustomerAccount, string, error) {
	contact := customerContact(in.Contact)
	if len(in.Password) < 10 || len(in.Password) > 72 || (!phoneNumber.MatchString(contact) && !wechatID.MatchString(contact)) {
		return m.CustomerAccount{}, "", api.Problem(401, "账号或密码不正确")
	}
	account, err := s.Dao.ByContact(ctx, contact)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return account, "", err
	}
	hash := account.PasswordHash
	if hash == "" {
		hash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5WxhUqjKQ1mJ0Y1aGfFFh14Hk6Cvhva"
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil || account.ID == "" {
		return account, "", api.Problem(401, "账号或密码不正确")
	}
	token, session := newCustomerSession(account.ID)
	return account, token, s.Dao.CreateSession(ctx, session)
}

func (s *CustomerService) Account(ctx context.Context, token string) (m.CustomerAccount, error) {
	if !customerToken.MatchString(token) {
		return m.CustomerAccount{}, api.Problem(401, "请登录顾客账号")
	}
	account, err := s.Dao.SessionAccount(ctx, utils.Fingerprint(token))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return account, api.Problem(401, "顾客登录已失效，请重新登录")
	}
	return account, err
}

func (s *CustomerService) Logout(ctx context.Context, token string) error {
	return s.Dao.DeleteSession(ctx, utils.Fingerprint(token))
}

func (s *CustomerService) SaveProfile(ctx context.Context, account m.CustomerAccount, in m.CustomerProfileInput) (m.CustomerAccount, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Dorm = strings.TrimSpace(in.Dorm)
	if !customerNameValid(in.Name) || len(in.Dorm) > 300 {
		return account, api.Problem(400, "请检查称呼和寝室地址长度")
	}
	return s.Dao.UpdateProfile(ctx, account, in)
}
