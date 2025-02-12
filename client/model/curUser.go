package model

import (
	"chatroom/common/message"
	"net"
)

var (
	CurUsr CurUser
)

type CurUser struct {
	Conn net.Conn
	Usr  message.User
}
