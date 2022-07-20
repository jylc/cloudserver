package aria2

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type SelectFileService struct {
	Indexes []int `json:"indexes" binding:"required"`
}

type DownloadTaskService struct {
	GID string `json:"gid" binding:"required"`
}

type DownloadListService struct {
	Page uint `json:"page"`
}

func (service *SelectFileService) Select(c *gin.Context) serializer.Response {
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)
	download, err := models.GetDownloadByGid(c.Param("gid"), user.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Download record does not exist", err)
	}

	if download.StatusInfo.BitTorrent.Mode != "multi" || (download.Status != common.Downloading && download.Status != common.Paused) {
		return serializer.Err(serializer.CodeNoPermissionErr, "This download task cannot select a file", err)
	}

	node := cluster.Default.GetNodeByID(download.GetNodeID())
	if err := node.GetAria2Instance().Select(download, service.Indexes); err != nil {
		return serializer.Err(serializer.CodeNotSet, "operation failed", err)
	}

	return serializer.Response{}
}

func (service *DownloadTaskService) Delete(c *gin.Context) serializer.Response {
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)

	download, err := models.GetDownloadByGid(c.Param("gid"), user.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Download record does not exist", err)
	}

	if download.Status >= common.Error {
		if err := download.Delete(); err != nil {
			return serializer.Err(serializer.CodeDBError, "Task record deletion failed", err)
		}
		return serializer.Response{}
	}

	node := cluster.Default.GetNodeByID(download.GetNodeID())
	if node == nil {
		return serializer.Err(serializer.CodeInternalSetting, "Target node unavailable", err)
	}
	if err := node.GetAria2Instance().Cancel(download); err != nil {
		return serializer.Err(serializer.CodeNotSet, "operation failed", err)
	}
	return serializer.Response{}
}

func (service *DownloadListService) Downloading(c *gin.Context, user *models.User) serializer.Response {
	downloads := models.GetDownloadsByStatusAndUser(service.Page, user.ID, common.Downloading, common.Paused, common.Ready)
	intervals := make(map[uint]int)
	for _, download := range downloads {
		if _, ok := intervals[download.ID]; !ok {
			if node := cluster.Default.GetNodeByID(download.GetNodeID()); node != nil {
				intervals[download.ID] = node.DBModel().Aria2OptionsSerialized.Interval
			}
		}
	}

	return serializer.BuildDownloadingResponse(downloads, intervals)
}

func (service *DownloadListService) Finished(c *gin.Context, user *models.User) serializer.Response {
	downloads := models.GetDownloadsByStatusAndUser(service.Page, user.ID, common.Error, common.Complete, common.Canceled, common.Unknown)
	return serializer.BuildFinishedListResponse(downloads)
}
