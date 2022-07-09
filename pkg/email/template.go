package email

import (
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/utils"
)

func NewActivationEmail(userName, activateURL string) (string, string) {
	options := models.GetSettingByNames("siteName", "siteURL", "siteTitle", "mail_activation_template")
	replace := map[string]string{
		"{siteTitle}":     options["siteName"],
		"{userName}":      userName,
		"{activationUrl}": activateURL,
		"{siteUrl}":       options["siteURL"],
		"{siteSecTitle}":  options["siteTitle"],
	}
	return fmt.Sprintf("[%s] register avtivate", options["siteName"]),
		utils.Replace(options["mail_activation_template"], replace)
}
