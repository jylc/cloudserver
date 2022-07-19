package serializer

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/hashid"
	"time"
)

type Object struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Pic           string    `json:"pic"`
	Size          uint64    `json:"size"`
	Type          string    `json:"type"`
	Date          time.Time `json:"date"`
	CreateDate    time.Time `json:"create_date"`
	Key           string    `json:"key,omitempty"`
	SourceEnabled bool      `json:"source_enabled"`
}

type ObjectList struct {
	Parent  string         `json:"parent,omitempty"`
	Objects []Object       `json:"objects"`
	Policy  *PolicySummary `json:"policy,omitempty"`
}

type PolicySummary struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	MaxSize  uint64   `json:"max_size"`
	FileType []string `json:"file_type"`
}

func BuildObjectList(parent uint, objects []Object, policy *models.Policy) ObjectList {
	res := ObjectList{
		Objects: objects,
	}
	if parent > 0 {
		res.Parent = hashid.HashID(parent, hashid.FolderID)
	}

	if policy != nil {
		res.Policy = &PolicySummary{
			ID:       hashid.HashID(policy.ID, hashid.PolicyID),
			Name:     policy.Name,
			Type:     policy.Type,
			MaxSize:  policy.MaxSize,
			FileType: policy.OptionsSerialized.FileType,
		}
	}
	return res
}
