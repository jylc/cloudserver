package serializer

import "github.com/jylc/cloudserver/models"

type NodePingReq struct {
	SiteURL       string       `json:"site_url"`
	SiteID        string       `json:"site_id"`
	IsUpdate      bool         `json:"is_update"`
	CredentialTTL int          `json:"credential_ttl"`
	Node          *models.Node `json:"node"`
}

type NodePingResp struct {
}
