package utils

import (
	"chatroom/common/message"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const (
	MAXIMUM_NUMBER_OF_SIMULTANEOUS_REQUESTS = 5
)

var (
	IntResChan           = make(chan int)
	IntInputRequestChan  = make(chan struct{}, MAXIMUM_NUMBER_OF_SIMULTANEOUS_REQUESTS)
	StrResChan           = make(chan string)
	StrInputRequestChan  = make(chan struct{}, MAXIMUM_NUMBER_OF_SIMULTANEOUS_REQUESTS)
	TextResChan          = make(chan string)
	TextInputRequestChan = make(chan struct{}, MAXIMUM_NUMBER_OF_SIMULTANEOUS_REQUESTS)
)

type Transfer struct {
	Conn net.Conn
	Buf  [8096]byte //传输时使用的缓冲
}

func (tf *Transfer) ReadPkg() (mes message.Message, er error) {
	n, err := tf.Conn.Read(tf.Buf[:4])
	if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("服务器端断开连接")
		} else {
			er = fmt.Errorf("读取服务器端发送的消息长度失败：%v", err)
		}
		return
	} else if n != 4 {
		er = fmt.Errorf("服务器端发送的消息长度信息存在丢包")
	}
	pkgLen := binary.BigEndian.Uint32(tf.Buf[:n])
	n, err = tf.Conn.Read(tf.Buf[:pkgLen])
	if n != int(pkgLen) {
		er = fmt.Errorf("服务器端发送的消息本体存在丢包")
		return
	} else if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("服务器端断开连接")
		} else {
			er = fmt.Errorf("服务器端发送消息本体失败：%v", err)
		}
		return
	}
	err = json.Unmarshal(tf.Buf[:pkgLen], &mes)
	if err != nil {
		er = fmt.Errorf("反序列化失败:%v", err)
		return
	}
	return
}

func (tf *Transfer) WritePkg(data []byte) (er error) {
	pkgLen := uint32(len(data))
	binary.BigEndian.PutUint32(tf.Buf[0:4], pkgLen)
	n, err := tf.Conn.Write(tf.Buf[0:4])
	if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("服务器端断开连接")
		} else {
			er = fmt.Errorf("向服务器端发送消息长度失败：%v", err)
		}
		return
	} else if n != 4 {
		er = fmt.Errorf("向服务器端发送了错误的消息长度信息")
		return
	}
	n, err = tf.Conn.Write(data)
	if n != len(data) {
		er = fmt.Errorf("向服务器端发送了错误的消息")
		return
	} else if err != nil {
		if err == io.EOF {
			er = fmt.Errorf("服务器端断开连接")
		} else {
			er = fmt.Errorf("向服务器端发送消息失败：%v", err)
		}
		return
	}
	return
}

// 读取整数输入
func ReadIntInput() int {
	IntInputRequestChan <- struct{}{}
	return <-IntResChan
}

// 读取字符串输入
func ReadStringInput() string {
	StrInputRequestChan <- struct{}{}
	return <-StrResChan
}

// 读取文本内容用于发送消息
func ReadTextInput() string {
	TextInputRequestChan <- struct{}{}
	return <-TextResChan
}
