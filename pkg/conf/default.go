package conf

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
}
