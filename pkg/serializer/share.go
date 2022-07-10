package serializer

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/hashid"
	"time"
)

type Share struct {
	Key        string        `json:"key"`
	Locked     bool          `json:"locked"`
	IsDir      bool          `json:"is_dir"`
	CreateData time.Time     `json:"create_data,omitempty"`
	Downloads  int           `json:"downloads"`
	Views      int           `json:"views"`
	Expire     int64         `json:"expire"`
	Preview    bool          `json:"preview"`
	Creator    *shareCreator `json:"creator,omitempty"`
	Source     *shareCreator `json:"source,omitempty"`
}

type shareCreator struct {
	Key       string `json:"key"`
	Nick      string `json:"nick"`
	GroupName string `json:"group_name"`
}

type shareSource struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

type myShareItem struct {
	Key             string       `json:"key"`
	IsDir           bool         `json:"is_dir"`
	Password        string       `json:"password"`
	CreateDate      time.Time    `json:"create_date,omitempty"`
	Downloads       int          `json:"downloads"`
	RemainDownloads int          `json:"remain_downloads"`
	Views           int          `json:"views"`
	Expire          int64        `json:"expire"`
	Preview         bool         `json:"preview"`
	Source          *shareSource `json:"source,omitempty"`
}

func BuildShareList(shares []models.Share, total int) Response {
	res := make([]myShareItem, 0, total)
	now := time.Now().Unix()
	for i := 0; i < len(shares); i++ {
		item := myShareItem{
			Key:             hashid.HashID(shares[i].ID, hashid.ShareID),
			IsDir:           shares[i].IsDir,
			Password:        shares[i].Password,
			CreateDate:      shares[i].CreatedAt,
			Downloads:       shares[i].Downloads,
			RemainDownloads: shares[i].RemainDownloads,
			Views:           shares[i].Views,
			Expire:          -1,
			Preview:         shares[i].PreviewEnabled,
		}

		if shares[i].Expires != nil {
			item.Expire = shares[i].Expires.Unix() - now
			if item.Expire == 0 {
				item.Expire = 0
			}
		}

		if shares[i].File.ID != 0 {
			item.Source = &shareSource{
				Name: shares[i].File.Name,
				Size: shares[i].File.Size,
			}
		} else if shares[i].Folder.ID != 0 {
			item.Source = &shareSource{
				Name: shares[i].Folder.Name,
			}
		}
		res = append(res, item)
	}
	return Response{
		Data: map[string]interface{}{
			"total": total,
			"items": res,
		},
	}
}
