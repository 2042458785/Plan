package http_controller

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type CatalogController struct {
	api.BaseApi
	Service *service.CatalogService
}

func NewCatalogController(s *service.CatalogService) *CatalogController {
	return &CatalogController{api.NewBaseApi(), s}
}
func (c *CatalogController) Catalog(ctx *gin.Context) {
	out, e := c.Service.Catalog(ctx.Request.Context(), ctx.GetUint("adminID") != 0)
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}
func (c *CatalogController) SaveProduct(ctx *gin.Context) {
	var in m.ProductInput
	if e := c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &in}).GetError(); e != nil {
		return
	}
	id := ctx.Param("id")
	if len(id) > 48 {
		api.Fail(ctx, 400, "商品编号不正确")
		return
	}
	out, e := c.Service.SaveProduct(ctx.Request.Context(), id, in, ctx.GetUint("adminID"))
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}
func (c *CatalogController) SaveSettings(ctx *gin.Context) {
	var in m.ShopSetting
	if e := c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &in}).GetError(); e != nil {
		return
	}
	out, e := c.Service.SaveSettings(ctx.Request.Context(), in, ctx.GetUint("adminID"))
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}

func (c *CatalogController) Dashboard(ctx *gin.Context) {
	out, e := c.Service.Dashboard(ctx.Request.Context())
	if e != nil {
		api.RespondError(ctx, e)
		return
	}
	api.Ok(ctx, out)
}

func (c *CatalogController) SaveCategory(ctx *gin.Context) {
	var id uint64
	if ctx.Param("id") != "new" {
		var err error
		id, err = strconv.ParseUint(ctx.Param("id"), 10, strconv.IntSize)
		if err != nil || id == 0 {
			api.Fail(ctx, 400, "分类编号不正确")
			return
		}
	}
	var input m.CategoryInput
	if e := c.BuildRequest(api.BuildRequestOption{Ctx: ctx, DTO: &input}).GetError(); e != nil {
		return
	}
	category, err := c.Service.SaveCategory(ctx.Request.Context(), uint(id), input.Name, ctx.GetUint("adminID"))
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, category)
}

func (c *CatalogController) DeleteCategory(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, strconv.IntSize)
	if err != nil || id == 0 {
		api.Fail(ctx, 400, "分类编号不正确")
		return
	}
	if err := c.Service.DeleteCategory(ctx.Request.Context(), uint(id), ctx.GetUint("adminID")); err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, nil)
}

func (c *CatalogController) UploadImage(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, service.MaxImageSize+(64<<10))
	if err := ctx.Request.ParseMultipartForm(service.MaxImageSize); err != nil {
		api.Fail(ctx, 400, "请上传不超过 4 MB 的图片")
		return
	}
	defer ctx.Request.MultipartForm.RemoveAll()
	file, _, err := ctx.Request.FormFile("image")
	if err != nil {
		api.Fail(ctx, 400, "请选择商品图片")
		return
	}
	defer file.Close()
	path, err := c.Service.UploadImage(ctx.Request.Context(), file)
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	api.Ok(ctx, gin.H{"url": path})
}

func (c *CatalogController) Image(ctx *gin.Context) {
	image, err := c.Service.Image(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		api.RespondError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "public, max-age=31536000, immutable")
	ctx.Data(200, image.ContentType, image.Data)
}
