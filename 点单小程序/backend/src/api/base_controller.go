package api

import (
	"Boutique/src/global"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ResponseJson struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
type BaseApi struct{ Logger *zap.SugaredLogger }

func NewBaseApi() BaseApi         { return BaseApi{Logger: global.Logger} }
func Ok(c *gin.Context, data any) { c.JSON(200, ResponseJson{200, "ok", data}) }
func Fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ResponseJson{status, message, nil})
}

type BusinessError struct {
	Status  int
	Message string
}

func (e *BusinessError) Error() string         { return e.Message }
func Problem(status int, message string) error { return &BusinessError{status, message} }
func RespondError(c *gin.Context, e error) {
	var business *BusinessError
	if errors.As(e, &business) {
		Fail(c, business.Status, business.Message)
		return
	}
	if global.Logger != nil {
		global.Logger.Errorw("接口处理失败", "module", "api", "op", "RespondError", "err", e)
	}
	Fail(c, 500, "服务暂时不可用，请稍后重试")
}

// BuildRequest 沿用参考工程的请求绑定入口，但结果按请求返回，不共享可变 Context。
type BuildRequestOption struct {
	Ctx *gin.Context
	DTO any
}
type RequestResult struct{ err error }

func (r RequestResult) GetError() error { return r.err }
func (b BaseApi) BuildRequest(option BuildRequestOption) RequestResult {
	e := option.Ctx.ShouldBindJSON(option.DTO)
	if e != nil {
		Fail(option.Ctx, 400, "请求参数不正确，请检查填写内容")
	}
	return RequestResult{e}
}
