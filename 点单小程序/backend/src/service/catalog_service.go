package service

import (
	"Boutique/src/api"
	"Boutique/src/dao"
	m "Boutique/src/model/http_model"
	"context"
	"net/url"
	"strings"
	"unicode/utf8"
)

type CatalogService struct{ Dao *dao.CatalogDao }

func (s *CatalogService) Catalog(ctx context.Context, admin bool) (map[string]any, error) {
	products, categories, setting, e := s.Dao.Catalog(ctx, admin)
	return map[string]any{"products": products, "categories": categories, "settings": setting, "deliveryFee": m.DeliveryFeeCents}, e
}
func (s *CatalogService) SaveProduct(ctx context.Context, id string, p m.ProductInput, adminID uint) (m.ProductView, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.Category = strings.TrimSpace(p.Category)
	if p.Name == "" || len(p.Name) > 100 || (p.CategoryID == 0 && p.Category == "") || len(p.Category) > 240 || p.Price < 1 || p.Price > 1000000 || len(p.Description) > 500 || len(p.Material) > 300 || len(p.Details) > 3000 || (p.Care != nil && len(*p.Care) > 3000) || len(p.English) > 200 || len(p.Image) > 1000 || len(p.SKUs) < 1 || len(p.SKUs) > 120 {
		return m.ProductView{}, api.Problem(400, "请检查名称、分类、价格、规格和描述长度")
	}
	if p.Gallery == nil && !validCatalogImage(p.Image) {
		return m.ProductView{}, api.Problem(400, "请上传图片，或填写站内图片路径及 HTTPS 图片地址")
	}
	keys := map[string]bool{}
	colors := map[string]bool{}
	for i, sku := range p.SKUs {
		sku.Variant = strings.TrimSpace(sku.Variant)
		sku.Color = strings.TrimSpace(sku.Color)
		p.SKUs[i] = sku
		key := strings.ToLower(sku.Variant) + "\x00" + strings.ToLower(sku.Color)
		if strings.TrimSpace(sku.Variant) == "" || strings.TrimSpace(sku.Color) == "" || len(sku.Variant) > 100 || len(sku.Color) > 100 || sku.Stock < 0 || sku.Stock > 100000 || keys[key] {
			return m.ProductView{}, api.Problem(400, "规格、颜色或库存无效，不能出现重复规格")
		}
		keys[key] = true
		colors[sku.Color] = true
	}
	if !validPhotoGroup(p.Gallery) {
		return m.ProductView{}, api.Problem(400, "商品相册最多 8 张，图片地址须有效且不重复")
	}
	for color, images := range p.ColorImages {
		if !colors[color] || !validPhotoGroup(images) {
			return m.ProductView{}, api.Problem(400, "颜色须与规格一致，每种颜色最多 8 张不同的图片")
		}
	}
	return s.Dao.SaveProduct(ctx, id, p, adminID)
}
func (s *CatalogService) SaveSettings(ctx context.Context, v m.ShopSetting, adminID uint) (m.ShopSetting, error) {
	v.QuickCover = strings.TrimSpace(v.QuickCover)
	if v.QuickCover != "" && (len(v.QuickCover) > 1000 || !validCatalogImage(v.QuickCover)) {
		return v, api.Problem(400, "请上传封面图片或填写有效的 HTTPS 图片地址")
	}
	if strings.TrimSpace(v.Name) == "" || len(v.Name) > 100 || len(v.Announcement) > 1000 || len(v.Location) > 300 || len(v.Hours) > 100 {
		return v, api.Problem(400, "店铺信息长度不正确")
	}
	return s.Dao.SaveSettings(ctx, v, adminID)
}

func (s *CatalogService) Dashboard(ctx context.Context, summaryOnly bool) (map[string]any, error) {
	return s.Dao.Dashboard(ctx, summaryOnly)
}

func (s *CatalogService) SaveCategory(ctx context.Context, id uint, name string, adminID uint) (m.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "全部商品" || utf8.RuneCountInString(name) > 60 {
		return m.Category{}, api.Problem(400, "请输入 1 至 60 字的分类名称，不能使用“全部商品”")
	}
	return s.Dao.SaveCategory(ctx, id, name, adminID)
}

func (s *CatalogService) DeleteCategory(ctx context.Context, id, adminID uint) error {
	return s.Dao.DeleteCategory(ctx, id, adminID)
}

func validPhotoGroup(images []string) bool {
	if len(images) > 8 {
		return false
	}
	seen := make(map[string]bool, len(images))
	for _, image := range images {
		if len(image) > 1000 || !validCatalogImage(image) || seen[image] {
			return false
		}
		seen[image] = true
	}
	return true
}

func validCatalogImage(path string) bool {
	if (strings.HasPrefix(path, "/images/") && !strings.Contains(path, "..")) || mediaPath.MatchString(path) {
		return true
	}
	u, err := url.Parse(path)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}
