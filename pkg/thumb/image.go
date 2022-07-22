package thumb

import (
	"errors"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/utils"
	"golang.org/x/image/draw"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
)

type Thumb struct {
	src image.Image
	ext string
}

func NewThumbFromFile(file io.Reader, name string) (*Thumb, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if len(ext) == 0 {
		return nil, errors.New("Unknown image type")
	}

	var err error
	var img image.Image
	switch ext[1:] {
	case "jpg":
		img, err = jpeg.Decode(file)
	case "jpeg":
		img, err = jpeg.Decode(file)
	case "gif":
		img, err = gif.Decode(file)
	case "png":
		img, err = png.Decode(file)
	default:
		return nil, errors.New("Unknown image type")
	}
	if err != nil {
		return nil, err
	}

	return &Thumb{
		src: img,
		ext: ext[1:],
	}, nil
}

func (image *Thumb) CreateAvatar(uid uint) error {
	// 读取头像相关设定
	savePath := utils.RelativePath(models.GetSettingByName("avatar_path"))
	s := models.GetIntSetting("avatar_size_s", 50)
	m := models.GetIntSetting("avatar_size_m", 130)
	l := models.GetIntSetting("avatar_size_l", 200)

	// 生成头像缩略图
	src := image.src
	for k, size := range []int{s, m, l} {
		//image.src = resize.Resize(uint(size), uint(size), src, resize.Lanczos3)
		image.src = Resize(uint(size), uint(size), src)
		err := image.Save(filepath.Join(savePath, fmt.Sprintf("avatar_%d_%d.png", uid, k)))
		if err != nil {
			return err
		}
	}

	return nil
}

func Resize(newWidth, newHeight uint, img image.Image) image.Image {
	// Set the expected size that you want:
	dst := image.NewRGBA(image.Rect(0, 0, int(newWidth), int(newHeight)))
	// Resize:
	draw.BiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Src, nil)
	return dst
}

func (image *Thumb) Save(path string) (err error) {
	out, err := utils.CreatNestedFile(path)

	if err != nil {
		return err
	}
	defer out.Close()
	switch models.GetSettingByNameWithDefault("thumb_encode_method", "jpg") {
	case "png":
		err = png.Encode(out, image.src)
	default:
		err = jpeg.Encode(out, image.src, &jpeg.Options{Quality: models.GetIntSetting("thumb_encode_quality", 85)})
	}

	return err
}
