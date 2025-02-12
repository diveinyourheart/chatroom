package main

import (
	"chatroom/server/model"
	"chatroom/server/utils"
	"net"
	"time"
)

func process(conn net.Conn) {
	defer conn.Close()
	defer func() {
		if r := recover(); r != nil {
			utils.Log("Recovered from panic: %v", r)
		}
	}()
	processor := &Processor{
		Conn:          conn,
		RemoteAddress: conn.RemoteAddr().String(),
	}
	processor.process2()
}

func initUserDao() {
	model.MyUserDao = model.NewUserDao(pl)
}

func main() {
	initPool("localhost:6379", 16, 0, 300*time.Second)
	initUserDao()
	utils.Log("服务器在8889端口监听......")
	listen, err := net.Listen("tcp", "0.0.0.0:8889")
	if err != nil {
		utils.Log("net.listen err = %v", err)
		return
	}
	defer listen.Close()

	for {
		utils.Log("等待客户端来连接服务器")
		conn, err := listen.Accept()
		if err != nil {
			utils.Log("listen.Accept err=%v", err)
			continue
		}
		go process(conn)
	}
}
