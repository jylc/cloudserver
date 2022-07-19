package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/email"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/service/admin"
	"io"
)

func AdminSummary(c *gin.Context) {
	var service admin.NoParamService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Summary()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminNews(c *gin.Context) {
	tag := "announcements"
	if c.Query("tag") != "" {
		tag = c.Query("tag")
	}
	r := request.NewClient()
	res := r.Request("GET", "https://forum.cloudreve.org/api/discussions?include=startUser%2ClastUser%2CstartPost%2Ctags&filter%5Bq%5D=%20tag%3A"+tag+"&sort=-startTime&page%5Blimit%5D=10", nil)
	if res.Err == nil {
		io.Copy(c.Writer, res.Response.Body)
	}
}

func AdminChangeSetting(c *gin.Context) {
	var service admin.BatchSettingChangeService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Change()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetSetting(c *gin.Context) {
	var service admin.BatchSettingGet
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Get()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetGroups(c *gin.Context) {
	var service admin.NoParamService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.GroupList()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
func AdminReloadService(c *gin.Context) {
	service := c.Param("service")
	switch service {
	case "email":
		email.Init()
	case "aria2":
		aria2.Init(true, cluster.Default, mq.GlobalMQ)
	}
	c.JSON(200, serializer.Response{})
}

func AdminSendTestMail(c *gin.Context) {
	var service admin.MailTestService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Send()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminTestAria2(c *gin.Context) {
	var service admin.Aria2TestService
	if err := c.ShouldBindJSON(&service); err == nil {
		var res serializer.Response
		if service.Type == models.MasterNodeType {
			res = service.TestMaster()
		} else {
			res = service.TestSlave()
		}
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListPolicy(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Policies()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminTestPath(c *gin.Context) {
	var service admin.PathTestService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Test()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminTestSlave(c *gin.Context) {
	var service admin.SlaveTestService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Test()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminAddPolicy(c *gin.Context) {
	var service admin.AddPolicyService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Add()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminAddCORS(c *gin.Context) {
	var service admin.PolicyService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.AddCORS()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminAddSCF(c *gin.Context) {
	var service admin.PolicyService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.AddSCF()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminOneDriveOAuth(c *gin.Context) {
	var service admin.PolicyService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.GetOAuth(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetPolicy(c *gin.Context) {
	var service admin.PolicyService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Get()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeletePolicy(c *gin.Context) {
	var service admin.PolicyService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Delete()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListGroup(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Groups()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminAddGroup(c *gin.Context) {
	var service admin.AddGroupService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Add()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeleteGroup(c *gin.Context) {
	var service admin.GroupService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetGroup(c *gin.Context) {
	var service admin.GroupService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Get()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListUser(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Users()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminAddUser(c *gin.Context) {
	var service admin.AddUserService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Add()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetUser(c *gin.Context) {
	var service admin.UserService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Get()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeleteUser(c *gin.Context) {
	var service admin.UserBatchService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminBanUser(c *gin.Context) {
	var service admin.UserService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Ban()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListFile(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Files()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetFile(c *gin.Context) {
	var service admin.FileService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Get(c)
		if res.Code == -301 {
			c.Redirect(302, res.Data.(string))
			return
		}

		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeleteFile(c *gin.Context) {
	var service admin.FileBatchService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListShare(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Shares()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
func AdminDeleteShare(c *gin.Context) {
	var service admin.ShareBatchService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListDownload(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Downloads()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeleteDownload(c *gin.Context) {
	var service admin.TaskBatchService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListTask(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Tasks()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeleteTask(c *gin.Context) {
	var service admin.TaskBatchService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.DeleteGeneral(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminCreateImportTask(c *gin.Context) {
	var service admin.ImportTaskService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Create(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListFolders(c *gin.Context) {
	var service admin.ListFolderService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.List(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminListNodes(c *gin.Context) {
	var service admin.ListService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Nodes()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminAddNode(c *gin.Context) {
	var service admin.AddNodeService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Add()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminToggleNode(c *gin.Context) {
	var service admin.ToggleNodeService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Toggle()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminDeleteNode(c *gin.Context) {
	var service admin.NodeService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Delete()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AdminGetNode(c *gin.Context) {
	var service admin.NodeService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Get()
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
