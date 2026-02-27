package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	"github.com/wenlng/go-captcha-assets/bindata/chars"
	"github.com/wenlng/go-captcha-assets/resources/fonts/fzshengsksjw"
	"github.com/wenlng/go-captcha-assets/resources/images"
	resourceTiles "github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/click"
	"github.com/wenlng/go-captcha/v2/rotate"
	"github.com/wenlng/go-captcha/v2/slide"
	"go.uber.org/zap"

	"gkipass/plane/internal/config"
)

/*
GoCaptchaService GoCaptcha 行为验证码服务
功能：集成 GoCaptcha 库，提供点选、滑动、拖拽、旋转四种验证码模式，
支持生成验证码图片和验证用户行为数据
*/
type GoCaptchaService struct {
	config *config.CaptchaConfig
	logger *zap.Logger

	clickCaptcha  click.Captcha
	slideCaptcha  slide.Captcha
	rotateCaptcha rotate.Captcha

	/* 验证码数据缓存（内存存储，后续可切换到 Redis） */
	store      sync.Map
	expiration time.Duration
}

/*
CaptchaData 验证码缓存数据
功能：存储生成的验证码答案信息，用于后续验证
*/
type CaptchaData struct {
	Mode      string      `json:"mode"`
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
}

/*
GoCaptchaGenerateResponse 验证码生成响应
功能：返回给前端的验证码图片和元数据
*/
type GoCaptchaGenerateResponse struct {
	CaptchaID   string `json:"captcha_id"`
	Mode        string `json:"mode"`
	MasterImage string `json:"master_image"`
	ThumbImage  string `json:"thumb_image,omitempty"`
	TileImage   string `json:"tile_image,omitempty"`
}

/*
GoCaptchaVerifyRequest 验证码验证请求
功能：接收前端提交的用户行为验证数据
*/
type GoCaptchaVerifyRequest struct {
	CaptchaID string          `json:"captcha_id"`
	Mode      string          `json:"mode"`
	Dots      json.RawMessage `json:"dots,omitempty"`
	X         float64         `json:"x,omitempty"`
	Y         float64         `json:"y,omitempty"`
	Angle     float64         `json:"angle,omitempty"`
}

/*
NewGoCaptchaService 创建 GoCaptcha 服务
功能：初始化所有支持的验证码模式（点选、滑动、旋转）
*/
func NewGoCaptchaService(cfg *config.CaptchaConfig) (*GoCaptchaService, error) {
	svc := &GoCaptchaService{
		config:     cfg,
		logger:     zap.L().Named("gocaptcha"),
		expiration: time.Duration(cfg.Expiration) * time.Second,
	}

	if svc.expiration == 0 {
		svc.expiration = 5 * time.Minute
	}

	/* 初始化点选验证码 */
	if err := svc.initClickCaptcha(); err != nil {
		return nil, fmt.Errorf("初始化点选验证码失败: %w", err)
	}

	/* 初始化滑动验证码 */
	if err := svc.initSlideCaptcha(); err != nil {
		return nil, fmt.Errorf("初始化滑动验证码失败: %w", err)
	}

	/* 初始化旋转验证码 */
	if err := svc.initRotateCaptcha(); err != nil {
		return nil, fmt.Errorf("初始化旋转验证码失败: %w", err)
	}

	/* 启动过期清理协程 */
	go svc.cleanupExpired()

	svc.logger.Info("GoCaptcha 服务初始化成功")
	return svc, nil
}

/*
initClickCaptcha 初始化点选验证码
功能：配置点选验证码的字符集、字体和背景图片
*/
func (s *GoCaptchaService) initClickCaptcha() error {
	builder := click.NewBuilder(
		click.WithRangeLen(option.RangeVal{Min: 4, Max: 6}),
		click.WithRangeVerifyLen(option.RangeVal{Min: 2, Max: 4}),
	)

	/* 加载字符集 */
	charList := chars.GetChineseChars()

	/* 加载字体（返回单个字体，包装为切片） */
	font, err := fzshengsksjw.GetFont()
	if err != nil {
		return fmt.Errorf("加载字体失败: %w", err)
	}

	/* 加载背景图片 */
	bgImages, err := images.GetImages()
	if err != nil {
		return fmt.Errorf("加载背景图片失败: %w", err)
	}

	builder.SetResources(
		click.WithChars(charList),
		click.WithFonts([]*truetype.Font{font}),
		click.WithBackgrounds(bgImages),
	)

	s.clickCaptcha = builder.Make()
	return nil
}

/*
initSlideCaptcha 初始化滑动验证码
功能：配置滑动验证码的背景图、滑块图等资源
*/
func (s *GoCaptchaService) initSlideCaptcha() error {
	builder := slide.NewBuilder()

	/* 加载背景图片 */
	bgImages, err := images.GetImages()
	if err != nil {
		return fmt.Errorf("加载背景图片失败: %w", err)
	}

	/* 加载滑块图片（需要从 tiles.GraphImage 转换为 slide.GraphImage） */
	tileGraphs, err := resourceTiles.GetTiles()
	if err != nil {
		return fmt.Errorf("加载滑块图片失败: %w", err)
	}

	var slideGraphs []*slide.GraphImage
	for _, tg := range tileGraphs {
		slideGraphs = append(slideGraphs, &slide.GraphImage{
			OverlayImage: tg.OverlayImage,
			ShadowImage:  tg.ShadowImage,
			MaskImage:    tg.MaskImage,
		})
	}

	builder.SetResources(
		slide.WithBackgrounds(bgImages),
		slide.WithGraphImages(slideGraphs),
	)

	s.slideCaptcha = builder.Make()
	return nil
}

/*
initRotateCaptcha 初始化旋转验证码
功能：配置旋转验证码的背景图资源
*/
func (s *GoCaptchaService) initRotateCaptcha() error {
	builder := rotate.NewBuilder()

	/* 加载背景图片 */
	bgImages, err := images.GetImages()
	if err != nil {
		return fmt.Errorf("加载背景图片失败: %w", err)
	}

	builder.SetResources(
		rotate.WithImages(bgImages),
	)

	s.rotateCaptcha = builder.Make()
	return nil
}

/*
Generate 生成验证码
功能：根据指定模式生成对应类型的验证码图片和验证数据
*/
func (s *GoCaptchaService) Generate(mode string) (*GoCaptchaGenerateResponse, error) {
	if mode == "" {
		mode = s.config.GoCaptchaMode
	}
	if mode == "" {
		mode = "click"
	}

	captchaID := uuid.New().String()

	switch mode {
	case "click":
		return s.generateClick(captchaID)
	case "slide":
		return s.generateSlide(captchaID)
	case "rotate":
		return s.generateRotate(captchaID)
	default:
		return nil, fmt.Errorf("不支持的验证码模式: %s", mode)
	}
}

/*
generateClick 生成点选验证码
功能：生成文字点选验证码，返回主图和缩略图
*/
func (s *GoCaptchaService) generateClick(captchaID string) (*GoCaptchaGenerateResponse, error) {
	captData, err := s.clickCaptcha.Generate()
	if err != nil {
		return nil, fmt.Errorf("生成点选验证码失败: %w", err)
	}

	dotData := captData.GetData()
	if dotData == nil {
		return nil, fmt.Errorf("生成点选验证码数据为空")
	}

	/* 缓存验证数据 */
	s.store.Store(captchaID, &CaptchaData{
		Mode:      "click",
		Data:      dotData,
		CreatedAt: time.Now(),
	})

	/* 获取图片 Base64 */
	masterImage, err := captData.GetMasterImage().ToBase64()
	if err != nil {
		return nil, fmt.Errorf("编码主图失败: %w", err)
	}
	thumbImage, err := captData.GetThumbImage().ToBase64()
	if err != nil {
		return nil, fmt.Errorf("编码缩略图失败: %w", err)
	}

	return &GoCaptchaGenerateResponse{
		CaptchaID:   captchaID,
		Mode:        "click",
		MasterImage: masterImage,
		ThumbImage:  thumbImage,
	}, nil
}

/*
generateSlide 生成滑动验证码
功能：生成滑块验证码，返回主图和滑块图
*/
func (s *GoCaptchaService) generateSlide(captchaID string) (*GoCaptchaGenerateResponse, error) {
	captData, err := s.slideCaptcha.Generate()
	if err != nil {
		return nil, fmt.Errorf("生成滑动验证码失败: %w", err)
	}

	blockData := captData.GetData()
	if blockData == nil {
		return nil, fmt.Errorf("生成滑动验证码数据为空")
	}

	/* 缓存验证数据 */
	s.store.Store(captchaID, &CaptchaData{
		Mode:      "slide",
		Data:      blockData,
		CreatedAt: time.Now(),
	})

	/* 获取图片 Base64 */
	masterImage, err := captData.GetMasterImage().ToBase64()
	if err != nil {
		return nil, fmt.Errorf("编码主图失败: %w", err)
	}
	tileImage, err := captData.GetTileImage().ToBase64()
	if err != nil {
		return nil, fmt.Errorf("编码滑块图失败: %w", err)
	}

	return &GoCaptchaGenerateResponse{
		CaptchaID:   captchaID,
		Mode:        "slide",
		MasterImage: masterImage,
		TileImage:   tileImage,
	}, nil
}

/*
generateRotate 生成旋转验证码
功能：生成旋转验证码，返回主图和缩略图
*/
func (s *GoCaptchaService) generateRotate(captchaID string) (*GoCaptchaGenerateResponse, error) {
	captData, err := s.rotateCaptcha.Generate()
	if err != nil {
		return nil, fmt.Errorf("生成旋转验证码失败: %w", err)
	}

	blockData := captData.GetData()
	if blockData == nil {
		return nil, fmt.Errorf("生成旋转验证码数据为空")
	}

	/* 缓存验证数据 */
	s.store.Store(captchaID, &CaptchaData{
		Mode:      "rotate",
		Data:      blockData,
		CreatedAt: time.Now(),
	})

	/* 获取图片 Base64 */
	masterImage, err := captData.GetMasterImage().ToBase64()
	if err != nil {
		return nil, fmt.Errorf("编码主图失败: %w", err)
	}
	thumbImage, err := captData.GetThumbImage().ToBase64()
	if err != nil {
		return nil, fmt.Errorf("编码缩略图失败: %w", err)
	}

	return &GoCaptchaGenerateResponse{
		CaptchaID:   captchaID,
		Mode:        "rotate",
		MasterImage: masterImage,
		ThumbImage:  thumbImage,
	}, nil
}

/*
Verify 验证验证码
功能：根据模式验证用户提交的行为数据是否正确
*/
func (s *GoCaptchaService) Verify(req *GoCaptchaVerifyRequest) (bool, error) {
	/* 获取缓存的验证数据 */
	val, ok := s.store.LoadAndDelete(req.CaptchaID)
	if !ok {
		return false, fmt.Errorf("验证码不存在或已过期")
	}

	cached := val.(*CaptchaData)

	/* 检查过期 */
	if time.Since(cached.CreatedAt) > s.expiration {
		return false, fmt.Errorf("验证码已过期")
	}

	switch cached.Mode {
	case "click":
		return s.verifyClick(cached, req)
	case "slide":
		return s.verifySlide(cached, req)
	case "rotate":
		return s.verifyRotate(cached, req)
	default:
		return false, fmt.Errorf("不支持的验证码模式: %s", cached.Mode)
	}
}

/*
verifyClick 验证点选验证码
功能：比对用户点击坐标与正确答案坐标，允许一定误差范围
*/
func (s *GoCaptchaService) verifyClick(cached *CaptchaData, req *GoCaptchaVerifyRequest) (bool, error) {
	/* 解析用户提交的点击坐标 */
	var userDots []map[string]interface{}
	if err := json.Unmarshal(req.Dots, &userDots); err != nil {
		return false, fmt.Errorf("解析点击数据失败: %w", err)
	}

	/* 获取正确答案 */
	dotMap, ok := cached.Data.(map[int]*click.Dot)
	if !ok {
		return false, fmt.Errorf("验证数据格式错误")
	}

	/* 检查数量是否匹配 */
	if len(userDots) != len(dotMap) {
		return false, nil
	}

	/* 逐个比对坐标（允许 25px 的误差） */
	const padding = 25
	for i, userDot := range userDots {
		correctDot, exists := dotMap[i]
		if !exists {
			return false, nil
		}

		ux, _ := userDot["x"].(float64)
		uy, _ := userDot["y"].(float64)

		if !click.Validate(int(ux), int(uy), correctDot.X, correctDot.Y, correctDot.Width, correctDot.Height, padding) {
			return false, nil
		}
	}

	return true, nil
}

/*
verifySlide 验证滑动验证码
功能：比对用户滑动的最终位置与正确目标位置
*/
func (s *GoCaptchaService) verifySlide(cached *CaptchaData, req *GoCaptchaVerifyRequest) (bool, error) {
	block, ok := cached.Data.(*slide.Block)
	if !ok {
		return false, fmt.Errorf("验证数据格式错误")
	}

	/* 允许 5px 的误差 */
	const tolerance = 5.0
	dx := math.Abs(req.X - float64(block.X))
	if dx > tolerance {
		return false, nil
	}

	return true, nil
}

/*
verifyRotate 验证旋转验证码
功能：比对用户旋转的角度与正确角度
*/
func (s *GoCaptchaService) verifyRotate(cached *CaptchaData, req *GoCaptchaVerifyRequest) (bool, error) {
	block, ok := cached.Data.(*rotate.Block)
	if !ok {
		return false, fmt.Errorf("验证数据格式错误")
	}

	/* 允许 5 度的误差 */
	const tolerance = 5.0
	da := math.Abs(req.Angle - float64(block.Angle))
	if da > tolerance {
		return false, nil
	}

	return true, nil
}

/*
cleanupExpired 清理过期的验证码数据
功能：定期扫描并删除已过期的验证码缓存，防止内存泄漏
*/
func (s *GoCaptchaService) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.store.Range(func(key, value interface{}) bool {
			data := value.(*CaptchaData)
			if now.Sub(data.CreatedAt) > s.expiration {
				s.store.Delete(key)
			}
			return true
		})
	}
}
