package explorer

import (
	"context"
	model "github.com/cloudreve/Cloudreve/v3/models"
	"github.com/cloudreve/Cloudreve/v3/pkg/filesystem"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"strings"
)

type ItemSearchService struct {
	Type     string `uri:"type" binding:"required"`
	Keywords string `uri:"keywords" binding:"required"`
	Path     string `form:"path"`
}

func (service *ItemSearchService) Search(c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	if service.Path != "" {
		ok, parent := fs.IsPathExist(service.Path)
		if !ok {
			return serializer.Err(serializer.CodeParentNotExist, "Cannot find parent folder", nil)
		}
		fs.Root = parent
	}

	switch service.Type {
	case "keywords":
		return service.SearchKeywords(c, fs, "%"+service.Keywords+"%")
	case "image":
		return service.SearchKeywords(c, fs, "%.bmp", "%.iff", "%.png", "%.gif", "%.jpg", "%.jpeg", "%.psd", "%.svg", "%.webp")
	case "video":
		return service.SearchKeywords(c, fs, "%.mp4", "%.flv", "%.avi", "%.wmv", "%.mkv", "%.rm", "%.rmvb", "%.mov", "%.ogv")
	case "audio":
		return service.SearchKeywords(c, fs, "%.mp3", "%.flac", "%.ape", "%.wav", "%.acc", "%.ogg", "%.midi", "%.mid")
	case "doc":
		return service.SearchKeywords(c, fs, "%.txt", "%.md", "%.pdf", "%.doc", "%.docx", "%.ppt", "%.pptx", "%.xls", "%.xlsx", "%.pub")
	case "tag":
		if tid, err := hashid.DecodeHashID(service.Keywords, hashid.TagID); err == nil {
			if tag, err := model.GetTagsByID(tid, fs.User.ID); err == nil {
				if tag.Type == model.FileTagType {
					exp := strings.Split(tag.Expression, "\n")
					expInput := make([]interface{}, len(exp))
					for i := 0; i < len(exp); i++ {
						expInput[i] = exp[i]
					}
					return service.SearchKeywords(c, fs, expInput...)
				}
			}
		}
		return serializer.Err(serializer.CodeNotFound, "标签不存在", nil)
	default:
		return serializer.ParamErr("未知搜索类型", nil)
	}
}

func (service *ItemSearchService) SearchKeywords(c *gin.Context, fs *filesystem.FileSystem, keywords ...interface{}) serializer.Response {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	objects, err := fs.Search(ctx, keywords)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
		Data: map[string]interface{}{
			"parent":  0,
			"objects": objects,
		},
	}
}
