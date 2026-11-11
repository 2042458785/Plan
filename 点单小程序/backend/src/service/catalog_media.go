package service

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"

	"gorm.io/gorm"
)

const MaxImageSize = 4 << 20

var mediaID = regexp.MustCompile(`^[a-f0-9]{48}$`)
var mediaPath = regexp.MustCompile(`^/api/v1/media/[a-f0-9]{48}$`)

func (s *CatalogService) UploadImage(ctx context.Context, file io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(file, MaxImageSize+1))
	if err != nil || len(data) > MaxImageSize {
		return "", api.Problem(400, "图片读取失败或超过 4 MB")
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
	default:
		return "", api.Problem(400, "请上传 JPG、PNG、WebP 或 GIF 图片")
	}
	image := m.ProductImage{ID: utils.RandomID(), ContentType: contentType, Data: data}
	if err := s.Dao.CreateImage(ctx, &image); err != nil {
		return "", err
	}
	return "/api/v1/media/" + image.ID, nil
}

func (s *CatalogService) Image(ctx context.Context, id string) (m.ProductImage, error) {
	if !mediaID.MatchString(id) {
		return m.ProductImage{}, api.Problem(404, "图片不存在")
	}
	image, err := s.Dao.FindImage(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m.ProductImage{}, api.Problem(404, "图片不存在")
	}
	return image, err
}
