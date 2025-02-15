package processes

import (
	"chatroom/client/model"
	"chatroom/client/utils"
	"chatroom/common/message"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

var (
	SERVER_IPv4_ADDRESS      string = "0.0.0.0:8889"
	logOutChan                      = make(chan struct{}, 3)
	blockShowMenuChan               = make(chan struct{})
	gCMesRealTimeDisplayChan        = make(chan struct{}, 1)
	newGCMesChan                    = make(chan *ForStoringMes)
	quitGroupChatChan               = make(chan struct{})
	updateUnreadGCMesDone           = make(chan struct{})
	mu                       sync.Mutex
)

func HandleLogOut() {
	model.CurUsr.Conn.Close()
	model.CurUsr = model.CurUser{}
	FrdMgr.NewFriendMgr()
	GCMgr.NewGroupChatMgr()
	logOutChan <- struct{}{}
	logOutChan <- struct{}{}
	logOutChan <- struct{}{}
}

// 显示登录成功后的界面
func ShowMenu(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-blockShowMenuChan:
			<-blockShowMenuChan
		case <-logOutChan:
			return
		default:
			fmt.Println("----------1. 显示好友列表----------")
			fmt.Println("----------2. 发送消息----------")
			fmt.Println("----------3. 消息列表----------")
			fmt.Println("----------4. 添加新的好友----------")
			fmt.Println("----------5. 显示群聊列表----------")
			fmt.Println("----------6. 创建群聊----------")
			fmt.Println("----------7. 加入群聊----------")
			fmt.Println("----------8. 退出登录----------")
			fmt.Println("----------9. 退出系统----------")
			fmt.Println("请选择（1-6）：")
			key := utils.ReadIntInput()
			if model.CurUsr.Usr.UserId == 0 {
				//说明刚刚在此客户端登录的用户在远端异地登录，此客户端的用户被动登出
				fmt.Println("此账号已在异端登录")
				return
			}
			switch key {
			case 1:
				FrdMgr.outputFriendsList()
			case 2:
				FrdMgr.outputFriendsList()
				fmt.Println("请输入目标用户的ID：")
				desId := utils.ReadIntInput()
				if _, exist := FrdMgr.GetFriendNameById(desId); !exist || desId == model.CurUsr.Usr.UserId {
					fmt.Printf("%d不是你的好友\n", desId)
					continue
				}
				fmt.Println("请输入消息内容：")
				content := utils.ReadTextInput()
				err := SmsPrcs.sendOneToOneMesById(desId, content)
				if err != nil {
					fmt.Println("发送失败：", err)
					continue
				}
				fmt.Println("发送成功！")
			case 3:
				FrdMgr.outputFriendMesList()
				fmt.Println("请输入你想要查看消息的用户的ID")
				targetID := utils.ReadIntInput()
				if _, exist := FrdMgr.GetFriendNameById(targetID); !exist || targetID == model.CurUsr.Usr.UserId {
					fmt.Printf("%d不是你的好友\n", targetID)
				} else {
					readMsgCountChan <- targetID
					FileRW.GenerateFriendMesList(targetID)
				}
			case 4:
				fmt.Println("请输入你想要添加的用户的ID：")
				strangerId := utils.ReadIntInput()
				_, exist := FrdMgr.GetFriendNameById(strangerId)
				if exist {
					fmt.Printf("%d已经是你的好友了\n", strangerId)
				} else {
					fmt.Println("输入备注：")
					note := utils.ReadTextInput()
					err := UserPrcs.sendAddFriendRequest(strangerId, note)
					if err != nil {
						fmt.Println("发送好友申请失败：", err)
					} else {
						fmt.Println("添加好友申请发送成功")
					}
				}
			case 5:
				GCMgr.outputGCMesList()
				fmt.Println("----------输入群ID进行下一步操作----------")
				inputID := utils.ReadIntInput()
				GCInfo := GCMgr.GetGroupChatInfoByID(inputID)
				for GCInfo == nil {
					fmt.Println("你的群聊列表中没有这一群聊，请重新输入")
					inputID = utils.ReadIntInput()
					GCInfo = GCMgr.GetGroupChatInfoByID(inputID)
				}
				fmt.Println("----------可选操作----------")
				fmt.Println("----------0. 回到上一级菜单----------")
				fmt.Println("----------1. 进入群聊----------")
				fmt.Println("----------2. 显示群成员----------")
				cnt := 2
				if GCInfo.GroupLeader == model.CurUsr.Usr.UserId {
					cnt++
					fmt.Printf("----------%d. 添加群成员为管理员----------\n", cnt)
				}
				inputSelect := utils.ReadIntInput()
				switch inputSelect {
				case 0:
					continue
				case 1:
					readGCMsgCountChan <- GCInfo.GroupID
					go GCMesRealTimeDisplay()
					FileRW.outputGroupChatHistoryMes(GCInfo.GroupID)
					fmt.Println("已进入聊天室，你可以输入你想要发送的文本信息，输入quit+回车退出聊天室")
					fmt.Printf("\n------------------------------------------------------------\n")
					var txt string
					for {
						txt = utils.ReadTextInput()
						if txt == "quit\n" {
							quitGroupChatChan <- struct{}{}
							break
						}
						SmsPrcs.SendGroupChatMes(model.CurUsr.Usr.UserId, GCInfo, txt)
					}
				case 2:
					GCMgr.OutputGCMembers(GCInfo)
				case 3:
					if cnt >= 3 {
						err := UserPrcs.GCOwnerAddGCManager(GCInfo)
						if err == nil {
							fmt.Println("已请求服务器")
						} else {
							fmt.Println("请求服务器失败", err)
						}
					}
				default:
					continue
				}
			case 6:
				fmt.Println("请输入群聊名称,按回车键跳过")
				GCName := utils.ReadStringInput()
				err := UserPrcs.CreateGroupChat(GCName)
				if err != nil {
					fmt.Println("创建群聊失败：", err)
				} else {
					fmt.Println("群聊创建申请发送成功")
				}
			case 7:
				fmt.Println("请输入你想加入的群聊的ID:")
				GCID := utils.ReadIntInput()
				_, exist := GCMgr.GetGCNameById(GCID)
				if exist {
					fmt.Println("你已在该群聊中")
				} else {
					fmt.Println("请输入备注，仅群主和管理员可见")
					note := utils.ReadStringInput()
					err := UserPrcs.Apply2JoinAGC(GCID, note)
					if err != nil {
						fmt.Println("申请加入群聊失败：", err)
					} else {
						fmt.Println("加入群聊申请发送成功")
					}
				}
			case 8:
				HandleLogOut()
				return
			case 9:
				fmt.Println("你选择了退出系统...")
				os.Exit(0)
			default:
				fmt.Println("你输入的选项不正确...")
			}
		}
	}
}

func ProcessServerMes(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	//创建一个transfer实例，不停的读取服务器发送的消息
	tf := utils.Transfer{
		Conn: conn,
	}
	for {
		mes, err := tf.ReadPkg()
		if err != nil {
			//这里可以创建日志文件用来记录读包的错误是什么
			//结合model.CurUsr.Usr.UserId来判断是网络错误
			//还是执行了HandleLogOut导致的网络被关闭
			return
		}
		switch mes.Type {
		//这种情况代表服务器收到加入群聊申请，没有操作权限，
		//因此将消息转发给群管理员做决定
		case message.GroupManageMesType:
			err := UserPrcs.HandleGroupManageMes(&mes)
			if err != nil {
				fmt.Println(err)
			}
		case message.GroupManageResMesType:
			err := UserPrcs.HandleGroupManageResMes(&mes)
			if err != nil {
				fmt.Println(err)
			}
		case message.LoggedInOnAnotherDeviceType:
			UserPrcs.HandleLoggedInOnAnotherDevice(&mes)
			return
		case message.AddFriendResMesType:
			err = UserPrcs.HandleAddFriendResMes(&mes)
			if err != nil {
				fmt.Println(err)
			}
		case message.NotifyUserStatusMesType:
			var notifyMes message.NotifyUserStatusMes
			err = json.Unmarshal([]byte(mes.Data), &notifyMes)
			if err != nil {
				fmt.Println("反序列化服务器端返回的信息失败：", err)
				continue
			}
			FrdMgr.updateFriendStatus(notifyMes)
		case message.OneToOneMesType:
			var mesFromFriend message.OneToOneMes
			err = json.Unmarshal([]byte(mes.Data), &mesFromFriend)
			if err != nil {
				fmt.Println("反序列化服务器端返回的信息失败：", err)
				continue
			}
			FileRW.SaveOneToOneMes(&mesFromFriend)
			unreadMsgCountChan <- mesFromFriend.OriginId
			fmt.Printf("提示信息：%s发来了一条消息，", FrdMgr.GetAFamilierName(mesFromFriend.OriginId))
			cnt := FrdMgr.GetUnreadMesCount(mesFromFriend.OriginId)
			if cnt > 0 {
				fmt.Printf("有%d条未读消息", cnt)
			}
			fmt.Printf("\n")
		case message.AddFriendMesType:
			blockShowMenuChan <- struct{}{}
			err = UserPrcs.HandleAddFriendRequest(&mes)
			if err != nil {
				fmt.Println(err)
			}
			blockShowMenuChan <- struct{}{}
		case message.SendingOneToOneMesFailureNoticeType:
			var notice message.SendingOneToOneMesFailureNotice
			err = json.Unmarshal([]byte(mes.Data), &notice)
			if err != nil {
				fmt.Println("反序列化失败：", err)
			}
			fmt.Println(notice.Error)
		case message.GroupChatMesType:
			var GCMes message.GroupChatMes
			err := json.Unmarshal([]byte(mes.Data), &GCMes)
			if err != nil {
				fmt.Println("反序列化失败：", err)
			}
			select {
			case <-gCMesRealTimeDisplayChan:
				newGCMesChan <- TransferGCMes(&GCMes)
			default:
				unreadGCMsgCountChan <- GCMes.GroupChatId
				<-updateUnreadGCMesDone
				GC := GCMgr.GetGroupChatInfoByID(GCMes.GroupChatId)
				unreadMesCount, _ := GCMgr.GetGCUnreadMesCountByID(GCMes.GroupChatId)
				fmt.Printf("群聊%s有一条来自%s的新消息，还有未读消息%d条\n", GC.GroupName, GC.GroupMember[GCMes.OriginId].NickNameInGC, unreadMesCount)
			}
			FileRW.SaveGroupChatMes(&GCMes)
		default:
			fmt.Println("服务器端返回了一个未知类型的消息：", mes.Type)
		}
	}
}

func GCMesRealTimeDisplay() {

	for {
		select {
		case <-quitGroupChatChan:
			return
		case gCMesRealTimeDisplayChan <- struct{}{}:
			select {
			case newGCMesPtr := <-newGCMesChan:
				if newGCMesPtr != nil {
					newGCMesPtr.Visualize()
				}
			case <-quitGroupChatChan:
				<-gCMesRealTimeDisplayChan
				return
			}
		}
	}
}
