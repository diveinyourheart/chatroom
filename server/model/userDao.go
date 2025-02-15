package model

import (
	"chatroom/common/message"
	"chatroom/server/utils"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	DATABASE_USERS_NAME     string = "users"
	DATABASE_GROUPCHAT_NAME string = "groupChats"
	DATABASE_USER_MESSAGE   int    = 1 //选择索引为1的数据库存储离线用户接收到的消息
	DATABASE_GROUPCHAT      int    = 2
)

var (
	MyUserDao *UserDao
)

type UserDao struct {
	pool *redis.Pool
}

// 使用工厂模式，创建一个UserDao实例
func NewUserDao(pl *redis.Pool) (Ud *UserDao) {
	Ud = &UserDao{
		pool: pl,
	}
	return
}

func (UD *UserDao) AddNewManagerInGC(GCOwnerID int, userID int, GCID int) (err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	GC, err := UD.getGroupChatByID(conn, GCID)
	if err != nil {
		return fmt.Errorf("在数据库中获取群聊信息失败：%v", err)
	}
	if GCOwnerID != GC.GroupLeader {
		return fmt.Errorf("操作者不是该群的群主，无权限执行该操作")
	}
	_, exist := GC.GroupMember[userID]
	if !exist {
		return fmt.Errorf("被操作者不是该群的群成员")
	}
	count := message.MAX_GROUPMGR_NUMBER
	for _, v := range GC.GroupMgr {
		if v == 0 {
			count--
			continue
		}
		if v == userID {
			return fmt.Errorf("被操作者已经是该群的管理员了")
		}
	}
	if count == 5 {
		return fmt.Errorf("群管理员席位已满，每个群聊最多有%d个管理员", message.MAX_GROUPMGR_NUMBER)
	}
	for idx, v := range GC.GroupMgr {
		if v == 0 {
			GC.GroupMgr[idx] = userID
			break
		}
	}
	newInfo := message.UserInfoInGroupChat{
		Role:         message.GroupChatAdmin,
		NickNameInGC: GC.GroupMember[userID].NickNameInGC,
	}
	GC.GroupMember[userID] = newInfo
	err = UD.saveGroupChat(conn, GC)
	return
}

func (UD *UserDao) GetAllGroupChatsOfUserByID(userID int) (GCs []message.GroupChat, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	userInfo, err := UD.getUserById(conn, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户%d信息失败:%v", userID, err)
	}
	for GCID := range userInfo.UserGroupChats {
		GC, er := UD.getGroupChatByID(conn, GCID)
		if er != nil {
			utils.Log("为用户%d生成其加入群%d的信息错误：%v", userID, GCID, er)
		}
		GCs = append(GCs, *GC)
	}
	return
}

func (UD *UserDao) UpdateUserNewGC(userID int, GCID int, identification message.RoleInGroupChat) (err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	user, err := UD.getUserById(conn, userID)
	if err != nil {
		return fmt.Errorf("在数据库中获取用户信息失败：%v", err)
	}
	user.UserGroupChats[GCID] = identification
	err = UD.saveUser(conn, user)
	if err != nil {
		return fmt.Errorf("存入更新后的用户信息进入数据库失败：%v", err)
	}
	return
}

func (UD *UserDao) AddNewMemberToGC(usrID int, GCID int) (err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	GC, err := UD.getGroupChatByID(conn, GCID)
	if err != nil {
		return
	}
	user, er := UD.getUserById(conn, usrID)
	userName := ""
	if er == nil {
		userName = user.UserName
	}
	GC.GroupMember[usrID] = message.UserInfoInGroupChat{
		Role:         message.GroupChatMember,
		NickNameInGC: userName,
	}
	return UD.saveGroupChat(conn, GC)
}

func (UD *UserDao) CheckUserInSpecificGC(usrID int, GCID int) (exist bool, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	GC, err := UD.getGroupChatByID(conn, GCID)
	if err != nil {
		return
	}
	_, exist = GC.GroupMember[usrID]
	return
}

// 面向用户需求创建群聊
func (UD *UserDao) CreateGCAccording2Request(mes message.GroupManageMes) (newGroupChat *message.GroupChat, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	var randNumber int
	for {
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))
		randNumber = r.Intn(90000) + 10000
		er := UD.checkGCIDisExist(conn, randNumber)
		if er == redis.ErrNil {
			break
		}
	}
	user, er := UD.getUserById(conn, mes.OperandID)
	userName := ""
	if er == nil {
		userName = user.UserName
	}
	return UD.createGroupChatInDatabase(conn, randNumber, mes.OperandID, mes.ManageInfo, userName)
}

func (UD *UserDao) CheckGCIDisExist(ID int) (err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	err = UD.checkGCIDisExist(conn, ID)
	return
}

func (UD *UserDao) GetGroupChatByID(ID int) (GC *message.GroupChat, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	return UD.getGroupChatByID(conn, ID)
}

func (UD *UserDao) getGroupChatByID(conn redis.Conn, ID int) (GC *message.GroupChat, err error) {
	_, err = conn.Do("SELECT", DATABASE_GROUPCHAT)
	if err != nil {
		return nil, fmt.Errorf("选择数据库%d失败：%v", DATABASE_GROUPCHAT, err)
	}
	res, err := redis.String(conn.Do("HGET", DATABASE_GROUPCHAT_NAME, ID))
	_, er := conn.Do("SELECT", 0)
	if er != nil {
		utils.Log("选择数据库%d失败：%v", 0, er)
	}
	if err != nil {
		return nil, err
	}
	GC = &message.GroupChat{}
	err = json.Unmarshal([]byte(res), GC)
	if err != nil {
		err = fmt.Errorf("反序列化错误:%v", err)
	}
	return
}

func (UD *UserDao) checkGCIDisExist(conn redis.Conn, ID int) (err error) {
	_, err = conn.Do("SELECT", DATABASE_GROUPCHAT)
	if err != nil {
		return fmt.Errorf("选择数据库%d失败：%v", DATABASE_GROUPCHAT, err)
	}
	_, err = redis.String(conn.Do("HGET", DATABASE_GROUPCHAT_NAME, ID))
	_, er := conn.Do("SELECT", 0)
	if er != nil {
		utils.Log("选择数据库%d失败：%v", 0, er)
	}
	return
}

// 根据指定群ID，群主ID，群名在数据库2创建群聊
func (UD *UserDao) createGroupChatInDatabase(conn redis.Conn, GCID int, UserID int, GCName string, UserName string) (newGroupChat *message.GroupChat, err error) {
	if GCName == "" {
		GCName = fmt.Sprintf("%d的群聊%d", UserID, GCID)
	}
	newGroupChat = &message.GroupChat{
		GroupID:     GCID,
		GroupLeader: UserID,
		GroupName:   GCName,
		GroupMember: make(map[int]message.UserInfoInGroupChat, 512),
	}
	newGroupChat.GroupMgr[0] = UserID
	newGroupChat.GroupMember[UserID] = message.UserInfoInGroupChat{
		NickNameInGC: UserName,
		Role:         message.GroupChatOwner,
	}
	err = UD.saveGroupChat(conn, newGroupChat)
	return
}

func (UD *UserDao) saveGroupChat(conn redis.Conn, GC *message.GroupChat) (err error) {
	data, err := json.Marshal(*GC)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	_, err = conn.Do("SELECT", DATABASE_GROUPCHAT)
	if err != nil {
		return fmt.Errorf("选择数据库%d失败：%v", DATABASE_GROUPCHAT, err)
	}
	_, err = conn.Do("HSET", DATABASE_GROUPCHAT_NAME, GC.GroupID, string(data))
	_, er := conn.Do("SELECT", 0)
	if er != nil {
		utils.Log("选择数据库%d失败：%v", 0, er)
	}
	if err != nil {
		return fmt.Errorf("redis HSET命令将群信息存入数据库失败：%v", err)
	}
	return
}

func (UD *UserDao) FriendshipCheck(tx int, Rx int) (exist bool, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	user, err := UD.getUserById(conn, Rx)
	if err != nil {
		err = fmt.Errorf("数据库中不存在接收方%d", Rx)
		return
	}
	_, exist = user.UserFriends[tx]
	return
}

func (UD *UserDao) ConstructFriendById(userId int) (friendInfo *message.Friend, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	return UD.constructFriendById(conn, userId)
}

func (UD *UserDao) AddFriendForUser(requestor int, acceptor int) (friendInfo *message.Friend, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	usr, err := UD.getUserById(conn, requestor)
	if err != nil {
		return
	}
	if _, exist := usr.UserFriends[acceptor]; exist {
		return nil, fmt.Errorf("%d已经是%d的朋友了", acceptor, requestor)
	}
	usr.UserFriends[acceptor] = struct{}{}
	err = UD.saveUser(conn, usr)
	if err != nil {
		return nil, fmt.Errorf("数据库存入信息失败")
	}
	friendInfo, err = UD.constructFriendById(conn, acceptor)
	return
}

func (UD *UserDao) GetOfflineUserMessages(userId int) (messages [][]byte, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	_, err = conn.Do("SELECT", DATABASE_USER_MESSAGE)
	if err != nil {
		return nil, fmt.Errorf("选择数据库%d失败: %v", DATABASE_USER_MESSAGE, err)
	}

	for {
		data, err := redis.String(conn.Do("LPOP", userId))
		if err != nil {
			if err == redis.ErrNil {
				break
			}
			_, er := conn.Do("SELECT", 0)
			if er != nil {
				utils.Log("选择数据库%d失败: %v", 0, er)
			}
			return nil, fmt.Errorf("从数据库获取离线消息失败：%v", err)
		}
		mes := []byte(data)
		messages = append(messages, mes)
	}

	_, er := conn.Do("SELECT", 0)
	if er != nil {
		utils.Log("选择数据库%d失败: %v", 0, er)
	}

	return
}

func (UD *UserDao) StoreOfflineUserMessages(userId int, mes *message.Message) (err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	data, err := json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}

	_, err = conn.Do("SELECT", DATABASE_USER_MESSAGE)
	if err != nil {
		return fmt.Errorf("选择数据库%d失败: %v", DATABASE_USER_MESSAGE, err)
	}

	_, err = conn.Do("RPUSH", userId, string(data))
	_, er := conn.Do("SELECT", 0)
	if er != nil {
		utils.Log("选择数据库%d失败: %v", 0, er)
	}
	if err != nil {
		return fmt.Errorf("将离线用户接收消息推送到redis失败: %v", err)
	}

	return
}

func (UD *UserDao) ModifyUserStatusById(usrId int, status int) (err error) {
	conn := MyUserDao.pool.Get()
	defer conn.Close()
	usr, err := UD.getUserById(conn, usrId)
	if err != nil {
		return
	}
	usr.UserStatus = status
	err = UD.saveUser(conn, usr)
	return
}

func (UD *UserDao) saveUser(conn redis.Conn, user *message.User) (err error) {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	_, err = conn.Do("HSET", DATABASE_USERS_NAME, user.UserId, string(data))
	if err != nil {
		return fmt.Errorf("保存注册用户失败：%v", err)
	}
	return
}

func (UD *UserDao) getUserById(conn redis.Conn, id int) (user *message.User, err error) {
	res, err := redis.String(conn.Do("HGet", DATABASE_USERS_NAME, id))
	if err != nil {
		if err == redis.ErrNil {
			err = ERROR_USER_NOTEXISTS
		}
		return
	}
	user = &message.User{}
	err = json.Unmarshal([]byte(res), &user)
	if err != nil {
		err = fmt.Errorf("反序列化失败：%v", err)
	}
	return
}

func (UD *UserDao) GetUserById(id int) (user *message.User, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	return UD.getUserById(conn, id)
}

func (UD *UserDao) constructFriendById(conn redis.Conn, usrId int) (friend *message.Friend, err error) {
	usr, err := UD.getUserById(conn, usrId)
	if err != nil {
		return
	}
	friend = &message.Friend{}
	friend.FriendId = usr.UserId
	friend.FriendName = usr.UserName
	friend.FriendStatus = usr.UserStatus
	return
}

func (UD *UserDao) GetAllFriendsById(usrId int) (friends []message.Friend, err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	user, err := UD.getUserById(conn, usrId)
	if err != nil {
		return
	}
	for friendID := range user.UserFriends {
		friend, er := UD.constructFriendById(conn, friendID)
		if er != nil {
			utils.Log("为%d生成%d的好友信息失败：%v", usrId, friendID, er)
		} else {
			friends = append(friends, *friend)
		}
	}
	return
}

// 完成登陆的校验 Login
// 1. Login完成对用户的验证
func (UD *UserDao) Login(userId int, userPwd string) (user *message.User, err error) {
	conn := UD.pool.Get()
	user, err = UD.getUserById(conn, userId)
	conn.Close()
	if err != nil {
		return
	}
	if user.UserPwd != userPwd {
		err = ERROR_USER_PWD
		return
	}
	if user.UserStatus == message.UserOffline {
		err = UD.ModifyUserStatusById(userId, message.UserOnline)
		user.UserStatus = message.UserOnline
	}
	return
}

func (UD *UserDao) Register(usr *message.User) (err error) {
	conn := UD.pool.Get()
	defer conn.Close()
	_, err = UD.getUserById(conn, usr.UserId)
	if err != ERROR_USER_NOTEXISTS {
		if err == nil {
			err = ERROR_USER_EXISTS
		}
		return
	}
	err = UD.saveUser(conn, usr)
	return
}
