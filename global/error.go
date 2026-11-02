package global

import "github.com/Streamlet/notesite/util"

var errorChan chan error

func InitErrorChan() chan error {
	util.Assert(errorChan == nil, "error chan was already initialized")
	errorChan = make(chan error)
	return errorChan
}

func GetErrorChan() chan error {
	util.Assert(errorChan != nil, "error chan MUST be initialized before using")
	return errorChan
}
