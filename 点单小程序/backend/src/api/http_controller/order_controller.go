package http_controller

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

type OrderController struct {
	api.BaseApi
	Service *service.OrderService
}

func NewOrderController(s *service.OrderService) *OrderController {
	return &OrderController{api.NewBaseApi(), s}
}
func (c *OrderController) Create(ctx *gin.Context) {
	var in m.CreateOrderInput
	if e := c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &in}).GetError(); e != nil {
		return
	}
	if account, ok := currentCustomer(ctx); ok {
		in.CustomerID, in.Customer, in.Contact = account.ID, account.Name, account.Identifier
	}
	out, e := c.Service.Create(ctx.Request.Context(), in)
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	if in.CustomerID != "" {
		out.Token = ""
	}
	api.Ok(ctx, out)
}
func (c *OrderController) Lookup(ctx *gin.Context) {
	out, e := c.Service.Lookup(ctx.Request.Context(), ctx.Param("id"), ctx.GetHeader("X-Order-Token"))
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}
func (c *OrderController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "30"))
	if page < 1 || size < 1 || size > 200 || len(ctx.Query("search")) > 100 {
		api.Fail(ctx, 400, "分页参数不正确")
		return
	}
	items, total, e := c.Service.List(ctx.Request.Context(), page, size, ctx.Query("status"), ctx.Query("search"))
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, gin.H{"items": items, "total": total, "page": page, "size": size})
}
func (c *OrderController) Update(ctx *gin.Context) {
	var in m.UpdateOrderInput
	if e := c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &in}).GetError(); e != nil {
		return
	}
	if in.Version == 0 {
		api.Fail(ctx, 400, "请携带有效的订单版本号")
		return
	}
	out, e := c.Service.Update(ctx.Request.Context(), ctx.Param("id"), in, ctx.GetUint("adminID"))
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}

func (c *OrderController) CustomerOrders(ctx *gin.Context) {
	account, _ := currentCustomer(ctx)
	orders, err := c.Service.CustomerOrders(ctx.Request.Context(), account.ID)
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, orders)
}

func (c *OrderController) CustomerOrder(ctx *gin.Context) {
	account, _ := currentCustomer(ctx)
	order, err := c.Service.CustomerOrder(ctx.Request.Context(), account.ID, ctx.Param("id"))
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, order)
}
