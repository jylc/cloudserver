package authn

import (
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/jylc/cloudserver/models"
)

func NewAuthnInstance() (*webauthn.WebAuthn, error) {
	base := models.GetSiteURL()
	return webauthn.New(&webauthn.Config{
		RPDisplayName: models.GetSettingByName("siteName"),
		RPID:          base.Hostname(),
		RPOrigin:      base.String(),
	})
}
