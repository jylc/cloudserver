package hashid

import (
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/speps/go-hashids"
)

const (
	ShareID = iota
	UserID
	FileID
	FolderID
	TagID
	PolicyID
)

func HashEncode(v []int) (string, error) {
	hd := hashids.NewData()
	hd.Salt = conf.Sc.HashIDSalt

	data, err := hashids.NewWithData(hd)
	if err != nil {
		return "", err
	}
	id, err := data.Encode(v)
	if err != nil {
		return "", err
	}
	return id, nil
}

func HashID(id uint, t int) string {
	v, _ := HashEncode([]int{int(id), t})
	return v
}
