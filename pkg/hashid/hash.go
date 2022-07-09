package hashid

import (
	"errors"
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

var ErrTypeNotMatch = errors.New("ID type not match")

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

func HashDecode(raw string) ([]int, error) {
	hd := hashids.NewData()
	hd.Salt = conf.Sc.HashIDSalt

	data, err := hashids.NewWithData(hd)
	if err != nil {
		return []int{}, err
	}
	return data.DecodeWithError(raw)
}

func HashID(id uint, t int) string {
	v, _ := HashEncode([]int{int(id), t})
	return v
}

func DecodeHashID(id string, t int) (uint, error) {
	v, _ := HashDecode(id)
	if len(v) != 2 || v[1] != t {
		return 0, ErrTypeNotMatch
	}
	return uint(v[0]), nil
}
