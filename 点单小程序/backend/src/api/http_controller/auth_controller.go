package http_controller

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/service"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	api.BaseApi
	Service *service.AuthService
}

func NewAuthController(s *service.AuthService) *AuthController {
	return &AuthController{api.NewBaseApi(), s}
}
func (c *AuthController) Login(ctx *gin.Context) {
	var in m.LoginInput
	if e := c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &in}).GetError(); e != nil {
		return
	}
	out, e := c.Service.Login(ctx.Request.Context(), in, ctx.ClientIP())
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}
func (c *AuthController) Session(ctx *gin.Context) {
	api.Ok(ctx, gin.H{"id": ctx.GetUint("adminID"), "role": "admin"})
}
func (c *AuthController) Logout(ctx *gin.Context) {
	if e := c.Service.Logout(ctx.Request.Context(), ctx.GetString("sessionID")); e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, gin.H{"ok": true})
}
