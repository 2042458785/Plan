package http_controller

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/service"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

type CustomerController struct {
	api.BaseApi
	Service *service.CustomerService
}

func NewCustomerController(s *service.CustomerService) *CustomerController {
	return &CustomerController{api.NewBaseApi(), s}
}

func CustomerTransport(ctx *gin.Context) {
	peer, _, _ := net.SplitHostPort(ctx.Request.RemoteAddr)
	host := ctx.Request.Host
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	loopback := net.ParseIP(peer).IsLoopback()
	host = strings.Trim(host, "[]")
	localHost := host == "localhost" || net.ParseIP(host).IsLoopback()
	forwarded := ctx.GetHeader("X-Forwarded-Proto")
	if ctx.Request.TLS == nil && !(loopback && (forwarded == "https" || (forwarded == "" && localHost))) {
		api.Fail(ctx, 403, "请通过 HTTPS 使用顾客账号")
		return
	}
	ctx.Next()
}

func customerBearer(ctx *gin.Context) string {
	raw, _ := strings.CutPrefix(ctx.GetHeader("Authorization"), "Bearer ")
	return raw
}

func (c *CustomerController) RequireAccount(ctx *gin.Context) {
	account, err := c.Service.Account(ctx.Request.Context(), customerBearer(ctx))
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	ctx.Set("customerAccount", account)
	ctx.Next()
}

func currentCustomer(ctx *gin.Context) (m.CustomerAccount, bool) {
	value, ok := ctx.Get("customerAccount")
	if !ok {
		return m.CustomerAccount{}, false
	}
	return value.(m.CustomerAccount), true
}

func (c *CustomerController) Credentials(register bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input m.CustomerCredentials
		if c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &input}).GetError() != nil {
			return
		}
		authenticate := c.Service.Login
		if register {
			authenticate = c.Service.Register
		}
		account, token, err := authenticate(ctx.Request.Context(), input)
		if err != nil {
			api.RespondError(ctx, err)
			return
		}
		api.Ok(ctx, gin.H{"account": account, "token": token})
	}
}

func (c *CustomerController) Profile(ctx *gin.Context) {
	account, _ := currentCustomer(ctx)
	api.Ok(ctx, account)
}

func (c *CustomerController) SaveProfile(ctx *gin.Context) {
	account, _ := currentCustomer(ctx)
	var input m.CustomerProfileInput
	if c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &input}).GetError() != nil {
		return
	}
	account, err := c.Service.SaveProfile(ctx.Request.Context(), account, input)
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, account)
}

func (c *CustomerController) Logout(ctx *gin.Context) {
	if err := c.Service.Logout(ctx.Request.Context(), customerBearer(ctx)); err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, nil)
}
