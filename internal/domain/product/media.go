package product

// MediaType 媒体资源类型
type MediaType int8

const (
	MediaImage MediaType = iota + 1 // 图片
	MediaVideo                      // 视频
	MediaModel                      // 3D模型
)

// Media 商品媒体资源（值对象）：图片/视频/3D模型等展示素材。
// 只存类型和地址，资源文件本身由对象存储管理，这里不做存在性校验。
type Media struct {
	Type MediaType // 资源类型
	URL  string    // 资源地址
}

// validate 校验媒体字段
func (m Media) validate() error {
	if m.Type < MediaImage || m.Type > MediaModel {
		return ErrInvalidMediaType
	}
	if m.URL == "" {
		return ErrEmptyMediaURL
	}
	return nil
}

// validateMedias 校验媒体列表
func validateMedias(medias []Media) error {
	for _, m := range medias {
		if err := m.validate(); err != nil {
			return err
		}
	}
	return nil
}
