package share

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/jylc/cloudserver/service/explorer"
	"net/http"
	"path"
)

type UserGetService struct {
	Type string `form:"type" binding:"required,eq=hot,eq=default"`
	Page uint   `form:"page" binding:"required,min=1"`
}

type ShareGetService struct {
	Password string `form:"password" binding:"max=255"`
}

type Service struct {
	Path string `form:"path" uri:"path" binding:"max=65535"`
}

type ArchiveService struct {
	Path  string   `json:"path" binding:"required,max=65535"`
	Items []string `json:"items"`
	Dirs  []string `json:"dirs"`
}

type SearchService struct {
	explorer.ItemSearchService
}

type ShareListService struct {
	Page     uint   `form:"page" binding:"required,min=1"`
	OrderBy  string `form:"order_by" binding:"required,eq=created_at|eq=downloads|eq=views"`
	Order    string `form:"order" binding:"required,eq=DESC|eq=ASC"`
	Keywords string `form:"keywords"`
}

func (service *ShareGetService) Get(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)

	unlocked := true
	if share.Password != "" {
		sessionKey := fmt.Sprintf("share_unlock_%d", share.ID)
		unlocked = utils.GetSession(c, sessionKey) != nil
		if !unlocked && service.Password != "" {
			if service.Password == share.Password {
				unlocked = true
				utils.SetSession(c, map[string]interface{}{sessionKey: true})
			}
		}
	}

	if unlocked {
		share.Viewed()
	}

	return serializer.Response{
		Code: 0,
		Data: serializer.BuildShareResponse(share, unlocked),
	}
}

func (service *UserGetService) Get(c *gin.Context) serializer.Response {
	userID, _ := c.Get("object_id")
	user, err := models.GetActivateUserByID(userID.(uint))
	if err != nil || user.OptionsSerialized.ProfileOff {
		return serializer.Err(serializer.CodeNotFound, "user is not exit", err)
	}
	hotNum := models.GetIntSetting("hot_share_num", 10)
	if service.Type == "default" {
		hotNum = 10
	}
	orderBy := "create_at desc"
	if service.Type == "hot" {
		orderBy = "views desc"
	}

	shares, total := models.ListShares(user.ID, int(service.Page), hotNum, orderBy, true)

	for i := 0; i < len(shares); i++ {
		shares[i].Source()
	}

	res := serializer.BuildShareList(shares, total)
	res.Data.(map[string]interface{})["user"] = struct {
		ID    string `json:"id"`
		Nick  string `json:"nick"`
		Group string `json:"group"`
		Date  string `json:"date"`
	}{
		hashid.HashID(user.ID, hashid.UserID),
		user.Nick,
		user.Group.Name,
		user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	return res
}

func (service *Service) CreateDownloadSession(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)

	fs, err := filesystem.NewFileSystem(user)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	err = fs.SetTargetByInterface(share.Source())
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, "源文件不存在", err)
	}

	ctx := context.Background()

	if share.IsDir {
		fs.Root = &fs.DirTarget[0]

		err = fs.ResetFileIfNotExist(ctx, service.Path)
		if err != nil {
			return serializer.Err(serializer.CodeNotSet, err.Error(), err)
		}
	}

	downloadURL, err := fs.GetDownloadURL(ctx, 0, "download_timeout")
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
		Data: downloadURL,
	}
}

func (service *Service) PreviewContent(ctx context.Context, c *gin.Context, isText bool) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)

	if share.IsDir {
		ctx = context.WithValue(ctx, fsctx.FolderModelCtx, share.Source())
		ctx = context.WithValue(ctx, fsctx.PathCtx, service.Path)
	} else {
		ctx = context.WithValue(ctx, fsctx.FileModelCtx, share.Source())
	}
	subService := explorer.FileIDService{}

	return subService.PreviewContent(ctx, c, isText)
}
func (service *Service) CreateDocPreviewSession(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)

	ctx := context.Background()
	if share.IsDir {
		ctx = context.WithValue(ctx, fsctx.FolderModelCtx, share.Source())
		ctx = context.WithValue(ctx, fsctx.PathCtx, service.Path)
	} else {
		ctx = context.WithValue(ctx, fsctx.FileModelCtx, share.Source())
	}
	subService := explorer.FileIDService{}

	return subService.CreateDocPreviewSession(ctx, c)
}

func (service *Service) List(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)

	if !share.IsDir {
		return serializer.ParamErr("This share cannot list directories", nil)
	}

	if !path.IsAbs(service.Path) {
		return serializer.ParamErr("invalid path", nil)
	}

	// 创建文件系统
	fs, err := filesystem.NewFileSystem(share.Creator())
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	// 上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 重设根目录
	fs.Root = share.Source().(*models.Folder)
	fs.Root.Name = "/"

	// 分享Key上下文
	ctx = context.WithValue(ctx, fsctx.ShareKeyCtx, hashid.HashID(share.ID, hashid.ShareID))

	// 获取子项目
	objects, err := fs.List(ctx, service.Path, nil)
	if err != nil {
		return serializer.Err(serializer.CodeCreateFolderFailed, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
		Data: serializer.BuildObjectList(0, objects, nil),
	}
}

func (service *Service) Thumb(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)

	if !share.IsDir {
		return serializer.ParamErr("This share has no thumbnails", nil)
	}

	fs, err := filesystem.NewFileSystem(share.Creator())
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()
	// 重设根目录
	fs.Root = share.Source().(*models.Folder)

	// 找到缩略图的父目录
	exist, parent := fs.IsPathExist(service.Path)
	if !exist {
		return serializer.Err(serializer.CodeNotFound, "path does not exist", nil)
	}

	ctx := context.WithValue(context.Background(), fsctx.LimitParentCtx, parent)

	// 获取文件ID
	fileID, err := hashid.DecodeHashID(c.Param("file"), hashid.FileID)
	if err != nil {
		return serializer.ParamErr("Unable to resolve file ID", err)
	}

	// 获取缩略图
	resp, err := fs.GetThumb(ctx, uint(fileID))
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "Unable to get thumbnail", err)
	}

	if resp.Redirect {
		c.Header("Cache-Control", fmt.Sprintf("max-age=%d", resp.MaxAge))
		c.Redirect(http.StatusMovedPermanently, resp.URL)
		return serializer.Response{Code: -1}
	}

	defer resp.Content.Close()
	http.ServeContent(c.Writer, c.Request, "thumb.png", fs.FileTarget[0].UpdatedAt, resp.Content)

	return serializer.Response{Code: -1}
}

func (service *ArchiveService) Archive(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)

	if !user.Group.OptionsSerialized.ArchiveDownload {
		return serializer.Err(serializer.CodeNoPermissionErr, "Your user group does not have permission to do this", nil)
	}

	if !share.IsDir {
		return serializer.ParamErr("This share cannot be packaged", nil)
	}

	fs, err := filesystem.NewFileSystem(user)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	fs.Root = share.Source().(*models.Folder)
	exist, parent := fs.IsPathExist(service.Path)
	if !exist {
		return serializer.Err(serializer.CodeNotFound, "path does not exist", nil)
	}

	ctx := context.WithValue(context.Background(), fsctx.LimitParentCtx, parent)
	tempUser := share.Creator()
	tempUser.Group.OptionsSerialized.ArchiveDownload = true
	c.Set("user", tempUser)

	subService := explorer.ItemIDService{
		Dirs:  service.Dirs,
		Items: service.Items,
	}

	return subService.Archive(ctx, c)
}

func (service *ShareListService) Search(c *gin.Context) serializer.Response {
	// 列出分享
	shares, total := models.SearchShares(int(service.Page), 18, service.OrderBy+" "+
		service.Order, service.Keywords)
	// 列出分享对应的文件
	for i := 0; i < len(shares); i++ {
		shares[i].Source()
	}

	return serializer.BuildShareList(shares, total)
}

func (service *ShareListService) List(c *gin.Context, user *models.User) serializer.Response {
	// 列出分享
	shares, total := models.ListShares(user.ID, int(service.Page), 18, service.OrderBy+" "+
		service.Order, false)
	// 列出分享对应的文件
	for i := 0; i < len(shares); i++ {
		shares[i].Source()
	}

	return serializer.BuildShareList(shares, total)
}
