package conf

import "time"

var Dbc = &databaseConf{
	Type:        "mysql",
	Host:        "localhost",
	Port:        "3306",
	User:        "root",
	Password:    "Password123",
	Name:        "cloudserver_",
	TablePrefix: "cs_",
	Charset:     "utf8mb4",
}

var Rc = &redisConf{
	Server:   "localhost:6379",
	Password: "Password123",
	Db:       "0",
}

var Sc = &systemConf{
	AppMode:       "development",
	Port:          ":8888",
	SessionSecret: "cloudserver_session",
	HashIDSalt:    "something really hard to guss",
}

var Cc = &corsConfig{
	AllowOrigins:     []string{"UNSET"},
	AllowMethods:     []string{"PUT", "POST", "GET", "OPTIONS"},
	AllowHeaders:     []string{"Cookie", "X-Cr-Policy", "Authorization", "Content-Length", "Content-Type", "X-Cr-Path", "X-Cr-FileName"},
	ExposeHeaders:    nil,
	AllowCredentials: false,
	AllowOriginFunc:  nil,
	MaxAge:           12 * time.Hour,
}
