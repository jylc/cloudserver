package serializer

import (
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"github.com/jylc/cloudserver/models"
)

type NodePingReq struct {
	SiteURL       string       `json:"site_url"`
	SiteID        string       `json:"site_id"`
	IsUpdate      bool         `json:"is_update"`
	CredentialTTL int          `json:"credential_ttl"`
	Node          *models.Node `json:"node"`
}

type NodePingResp struct {
}

type SlaveTransferReq struct {
	Src    string         `json:"src"`
	Dst    string         `json:"dst"`
	Policy *models.Policy `json:"policy"`
}

func (s *SlaveTransferReq) Hash(id string) string {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("transfer-%s-%s-%s-%d", id, s.Src, s.Dst, s.Policy.ID)))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

const (
	SlaveTransferSuccess = "success"
	SlaveTransferFailed  = "failed"
)

type SlaveTransferResult struct {
	Error string
}

func init() {
	gob.Register(SlaveTransferResult{})
}
