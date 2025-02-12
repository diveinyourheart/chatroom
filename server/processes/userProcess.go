package processes

import (
	"chatroom/common/message"
	"chatroom/server/model"
	"chatroom/server/utils"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"
)

type UserProcess struct {
	Conn  net.Conn
	UsrId int
}

func (userP *UserProcess) PushMesToDesignatedUser(ID int, mes *message.Message) (err error) {
	up, err := UsrMgr.GetOnlineUserByID(ID)
	if err != nil {
		err = model.MyUserDao.StoreOfflineUserMessages(ID, mes)
		if err != nil {
			utils.Log("存储离线用户%d接收到的消息失败：%v", ID, err)
		}
	} else {
		data, er := json.Marshal(mes)
		if er != nil {
			return fmt.Errorf("序列化失败：%v", er)
		}
		tf := &utils.Transfer{
			Conn: up.Conn,
		}
		err = tf.WritePkg(data)
	}
	return
}

func (userP *UserProcess) ForwardAddFriendResMes(mes *message.Message) (err error) {
	var addFriendResMes message.AddFriendResMes
	err = json.Unmarshal([]byte(mes.Data), &addFriendResMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	targetUserId, originUserId := addFriendResMes.TargetUserID, userP.UsrId
	if addFriendResMes.IsAgree {
		friendInfo, err := model.MyUserDao.AddFriendForUser(targetUserId, originUserId)
		if err != nil {
			utils.Log("在数据库中为%d添加好友%d失败：%v", targetUserId, originUserId, err)
		} else {
			addFriendResMes.FriendInfo = *friendInfo
		}
		_, err = model.MyUserDao.AddFriendForUser(originUserId, targetUserId)
		if err != nil {
			utils.Log("在数据库中为%d添加好友%d失败：%v", originUserId, targetUserId, err)
		}
	} else {
		friendInfo, err := model.MyUserDao.ConstructFriendById(originUserId)
		if err != nil {
			utils.Log("为%d生成%d的好友信息失败：%v", targetUserId, originUserId, err)
		} else {
			addFriendResMes.FriendInfo = *friendInfo
		}
	}
	data, err := json.Marshal(addFriendResMes)
	if err != nil {
		utils.Log("为%d生成%d的好友信息失败：%v", targetUserId, originUserId, err)
	} else {
		mes.Data = string(data)
	}
	return userP.PushMesToDesignatedUser(targetUserId, mes)
}

func (userP *UserProcess) ForwardAddFriendRequestMes(mes *message.Message) (err error) {
	var addFriendMes message.AddFriendMes
	err = json.Unmarshal([]byte(mes.Data), &addFriendMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	targetUserId, requestorId := addFriendMes.TargetUserID, userP.UsrId
	friendInfo, err := model.MyUserDao.ConstructFriendById(requestorId)
	if err != nil {
		utils.Log("生成用户%d的朋友信息给%d失败：%v", requestorId, targetUserId, err)
	} else {
		addFriendMes.Requester = *friendInfo
	}
	data, err := json.Marshal(addFriendMes)
	if err != nil {
		utils.Log("生成用户%d的朋友信息给%d失败：%v", requestorId, targetUserId, err)
	} else {
		mes.Data = string(data)
	}
	return userP.PushMesToDesignatedUser(targetUserId, mes)
}

// 用户刚登陆时，需要将该用户在离线时接收到的消息全部推送给该用户
func (userP *UserProcess) PushOfflineUserMessages() (err error) {
	messages, err := model.MyUserDao.GetOfflineUserMessages(userP.UsrId)
	if err != nil {
		return fmt.Errorf("获取用户%d的离线消息失败：%v", userP.UsrId, err)
	}
	tf := &utils.Transfer{
		Conn: userP.Conn,
	}
	count := 0
	for _, mes := range messages {
		err = tf.WritePkg(mes)
		if err != nil {
			count++
			utils.Log("向%d推送一条离线消息失败：%v", userP.UsrId, err)
		}
	}
	if count == len(messages) {
		return fmt.Errorf("向%d推送离线消息失败：%v", userP.UsrId, err)
	}
	return
}

func (userP *UserProcess) SendOneToOneMes(mes *message.Message) (err error) {
	var oneToOneMes message.OneToOneMes
	err = json.Unmarshal([]byte(mes.Data), &oneToOneMes)
	if err != nil {
		err = fmt.Errorf("反序列化失败：%v", err)
		return
	}
	oneToOneMes.TXorRX = message.RX__
	originId, desId := oneToOneMes.OriginId, oneToOneMes.DesId
	exist, err := model.MyUserDao.FriendshipCheck(originId, desId)
	if !exist {
		notice := message.SendingOneToOneMesFailureNotice{
			DesId: desId,
		}
		if err != nil {
			notice.Error = fmt.Sprintf("%v", err)
		} else {
			notice.Error = fmt.Sprintf("%d不是你的好友，发送消息失败", desId)
			utils.Log("%d不是%d的好友，信息发送失败", desId, originId)
		}
		data, er := json.Marshal(notice)
		if er != nil {
			err = fmt.Errorf("序列化失败：%v", er)
			return
		}
		mes.Type = message.SendingOneToOneMesFailureNoticeType
		mes.Data = string(data)
		data, er = json.Marshal(mes)
		if er != nil {
			err = fmt.Errorf("序列化失败：%v", er)
			return
		}
		tf := &utils.Transfer{
			Conn: userP.Conn,
		}
		err = tf.WritePkg(data)
		return
	}
	up, er := UsrMgr.GetOnlineUserByID(desId)
	if er != nil {
		data, er := json.Marshal(oneToOneMes)
		if er != nil {
			err = fmt.Errorf("序列化失败导致保存用户%d的离线接收信息失败：%v", oneToOneMes.DesId, er)
			return
		}
		mes.Data = string(data)
		err = model.MyUserDao.StoreOfflineUserMessages(desId, mes)
		if err != nil {
			utils.Log("存储离线用户%d接收到的消息失败：%v", desId, err)
		}
	} else {
		utils.Log("向%d发送来自%d的信息", desId, originId)
		smsPrcs, _ := UsrMgr.GetOnlineUserSmsPrcsByUserPrcs(up)
		err = smsPrcs.SendOneToOneMes(oneToOneMes)
	}
	return
}

func (userP *UserProcess) HandleForOffline() {
	userP.NotifyOtherOnlineFriends(message.UserOffline)
	UsrMgr.DelOnlineUser(userP.UsrId)
	model.MyUserDao.ModifyUserStatusById(userP.UsrId, message.UserOffline)
}

// 告诉其他在线的人我的状态
func (userP *UserProcess) NotifyOtherOnlineFriends(status int) (err error) {
	user, err := model.MyUserDao.GetUserById(userP.UsrId)
	if err != nil {
		return
	}
	for friendID := range user.UserFriends {
		_, er := UsrMgr.GetOnlineUserByID(friendID)
		if er == nil {
			er = userP.NotifyMyStatus(friendID, status)
			if er != nil {
				utils.Log("通知%d的好友%d状态更改为%d失败", userP.UsrId, friendID, status)
			}
		}
	}
	return
}

func (userP *UserProcess) NotifyMyStatus(DesUserId int, status int) (err error) {
	mes := message.Message{
		Type: message.NotifyUserStatusMesType,
	}
	notifyMes := message.NotifyUserStatusMes{
		UserId: userP.UsrId,
		Status: status,
	}
	data, err := json.Marshal(notifyMes)
	if err != nil {
		err = fmt.Errorf("序列化失败：%v", err)
		return
	}
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		err = fmt.Errorf("序列化失败：%v", err)
		return
	}
	desUserPrcs, _ := UsrMgr.GetOnlineUserByID(DesUserId)
	tf := &utils.Transfer{
		Conn: desUserPrcs.Conn,
	}
	err = tf.WritePkg(data)
	if err != nil {
		err = fmt.Errorf("向客户端%v推送%v在线状态信息失败：%v", desUserPrcs.Conn.RemoteAddr().String(), userP.UsrId, err)
		return
	}
	return
}

func (userP *UserProcess) ServerProcessRegister(mes *message.Message) (myErr error) {
	var rgstMes message.RegisterMes
	err := json.Unmarshal([]byte(mes.Data), &rgstMes)
	if err != nil {
		myErr = fmt.Errorf("反序列化失败：%v", err)
		return
	}
	err = model.MyUserDao.Register(&rgstMes.Usr)
	var resMes message.Message
	resMes.Type = message.RegisterResMesType
	var RegisterResMes message.RegisterResMes
	if err != nil {
		RegisterResMes.Code = 400
		if err == model.ERROR_USER_EXISTS {
			RegisterResMes.Error = "该用户ID已存在"
		} else {
			RegisterResMes.Error = "请重新尝试"
		}
	} else {
		RegisterResMes.Code = 200
		fmt.Printf("%v注册成功\n", rgstMes.Usr)
	}
	data, err := json.Marshal(RegisterResMes)
	if err != nil {
		myErr = fmt.Errorf("序列化失败：%v", err)
		return
	}
	resMes.Data = string(data)
	data, err = json.Marshal(resMes)
	if err != nil {
		myErr = fmt.Errorf("序列化失败：%v", err)
		return
	}
	tf := &utils.Transfer{
		Conn: userP.Conn,
	}
	myErr = tf.WritePkg(data)
	return
}

func (userP *UserProcess) ServerProcessLogin(mes *message.Message) (myErr error) {
	var loginMes message.LoginMes
	err := json.Unmarshal([]byte(mes.Data), &loginMes)
	if err != nil {
		myErr = fmt.Errorf("反序列化失败：%v", err)
		return
	}
	var resMes message.Message
	resMes.Type = message.LoginResMesType
	var loginResMes message.LoginResMes
	user, err := model.MyUserDao.Login(loginMes.UserId, loginMes.UserPwd)
	if err != nil {
		if err == model.ERROR_USER_NOTEXISTS {
			loginResMes.Code = 500
			loginResMes.Error = "用户不存在"
		} else if err == model.ERROR_USER_PWD {
			loginResMes.Code = 500
			loginResMes.Error = "密码错误"
		} else {
			loginResMes.Code = 500
			loginResMes.Error = "出现未知错误"
		}
		myErr = model.ERROR_LOGIN_FAILURE
	} else {
		if up, err := UsrMgr.GetOnlineUserByID(loginMes.UserId); err == nil {
			hostname, err := os.Hostname()
			if err != nil {
				hostname = fmt.Sprintf("Error getting hostname:%v", err)
			}
			RemoteLoginNotification := message.LoggedInOnAnotherDevice{
				LoginTime:       time.Now(),
				OperatingSystem: runtime.GOOS,
				HostName:        hostname,
			}
			mes := message.Message{
				Type: message.LoggedInOnAnotherDeviceType,
			}
			temp, err := json.Marshal(RemoteLoginNotification)
			if err != nil {
				return fmt.Errorf("序列化失败导致%d登录失败，并且这一账号已在另一端登录", loginMes.UserId)
			}
			mes.Data = string(temp)
			temp, err = json.Marshal(mes)
			if err != nil {
				return fmt.Errorf("序列化失败导致%d登录失败，并且这一账号已在另一端登录", loginMes.UserId)
			}
			tsf := &utils.Transfer{
				Conn: up.Conn,
			}
			err = tsf.WritePkg(temp)
			if err != nil {
				return fmt.Errorf("发送包失败导致%d登录失败，并且这一账号已在另一端登录", loginMes.UserId)
			}
		}
		userP.UsrId = user.UserId
		loginResMes.Code = 200
		loginResMes.Usr = *user
		loginResMes.Friends, err = model.MyUserDao.GetAllFriendsById(user.UserId)
		if err != nil {
			loginResMes.Error = "登录成功但是获取好友列表失败."
		}
		loginResMes.GroupChats, err = model.MyUserDao.GetAllGroupChatsOfUserByID(user.UserId)
		if err != nil {
			loginResMes.Error += "登陆成功但是获取群聊列表失败"
		}
		fmt.Printf("%v登陆成功\n", user)
		UsrMgr.AddOnlineUser(userP)
	}
	data, err := json.Marshal(loginResMes)
	if err != nil {
		myErr = fmt.Errorf("序列化失败：%v", err)
		return
	}
	resMes.Data = string(data)
	data, err = json.Marshal(resMes)
	if err != nil {
		myErr = fmt.Errorf("序列化失败：%v", err)
		return
	}
	tf := &utils.Transfer{
		Conn: userP.Conn,
	}
	myErr = tf.WritePkg(data)
	return
}
