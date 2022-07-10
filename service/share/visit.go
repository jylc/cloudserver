package share

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type UserGetService struct {
	Type string `form:"type" binding:"required,eq=hot,eq=default"`
	Page uint   `form:"page" binding:"required,min=1"`
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
