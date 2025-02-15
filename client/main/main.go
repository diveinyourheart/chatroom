package main

import (
	"bufio"
	"chatroom/client/model"
	"chatroom/client/processes"
	"chatroom/client/utils"
	"fmt"
	"os"
	"sync"
)

var (
	writeMutex sync.Mutex
)

func main() {
	go keyboardTextInputManager()
	go keyboardIntInputManager()
	go keyboardStringInputManager()
	for {
		// 显示主菜单
		key := showMenu1()

		// 根据用户的选择执行对应的功能
		switch key {
		case 1:
			// 登录聊天室
			handleLogin()
		case 2:
			// 注册用户
			handleRegister()
		case 3:
			// 退出系统
			os.Exit(0)
		default:
			fmt.Println("无效选择，请重新输入")
		}
	}
}

func keyboardTextInputManager() {
	for {
		<-utils.TextInputRequestChan
		writeMutex.Lock()
		fmt.Println("请输入文本，输入exit+回车退出文本输入，输入rewrite+回车重写文本")
		scnr := bufio.NewScanner(os.Stdin)
		var txt string
		for scnr.Scan() {
			newLine := scnr.Text() + "\n"
			if newLine == "exit\n" {
				break
			} else if newLine == "rewrite\n" {
				fmt.Printf("\n----------以下为重写内容----------\n")
				txt = ""
			} else {
				txt += newLine
			}
		}
		if err := scnr.Err(); err != nil {
			fmt.Println("读取文本输入时发生错误:", err)
			txt = ""
		}
		writeMutex.Unlock()
		utils.TextResChan <- txt
	}
}

func keyboardStringInputManager() {
	for {
		<-utils.StrInputRequestChan
		writeMutex.Lock()
		fmt.Println("tips:输入的字符串不能带有空格")
		var str string
		fmt.Scanln(&str)
		writeMutex.Unlock()
		utils.StrResChan <- str
	}
}

func keyboardIntInputManager() {
	for {
		<-utils.IntInputRequestChan
		writeMutex.Lock()
		var key int
		var isValid bool = false
		for !isValid {
			_, err := fmt.Scanf("%d\n", &key)
			if err != nil {
				fmt.Println("输入整数错误，请重新输入")
			} else {
				isValid = true
			}
		}
		writeMutex.Unlock()
		utils.IntResChan <- key
	}
}

// 显示菜单并接收用户选择
func showMenu1() int {
	fmt.Println("------------------欢迎OuQ------------------")
	fmt.Println("\t\t\t 1 登陆")
	fmt.Println("\t\t\t 2 注册用户")
	fmt.Println("\t\t\t 3 退出系统")
	fmt.Println("\t\t\t 请选择（1-3）：")

	var key int
	_, err := fmt.Scanf("%d\n", &key)
	if err != nil {
		fmt.Println("输入错误，请重新输入")
		return -1
	}
	return key
}

// 处理用户登录逻辑
func handleLogin() {
	fmt.Println("请输入用户的ID：")
	Id := utils.ReadIntInput()
	fmt.Println("请输入用户的密码：")
	Pwd := utils.ReadStringInput()

	// 调用 login 函数
	conn, err := processes.UserPrcs.Login(Id, Pwd)
	defer conn.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	//处理服务器端返回的登录信息
	model.CurUsr.Conn = conn
	fmt.Println(model.CurUsr.Usr.UserName, "登录成功")
	var wg sync.WaitGroup
	wg.Add(1)
	go processes.ProcessServerMes(conn, &wg)
	wg.Add(1)
	go processes.ShowMenu(&wg)
	wg.Add(1)
	go processes.FrdMgr.AcquireUnreadMesCount(&wg)
	wg.Add(1)
	go processes.GCMgr.AcquireUnreadMesCount(&wg)
	wg.Wait()
}

// 处理用户注册逻辑
func handleRegister() {
	fmt.Println("正在进行用户注册")
	fmt.Println("请输入用户的ID(不能为0):")
	Id := utils.ReadIntInput()
	for Id == 0 {
		fmt.Println("ID不能为0哦，请重新输入：")
		Id = utils.ReadIntInput()
	}
	fmt.Println("请输入用户的密码：")
	Pwd := utils.ReadStringInput()
	fmt.Println("请输入用户的名字：")
	name := utils.ReadStringInput()

	up := &processes.UserProcess{}
	err := up.Register(Id, Pwd, name)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("----------注册成功----------")
}
