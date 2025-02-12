package processes

import (
	"chatroom/common/message"
	"chatroom/server/model"
	"chatroom/server/utils"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)

var (
	joinGroupChatApplyMap map[uint32]struct{}
)

func init() {
	joinGroupChatApplyMap = make(map[uint32]struct{}, 1024)
}

type joinGroupChatApply struct {
	Applier      int
	GroupChatID  int
	ApplyingTime time.Time
}

func (this joinGroupChatApply) Hash() uint32 {
	h := fnv.New32a()
	h.Write([]byte(strconv.Itoa(this.Applier)))
	h.Write([]byte(strconv.Itoa(this.GroupChatID)))
	h.Write([]byte(this.ApplyingTime.String()))
	return h.Sum32()
}

func (userP *UserProcess) ProcessGroupChatManageResMes(mes *message.Message) (err error) {
	var GCManageResMes message.GroupManageResMes
	err = json.Unmarshal([]byte(mes.Data), &GCManageResMes)
	if err != nil {
		return fmt.Errorf("反序列化失败，%v", err)
	}
	if GCManageResMes.ManageMesType != message.JOIN_GROUP_CHAT {
		return fmt.Errorf("服务器不应该接收到这一类消息,GCManageResMes.ManageMesType = %d", GCManageResMes.ManageMesType)
	}
	corrJoinApply := joinGroupChatApply{
		Applier:      GCManageResMes.OperandID,
		GroupChatID:  GCManageResMes.GroupChatID,
		ApplyingTime: GCManageResMes.JoinRequestTime,
	}
	_, exist := joinGroupChatApplyMap[corrJoinApply.Hash()]
	if !exist {
		utils.Log("此加群申请已经被其他管理员处理%v", corrJoinApply)
		return
	}
	delete(joinGroupChatApplyMap, corrJoinApply.Hash())
	if GCManageResMes.IsApproved {
		err = model.MyUserDao.AddNewMemberToGC(GCManageResMes.OperandID, GCManageResMes.GroupChatID)
		if err != nil {
			GCManageResMes.Code = 500
			GCManageResMes.Error = fmt.Sprintf("在数据库更新群%d的新成员%d的ID信息失败:%v", GCManageResMes.GroupChatID, GCManageResMes.OperandID, err)
			utils.Log("在数据库更新群%d的新成员%d的ID信息失败:%v", GCManageResMes.GroupChatID, GCManageResMes.OperandID, err)
		} else {
			GC, er := model.MyUserDao.GetGroupChatByID(GCManageResMes.GroupChatID)
			if er != nil {
				utils.Log("获取%d的群信息失败：%v", GCManageResMes.GroupChatID, er)
				GCManageResMes.Error = "加入群聊成功，但是返回的群信息为空"
			} else {
				GCManageResMes.NewUserInfoInGC = GC.GroupMember[GCManageResMes.OperandID]
				dt, er := json.Marshal(GCManageResMes)
				if er != nil {
					utils.Log("序列化失败导致生成用户%d的群内信息失败：%v", GCManageResMes.OperandID, err)
				} else {
					mes.Data = string(dt)
				}
				for member := range GC.GroupMember {
					if member == GCManageResMes.OperandID {
						continue
					}
					er := userP.PushMesToDesignatedUser(member, mes)
					if er != nil {
						utils.Log("向群%d的群成员%d推送新入群成员ID%d错误：%v", GCManageResMes.GroupChatID, member, GCManageResMes.OperandID, er)
					}
				}
				GCManageResMes.GroupChatInfo = *GC
			}
			GCManageResMes.Code = 200
			err = model.MyUserDao.UpdateUserNewGC(GCManageResMes.OperandID, GCManageResMes.GroupChatID, message.GroupChatMember)
			if err != nil {
				utils.Log("为用户%d更新新加入的群聊%d失败：%v", GCManageResMes.OperandID, GCManageResMes.GroupChatID, err)
			}
		}
	} else {
		GCManageResMes.Code = 500
		GCManageResMes.Error = "你的加群申请被管理员拒绝"
	}
	data, err := json.Marshal(GCManageResMes)
	if err != nil {
		utils.Log("序列化失败，返回入群申请结果时:%v", err)
	} else {
		mes.Data = string(data)
	}
	return userP.PushMesToDesignatedUser(GCManageResMes.OperandID, mes)
}

func (userP *UserProcess) ProcessGroupChatManageMes(mes *message.Message) (err error) {
	var GCManageMes message.GroupManageMes
	err = json.Unmarshal([]byte(mes.Data), &GCManageMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	GCManageResMes := message.GroupManageResMes{
		ManageMesType: GCManageMes.ManageMesType,
	}
	tf := &utils.Transfer{
		Conn: userP.Conn,
	}
	switch GCManageMes.ManageMesType {
	case message.ADD_ADMINISTRATOR:
		GCManageResMes.OperandID = GCManageMes.OperandID
		GCManageResMes.GroupChatID = GCManageMes.GroupChatID
		err = model.MyUserDao.AddNewManagerInGC(userP.UsrId, GCManageMes.OperandID, GCManageMes.GroupChatID)
		if err != nil {
			GCManageResMes.Code = 500
			GCManageResMes.Error = fmt.Sprintf("添加管理员失败:%v", err)
		} else {
			er := model.MyUserDao.UpdateUserNewGC(GCManageMes.OperandID, GCManageMes.GroupChatID, message.GroupChatAdmin)
			if er != nil {
				utils.Log("在数据库中添加管理员成功，但是在数据库中新管理员对应的用户的群身份更新失败:%v", er)
			}
			GCManageResMes.Code = 200
			GC, er := model.MyUserDao.GetGroupChatByID(GCManageMes.GroupChatID)
			if er != nil {
				utils.Log("通知群%d的群成员该群添加了新管理员%d失败：%v", GCManageMes.GroupChatID, GCManageMes.OperandID, er)
			} else {
				dt, er := json.Marshal(GCManageResMes)
				if er != nil {
					utils.Log("序列化失败导致通知群%d的群成员该群添加了新管理员%d失败：%v", GCManageMes.GroupChatID, GCManageMes.OperandID, er)
				} else {
					mes.Type = message.GroupManageResMesType
					mes.Data = string(dt)
					mark := true
					for member := range GC.GroupMember {
						er = userP.PushMesToDesignatedUser(member, mes)
						if er != nil {
							if member == GC.GroupLeader {
								mark = false
							}
							utils.Log("通知群%d的群成员%d该群添加了新管理员%d失败：%v", GCManageMes.GroupChatID, member, GCManageMes.OperandID, er)
						}
					}
					if mark {
						return
					}
				}
			}
		}
	case message.CREATE_A_GROUP_CHAT:
		newGroupChat, er := model.MyUserDao.CreateGCAccording2Request(GCManageMes)
		if er == nil {
			GCManageResMes.Code = 200
			GCManageResMes.GroupChatInfo = *newGroupChat
		} else {
			GCManageResMes.Code = 500
			GCManageResMes.Error = fmt.Sprintf("创建群聊失败：%v", err)
		}
		err = model.MyUserDao.UpdateUserNewGC(GCManageMes.OperandID, newGroupChat.GroupID, message.GroupChatOwner)
		if err != nil {
			utils.Log("为用户%d更新新的群聊信息%d失败：%v", GCManageMes.OperandID, newGroupChat.GroupID, err)
		}
	case message.JOIN_GROUP_CHAT:
		groupChatInfo, er := model.MyUserDao.GetGroupChatByID(GCManageMes.GroupChatID)
		if er == nil {
			newJoinApply := joinGroupChatApply{
				Applier:      GCManageMes.OperandID,
				GroupChatID:  GCManageMes.GroupChatID,
				ApplyingTime: GCManageMes.JoinRequestTime,
			}
			joinGroupChatApplyMap[newJoinApply.Hash()] = struct{}{}
			requestInfo, er := model.MyUserDao.ConstructFriendById(GCManageMes.OperandID)
			if er != nil {
				utils.Log("为%d群聊的管理员们生成群聊加入申请者%d的信息出错", GCManageMes.GroupChatID, GCManageMes.OperandID)
			} else {
				GCManageMes.OperandInfo = *requestInfo
			}
			data, er := json.Marshal(GCManageMes)
			if er != nil {
				utils.Log("为%d群聊的管理员们生成群聊加入申请者%d的信息出错", GCManageMes.GroupChatID, GCManageMes.OperandID)
			} else {
				mes.Data = string(data)
			}
			pushErrorNum, managerNum := 0, 0
			for _, manager := range groupChatInfo.GroupMgr {
				if manager == 0 {
					continue
				}
				managerNum++
				up, er := UsrMgr.GetOnlineUserByID(manager)
				if er != nil {
					er = model.MyUserDao.StoreOfflineUserMessages(manager, mes)
					if er != nil {
						utils.Log("将%d的加入群聊申请推送给离线群管理员%d失败：%v", GCManageMes.OperandID, manager, er)
						pushErrorNum++
					}
				} else {
					utils.Log("向在线用户%d发送来自%d的加入群聊申请", manager, GCManageMes.OperandID)
					data, er := json.Marshal(mes)
					if er != nil {
						utils.Log("序列化失败导致将%d的加入群聊申请推送给在线群管理员%d失败：%v", GCManageMes.OperandID, manager, er)
						pushErrorNum++
						continue
					}
					ntf := &utils.Transfer{
						Conn: up.Conn,
					}
					er = ntf.WritePkg(data)
					if er != nil {
						utils.Log("网络错误导致将%d的加入群聊申请推送给在线群管理员%d失败：%v", GCManageMes.OperandID, manager, er)
						pushErrorNum++
					}
				}
			}
			if pushErrorNum < managerNum {
				return
			}
			GCManageResMes.Code = 500
			GCManageResMes.Error = "向该群所有群管理员推送申请失败"
		} else {
			GCManageResMes.Code = 500
			if err == redis.ErrNil {
				GCManageResMes.Error = "尝试加入一个不存在的群聊"
			} else {
				GCManageResMes.Error = fmt.Sprintf("%v", err)
			}
		}
	default:
		return fmt.Errorf("服务器收到一个未知的群聊管理信息%d", GCManageMes.ManageMesType)
	}
	data, err := json.Marshal(GCManageResMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes = &message.Message{
		Type: message.GroupManageResMesType,
		Data: string(data),
	}
	data, err = json.Marshal(*mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	err = tf.WritePkg(data)
	return
}
