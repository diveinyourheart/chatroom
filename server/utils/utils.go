package utils

import (
	"chatroom/common/message"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

type Transfer struct {
	Conn net.Conn
	buf  [8096]byte //传输时使用的缓冲
}

func Log(format string, args ...interface{}) {
	fmt.Printf("[%s] ", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf(format, args...)
	fmt.Println()
}

func (tf *Transfer) ReadPkg() (mes message.Message, er error) {
	Log("服务器端正在读取客户端发送的数据")
	n, err := tf.Conn.Read(tf.buf[:4])
	if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("客户端断开连接")
		} else {
			er = fmt.Errorf("服务器端读取客户端发送的消息长度失败：%v", err)
		}
		return
	} else if n != 4 {
		er = fmt.Errorf("客户端发送的消息长度信息存在丢包")
	}
	pkgLen := binary.BigEndian.Uint32(tf.buf[:n])
	n, err = tf.Conn.Read(tf.buf[:pkgLen])
	if n != int(pkgLen) {
		er = fmt.Errorf("客户端发送的消息本体存在丢包")
		return
	} else if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("客户端断开连接")
		} else {
			er = fmt.Errorf("客户端发送消息本体失败：%v", err)
		}
		return
	}
	err = json.Unmarshal(tf.buf[:pkgLen], &mes)
	if err != nil {
		er = fmt.Errorf("反序列化失败:%v", err)
		return
	}
	return
}

func (tf *Transfer) WritePkg(data []byte) (er error) {
	pkgLen := uint32(len(data))
	binary.BigEndian.PutUint32(tf.buf[0:4], pkgLen)
	n, err := tf.Conn.Write(tf.buf[0:4])
	if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("客户端断开连接")
		} else {
			er = fmt.Errorf("向客户端发送消息长度失败：%v", err)
		}
		return
	} else if n != 4 {
		er = fmt.Errorf("向客户端发送了错误的消息长度信息")
		return
	}
	n, err = tf.Conn.Write(data)
	if n != len(data) {
		er = fmt.Errorf("向客户端发送了错误的消息")
		return
	} else if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("客户端断开连接")
		} else {
			er = fmt.Errorf("向客户端发送消息失败：%v", err)
		}
		return
	}
	return
}
