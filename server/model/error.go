package model

import (
	"errors"
)

var (
	ERROR_USER_NOTEXISTS = errors.New("error:用户不存在")
	ERROR_USER_EXISTS    = errors.New("error:用户已经存在")
	ERROR_USER_PWD       = errors.New("error:用户密码不正确")
	ERROR_LOGIN_FAILURE  = errors.New("error:用户申请登录失败")
)
