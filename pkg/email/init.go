package email

import "sync"

var Client Driver
var Lock sync.RWMutex
