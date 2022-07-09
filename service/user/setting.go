package user

type Enable2FA struct {
	Code string `json:"code" binding:"required"`
}

type SettingService struct {
}
