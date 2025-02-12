package processes

import (
	"chatroom/client/model"
	"chatroom/client/utils"
	"chatroom/common/message"
	"encoding/json"
	"fmt"
	"time"
)

var SmsPrcs *SmsProcess

type SmsProcess struct {
}

func init() {
	SmsPrcs = &SmsProcess{}
}

func (smsP *SmsProcess) SendGroupChatMes(userID int, GC *message.GroupChat, content string) (err error) {
	GCMes := message.GroupChatMes{
		SendingTime: time.Now(),
		OriginId:    userID,
		GroupChatId: GC.GroupID,
		Content:     content,
		TXorRX:      message.TX__,
	}
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
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	if err != nil {
		fmt.Printf("\n------------------------------------------------------------\n")
		TransferGCMes(&GCMes).Visualize()
		FileRW.SaveGroupChatMes(&GCMes)
	}
	return
}

func (smsP *SmsProcess) sendOneToOneMesById(id int, content string) (err error) {
	mes := message.Message{
		Type: message.OneToOneMesType,
	}
	oneToOneMes := message.OneToOneMes{
		DesId:       id,
		Content:     content,
		OriginId:    model.CurUsr.Usr.UserId,
		SendingTime: time.Now(),
		TXorRX:      message.TX__,
	}
	data, err := json.Marshal(oneToOneMes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		return fmt.Errorf("序列化失败：%v", err)
	}
	tf := &utils.Transfer{
		Conn: model.CurUsr.Conn,
	}
	err = tf.WritePkg(data)
	if err == nil {
		FileRW.SaveOneToOneMes(&oneToOneMes)
	}
	return
}
