package processes

import (
	"chatroom/client/model"
	"chatroom/client/utils"
	"chatroom/common/message"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

var UserPrcs *UserProcess

type UserProcess struct {
}

func init() {
	UserPrcs = &UserProcess{}
}

// 实现群主添加管理员
func (up *UserProcess) GCOwnerAddGCManager(GCInfo *message.GroupChat) (err error) {
	if model.CurUsr.Usr.UserId != GCInfo.GroupLeader {
		return fmt.Errorf("当前登录用户不是群%d的群主，不能为群添加管理员", GCInfo.GroupID)
	}
	GCMgr.OutputGCMembers(GCInfo)
	fmt.Println("请输入你想添加为管理员的用户的ID")
	targetID := utils.ReadIntInput()
	var isValid bool = false
	for !isValid {
		_, exist := GCInfo.GroupMember[targetID]
		if !exist {
			fmt.Printf("%d不是该群的群成员，请重新输入", targetID)
			targetID = utils.ReadIntInput()
			continue
		}
		for _, v := range GCInfo.GroupMgr {
			if v == 0 {
				continue
			}
			if v == targetID {
				return fmt.Errorf("%d已经是该群的管理员了", targetID)
			}
		}
		isValid = true
	}
	GCManageMes := message.GroupManageMes{
		ManageMesType: message.ADD_ADMINISTRATOR,
		OperandID:     targetID,
		GroupChatID:   GCInfo.GroupID,
	}
	data, err := json.Marshal(GCManageMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes := message.Message{
		Type: message.GroupManageMesType,
		Data: string(data),
	}
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	tf := &utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	return
}

func (up *UserProcess) HandleGroupManageMes(mes *message.Message) (err error) {
	var GCManageMes message.GroupManageMes
	err = json.Unmarshal([]byte(mes.Data), &GCManageMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	if GCManageMes.ManageMesType != message.JOIN_GROUP_CHAT {
		return fmt.Errorf("客户端不该收到这种信息类型：GCManageMes.ManageMesType =%d", GCManageMes.ManageMesType)
	}
	requestorInfo := GCManageMes.OperandInfo
	GCName, _ := GCMgr.GetGCNameById(GCManageMes.GroupChatID)
	fmt.Printf("ID为%d,昵称为%s申请加入%s群聊,是否同意？输入0代表拒绝，输入1代表同意\n", requestorInfo.FriendId, requestorInfo.FriendName, GCName)
	sign := utils.ReadIntInput()
	GCManageResMes := message.GroupManageResMes{
		ManageMesType:   message.JOIN_GROUP_CHAT,
		OperandID:       GCManageMes.OperandID,
		GroupChatID:     GCManageMes.GroupChatID,
		DecidedBy:       model.CurUsr.Usr.UserId,
		JoinRequestTime: GCManageMes.JoinRequestTime,
	}
	var isValid bool = false
	for !isValid {
		switch sign {
		case 0:
			GCManageResMes.IsApproved = false
			fmt.Println("你拒绝了群聊加入申请")
			isValid = true
		case 1:
			GCManageResMes.IsApproved = true
			fmt.Println("你同意了群聊加入申请")
			isValid = true
		default:
			fmt.Println("输入错误，请重新输入")
			sign = utils.ReadIntInput()
		}
	}
	data, err := json.Marshal(GCManageResMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes.Type = message.GroupManageResMesType
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	tf := utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	return
}

func (up *UserProcess) Apply2JoinAGC(GCID int, note string) (err error) {
	applyMes := message.GroupManageMes{
		ManageMesType:   message.JOIN_GROUP_CHAT,
		ManageInfo:      note,
		OperandID:       model.CurUsr.Usr.UserId,
		GroupChatID:     GCID,
		JoinRequestTime: time.Now(),
	}
	data, err := json.Marshal(applyMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes := message.Message{
		Type: message.GroupManageMesType,
		Data: string(data),
	}
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	tf := &utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	return
}

func (up *UserProcess) HandleGroupManageResMes(mes *message.Message) (err error) {
	var GCManageResMes message.GroupManageResMes
	err = json.Unmarshal([]byte(mes.Data), &GCManageResMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	switch GCManageResMes.ManageMesType {
	case message.CREATE_A_GROUP_CHAT:
		if GCManageResMes.Code == 200 {
			GCMgr.AddGroupChatToMap(&GCManageResMes.GroupChatInfo)
			fmt.Println("你申请创建的群聊创建成功，群聊的ID为：", GCManageResMes.GroupChatInfo.GroupID)
		} else {
			err = fmt.Errorf("你申请创建群聊失败：%s", GCManageResMes.Error)
		}
	case message.ADD_ADMINISTRATOR:
		GCName, _ := GCMgr.GetGCNameById(GCManageResMes.GroupChatID)
		if GCManageResMes.Code == 200 {
			GCMgr.ModifyGCMemberRole(GCManageResMes.GroupChatID, GCManageResMes.OperandID, message.GroupChatAdmin)
			if GCMgr.GetGCLeader(GCManageResMes.GroupChatID) == model.CurUsr.Usr.UserId {
				fmt.Printf("添加%d为群%s管理员成功\n", GCManageResMes.OperandID, GCName)
			}
			if model.CurUsr.Usr.UserId == GCManageResMes.OperandID {
				fmt.Printf("你已被添加为%s的管理员\n", GCName)
			}
		} else {
			if GCMgr.GetGCLeader(GCManageResMes.GroupChatID) == model.CurUsr.Usr.UserId {
				fmt.Printf("添加%d为群%s管理员失败,%s\n", GCManageResMes.OperandID, GCName, GCManageResMes.Error)
			}
		}
	case message.JOIN_GROUP_CHAT:
		if GCManageResMes.OperandID == model.CurUsr.Usr.UserId {
			if GCManageResMes.Code == 200 {
				if GCManageResMes.Error == "" {
					GCMgr.AddGroupChatToMap(&GCManageResMes.GroupChatInfo)
				} else {
					fmt.Println(GCManageResMes.Error)
				}
				fmt.Printf("加入群聊%d成功\n", GCManageResMes.GroupChatID)
			} else {
				fmt.Printf("你加入%d群聊的申请失败：%s\n", GCManageResMes.OperandID, GCManageResMes.Error)
			}
		} else {
			if GCManageResMes.IsApproved {
				GCMgr.AddNewMember2GC(GCManageResMes.OperandID, GCManageResMes.GroupChatID, GCManageResMes.NewUserInfoInGC)
			}
		}
	default:
		return fmt.Errorf("服务器返回了一个未知的群管理结果消息:%d", GCManageResMes.ManageMesType)
	}
	return
}

func (up *UserProcess) CreateGroupChat(GCName string) (err error) {
	GCmanageMes := message.GroupManageMes{
		ManageMesType: message.CREATE_A_GROUP_CHAT,
		OperandID:     model.CurUsr.Usr.UserId,
		ManageInfo:    GCName,
	}
	data, err := json.Marshal(GCmanageMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	mes := message.Message{
		Type: message.GroupManageMesType,
		Data: string(data),
	}
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	tf := &utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	return
}

func (up *UserProcess) HandleLoggedInOnAnotherDevice(mes *message.Message) {
	var notification message.LoggedInOnAnotherDevice
	err := json.Unmarshal([]byte(mes.Data), &notification)
	if err != nil {
		fmt.Println("你的账号已经在未知地点登录...")
	} else {
		fmt.Println("你的账号于", notification.LoginTime.Format("2006-01-02 15:04:05"), "在另外一个设备登录，设备信息如下：")
		fmt.Println("操作系统：", notification.OperatingSystem)
		fmt.Println("主机名：", notification.HostName)
	}
	HandleLogOut()
}

func (up *UserProcess) HandleAddFriendResMes(mes *message.Message) (err error) {
	var addFriendResMes message.AddFriendResMes
	err = json.Unmarshal([]byte(mes.Data), &addFriendResMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	if addFriendResMes.IsAgree {
		newFriend := &addFriendResMes.FriendInfo
		FrdMgr.AddNewFriendToMap(newFriend)
		fmt.Printf("ID为%d，昵称为%s的用户同意了你的好友申请\n", newFriend.FriendId, newFriend.FriendName)
		fmt.Println("为其添加备注(按回车键则跳过)：")
		noteName := utils.ReadStringInput()
		if noteName != "" {
			FrdMgr.SetNoteNameById(newFriend.FriendId, noteName)
		}
		FrdMgr.outputFriendsList()
	} else {
		err = fmt.Errorf("ID为%d，昵称为%s的用户拒绝了你的好友申请", addFriendResMes.FriendInfo.FriendId, addFriendResMes.FriendInfo.FriendName)
	}
	return
}

func (up *UserProcess) HandleAddFriendRequest(mes *message.Message) (err error) {
	var addFriendMes message.AddFriendMes
	err = json.Unmarshal([]byte(mes.Data), &addFriendMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	fmt.Printf("ID为%d,昵称为%s向你发来一个好友申请，并留言到\n%s\n", addFriendMes.Requester.FriendId, addFriendMes.Requester.FriendName, addFriendMes.Note)
	fmt.Println("请选择是否同意好友申请，输入0代表不同意，输入1代表同意")
	sign := utils.ReadIntInput()
	var addFriendResMes message.AddFriendResMes
	addFriendResMes.TargetUserID = addFriendMes.Requester.FriendId
	var isValid bool = false
	for !isValid {
		switch sign {
		case 0:
			addFriendResMes.IsAgree = false
			isValid = true
			fmt.Println("你拒绝了该好友添加申请")
		case 1:
			addFriendResMes.IsAgree = true
			isValid = true
			newFriend := &message.Friend{
				FriendId:     addFriendMes.Requester.FriendId,
				FriendName:   addFriendMes.Requester.FriendName,
				FriendStatus: addFriendMes.Requester.FriendStatus,
			}
			FrdMgr.AddNewFriendToMap(newFriend)
			fmt.Println("为其添加备注(按回车键则跳过)：")
			noteName := utils.ReadStringInput()
			if noteName != "" {
				FrdMgr.SetNoteNameById(newFriend.FriendId, noteName)
			}
			FrdMgr.outputFriendsList()
		default:
			fmt.Println("输入错误，请重新输入")
			sign = utils.ReadIntInput()
		}
	}
	data, err := json.Marshal(addFriendResMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes.Type = message.AddFriendResMesType
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	tf := &utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	return
}

func (up *UserProcess) sendAddFriendRequest(userId int, note string) (err error) {
	tf := utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	addFriendMes := message.AddFriendMes{
		Note:         note,
		TargetUserID: userId,
	}
	data, err := json.Marshal(addFriendMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes := message.Message{
		Type: message.AddFriendMesType,
		Data: string(data),
	}
	data, err = json.Marshal(mes)
	err = tf.WritePkg([]byte(data))
	return
}

func (up *UserProcess) Register(userId int, userPwd string, userName string) (err error) {
	conn, err := net.Dial("tcp", SERVER_IPv4_ADDRESS)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %v", err)
	}
	defer conn.Close()
	tf := &utils.Transfer{
		Conn: conn,
	}
	rgstMes := message.RegisterMes{
		Usr: message.User{
			UserId:         userId,
			UserPwd:        userPwd,
			UserName:       userName,
			UserFriends:    make(map[int]struct{}, 256),
			UserGroupChats: make(map[int]message.RoleInGroupChat, 256),
		},
	}
	mes := message.Message{
		Type: message.RegisterMesType,
	}
	data, err := json.Marshal(rgstMes)
	if err != nil {
		return fmt.Errorf("序列化注册消息失败: %v", err)
	}
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}
	err = tf.WritePkg(data)
	if err != nil {
		return err
	}
	RgstResMes := message.RegisterResMes{}
	mes, err = tf.ReadPkg()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(mes.Data), &RgstResMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	if RgstResMes.Code != 200 {
		return fmt.Errorf("注册失败：%v", RgstResMes.Error)
	}
	tf = nil
	return nil
}

// 完成登录功能
func (up *UserProcess) Login(userId int, userPwd string) (conn net.Conn, er error) {
	// 1. 连接到服务器
	conn, err := net.Dial("tcp", SERVER_IPv4_ADDRESS)
	if err != nil {
		er = fmt.Errorf("连接服务器失败: %v", err)
		return
	}

	tf := &utils.Transfer{
		Conn: conn,
	}
	// 2. 准备发送的消息
	mes := message.Message{
		Type: message.LoginMesType,
	}
	loginMes := message.LoginMes{
		UserId:  userId,
		UserPwd: userPwd,
	}
	// 序列化 LoginMes
	data, err := json.Marshal(loginMes)
	if err != nil {
		er = fmt.Errorf("序列化登录消息失败: %v", err)
		return
	}
	mes.Data = string(data)

	// 序列化 Message
	data, err = json.Marshal(mes)
	if err != nil {
		er = fmt.Errorf("序列化消息失败: %v", err)
		return
	}

	er = tf.WritePkg(data)
	if er != nil {
		return
	}
	mes, er = tf.ReadPkg()
	if er != nil {
		return
	}
	var loginResMes message.LoginResMes
	err = json.Unmarshal([]byte(mes.Data), &loginResMes)
	if err != nil {
		er = fmt.Errorf("反序列化失败：%v", err)
		return
	}
	if loginResMes.Code != 200 {
		er = fmt.Errorf("登陆失败：%v", loginResMes.Error)
		return
	}
	model.CurUsr.Usr = loginResMes.Usr
	for _, v := range loginResMes.Friends {
		if v.FriendId == model.CurUsr.Usr.UserId {
			continue
		}
		F := &message.Friend{
			FriendId:     v.FriendId,
			FriendName:   v.FriendName,
			FriendStatus: v.FriendStatus,
		}
		FrdMgr.AddNewFriendToMap(F)
	}
	for _, v := range loginResMes.GroupChats {
		GC := &message.GroupChat{
			GroupID:     v.GroupID,
			GroupName:   v.GroupName,
			GroupLeader: v.GroupLeader,
			GroupMgr:    v.GroupMgr,
			GroupMember: v.GroupMember,
		}
		GCMgr.AddGroupChatToMap(GC)
	}
	return
}
