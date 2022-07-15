package explorer

import (
	"encoding/gob"
	"github.com/jylc/cloudserver/pkg/hashid"
)

type ItemMoveService struct {
	SrcDir string        `json:"src_dir" binding:"required,min=1,max=65535"`
	Src    ItemIDService `json:"src"`
	Dst    string        `json:"dst" binding:"required,min=1,max=65535"`
}

type ItemRenameService struct {
	Src     ItemIDService `json:"src"`
	NewName string        `json:"new_name" binding:"required,min=1,max=255"`
}

type ItemService struct {
	Items []uint `json:"items"`
	Dirs  []uint `json:"dirs"`
}

type ItemIDService struct {
	Items  []string `json:"items"`
	Dirs   []string `json:"dirs"`
	Source *ItemService
}

func init() {
	gob.Register(ItemIDService{})
}

func (service *ItemIDService) Raw() *ItemService {
	if service.Source != nil {
		return service.Source
	}

	service.Source = &ItemService{
		Dirs:  make([]uint, 0, len(service.Dirs)),
		Items: make([]uint, 0, len(service.Items)),
	}

	for _, folder := range service.Dirs {
		id, err := hashid.DecodeHashID(folder, hashid.FolderID)
		if err == nil {
			service.Source.Dirs = append(service.Source.Dirs, id)
		}
	}

	for _, file := range service.Items {
		id, err := hashid.DecodeHashID(file, hashid.FileID)
		if err == nil {
			service.Source.Items = append(service.Source.Items, id)
		}
	}
	return service.Source
}
