package serializer

import (
	"fmt"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/hashid"
	"time"
)

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"user_name"`
	Nickname       string    `json:"nickname"`
	Status         int       `json:"status"`
	Avatar         string    `json:"avatar"`
	CreatedAt      time.Time `json:"created_at"`
	PreferredTheme string    `json:"preferred_theme"`
	Anonymous      bool      `json:"anonymous"`
	Group          group     `json:"group"`
	Tags           []tag     `json:"tags"`
}

type group struct {
	ID                   uint   `json:"id"`
	Name                 string `json:"name"`
	AllowShare           bool   `json:"allowShare"`
	AllowRemoteDownload  bool   `json:"allowRemoteDownload"`
	AllowArchiveDownload bool   `json:"allowArchiveDownload"`
	ShareDownload        bool   `json:"shareDownload"`
	CompressEnabled      bool   `json:"compress"`
	WebDAVEnabled        bool   `json:"webdav"`
	SourceBatchSize      int    `json:"sourceBatch"`
}

type tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Color      string `json:"color"`
	Type       int    `json:"type"`
	Expression string `json:"expression"`
}

type storage struct {
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Total uint64 `json:"total"`
}

func BuildUser(user models.User) User {
	tags, _ := models.GetTagsByUID(user.ID)
	return User{
		ID:             hashid.HashID(user.ID, hashid.UserID),
		Email:          user.Email,
		Nickname:       user.Nick,
		Status:         user.Status,
		Avatar:         user.Avatar,
		CreatedAt:      user.CreatedAt,
		PreferredTheme: user.OptionsSerialized.PreferredTheme,
		Anonymous:      user.IsAnonymous(),
		Group: group{
			ID:                   user.GroupID,
			Name:                 user.Group.Name,
			AllowShare:           user.Group.ShareEnabled,
			AllowRemoteDownload:  user.Group.OptionsSerialized.Aria2,
			AllowArchiveDownload: user.Group.OptionsSerialized.ArchiveDownload,
			ShareDownload:        user.Group.OptionsSerialized.ShareDownload,
			CompressEnabled:      user.Group.OptionsSerialized.ArchiveTask,
			WebDAVEnabled:        user.Group.WebDAVEnabled,
			SourceBatchSize:      user.Group.OptionsSerialized.SourceBatchSize,
		},
		Tags: buildTagRes(tags),
	}
}

func buildTagRes(tags []models.Tag) []tag {
	res := make([]tag, 0, len(tags))
	for i := 0; i < len(tags); i++ {
		newTag := tag{
			ID:    hashid.HashID(tags[i].ID, hashid.TagID),
			Name:  tags[i].Name,
			Icon:  tags[i].Icon,
			Color: tags[i].Color,
			Type:  tags[i].Type,
		}
		if newTag.Type != 0 {
			newTag.Expression = tags[i].Expression
		}
		res = append(res, newTag)
	}
	return res
}

func BuildUserResponse(user models.User) Response {
	return Response{
		Data: BuildUser(user),
	}
}

func BuildUserStorageResponse(user models.User) Response {
	total := user.Group.MaxStorage
	storageResp := storage{
		Used:  user.Storage,
		Free:  total - user.Storage,
		Total: total,
	}

	if total < user.Storage {
		storageResp.Free = 0
	}

	return Response{
		Data: storageResp,
	}
}

type WebAuthnCredentials struct {
	ID          []byte `json:"id"`
	FingerPrint string `json:"fingerprint"`
}

func BuildWebAuthnList(credentials []webauthn.Credential) []WebAuthnCredentials {
	res := make([]WebAuthnCredentials, 0, len(credentials))
	for _, v := range credentials {
		credential := WebAuthnCredentials{
			ID:          v.ID,
			FingerPrint: fmt.Sprintf("% X", v.Authenticator.AAGUID),
		}
		res = append(res, credential)
	}

	return res
}

func CheckLogin() Response {
	return Response{
		Code: CodeCheckLogin,
		Msg:  "未登录",
	}
}
