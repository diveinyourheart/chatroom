package processes

import (
	"chatroom/common/message"
	"chatroom/server/model"
	"chatroom/server/utils"
	"encoding/json"
	"fmt"
	"net"
)

type SmsProcess struct {
	Conn  net.Conn
	UsrId int
}

func (this *SmsProcess) ForwardGroupChatMes(mes *message.Message) (err error) {
	var GCMes message.GroupChatMes
	err = json.Unmarshal([]byte(mes.Data), &GCMes)
	if err != nil {
		return fmt.Errorf("反序列化失败：%v", err)
	}
	GCMes.TXorRX = message.RX__
	GCInfo, err := model.MyUserDao.GetGroupChatByID(GCMes.GroupChatId)
	if err != nil {
		return fmt.Errorf("在数据库获取群%d信息失败：%v", GCMes.GroupChatId, err)
	}
	for member := range GCInfo.GroupMember {
		if member == GCMes.OriginId {
			continue
		}
		up, err := UsrMgr.GetOnlineUserByID(member)
		if err != nil {
			data, er := json.Marshal(GCMes)
			if er != nil {
				utils.Log("序列化失败导致给群%d的群成员%d推送离线群消息失败：%v", GCInfo.GroupID, member, err)
				continue
			}
			mes.Data = string(data)
			er = model.MyUserDao.StoreOfflineUserMessages(member, mes)
			if er != nil {
				utils.Log("存储%d用户离线时接收到的群消息失败：%v", member, er)
			}
		} else {
			smsPrcs, _ := UsrMgr.GetOnlineUserSmsPrcsByUserPrcs(up)
			er := smsPrcs.SendGroupChatMes(GCMes)
			if er != nil {
				utils.Log("推送%d用户接收到的群消息失败：%v", member, er)
			}
		}
	}
	return
}

func (this *SmsProcess) SendGroupChatMes(GCMes message.GroupChatMes) (err error) {
	data, err := json.Marshal(GCMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes := message.Message{
		Type: message.GroupChatMesType,
		Data: string(data),
	}
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	tf := &utils.Transfer{
		Conn: this.Conn,
	}
	return tf.WritePkg(data)
}

func (this *SmsProcess) SendOneToOneMes(o2omes message.OneToOneMes) (err error) {
	data, err := json.Marshal(o2omes)
	if err != nil {
		return fmt.Errorf("反序列化失败:%v", err)
	}
	mes := message.Message{
		Type: message.OneToOneMesType,
		Data: string(data),
	}
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("反序列化失败:%v", err)
	}
	tf := &utils.Transfer{
		Conn: this.Conn,
	}
	err = tf.WritePkg(data)
	tf = nil
	return
}
