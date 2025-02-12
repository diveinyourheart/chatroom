package processes

import (
	"chatroom/client/model"
	"chatroom/client/utils"
	"chatroom/common/message"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type ChatMesType int

const (
	PRIVATE_MESSAGE_TYPE ChatMesType = iota
	GROUP_CHAT_TYPE
)

type ForStoringMes struct {
	SendingTime time.Time `json:"sendingTime"`
	SenderID    int       `json:"senderID"`
	SenderName  string    `json:"senderName"`
	Content     string    `json:"content"`
}

var FileRW *FileReadWrite
var batchSize int = 20

type FileReadWrite struct {
}

func init() {
	FileRW = &FileReadWrite{}
}

func (this ForStoringMes) Visualize() {
	fmt.Printf("发送时间：%v\t发送者：%s\n", this.SendingTime.Format("2006-01-02 15:04:05"), this.SenderName)
	fmt.Printf("\n")
	fmt.Printf("%s\n\n", this.Content)
	fmt.Printf("\n------------------------------------------------------------\n")
}

func TransferGCMes(mes *message.GroupChatMes) *ForStoringMes {
	newMes := &ForStoringMes{
		SendingTime: mes.SendingTime,
		SenderID:    mes.OriginId,
		Content:     mes.Content,
	}
	if mes.TXorRX == message.TX__ {
		newMes.SenderName = model.CurUsr.Usr.UserName
	} else {
		GC := GCMgr.GetGroupChatInfoByID(mes.GroupChatId)
		newMes.SenderName = GC.GroupMember[mes.OriginId].NickNameInGC
	}
	return newMes
}

func (this *FileReadWrite) outputGroupChatHistoryMes(GCID int) error {
	fmt.Printf("\n------------------------------------------------------------\n")
	record, err := this.ReadMess(0, GCID, GROUP_CHAT_TYPE)
	if err != nil {
		return fmt.Errorf("读取聊天消息记录文件错误:%v", err)
	}
	if record == nil {
		fmt.Println("无更早的聊天记录")
	}
	for i := len(record) - 1; i >= 0; i-- {
		record[i].Visualize()
	}
	return nil
}

func (this *FileReadWrite) GenerateFriendMesList(ID int) error {
	offset := 0
	cmd := "continue\n"
	fmt.Printf("\n------------------------------------------------------------\n")
	for cmd == "continue\n" {
		record, err := this.ReadMess(offset, ID, PRIVATE_MESSAGE_TYPE)
		if err != nil {
			return fmt.Errorf("读取聊天消息记录文件错误:%v", err)
		}
		if record == nil {
			fmt.Println("无更早的聊天记录")
			break
		}
		for i := len(record) - 1; i >= 0; i-- {
			record[i].Visualize()
		}
		fmt.Println("输入continue+回车键查看更早的聊天记录，此外输入任意键退出查看")
		cmd = utils.ReadStringInput() + "\n"
		offset += batchSize
	}
	return nil
}

func (this *FileReadWrite) generateUserMesSavingPath(oppositeID int, tp ChatMesType) string {
	var fileName string
	if tp == PRIVATE_MESSAGE_TYPE {
		fileName = fmt.Sprintf("../messageStorage/%d_%s/%d", model.CurUsr.Usr.UserId, model.CurUsr.Usr.UserName, oppositeID)
	} else if tp == GROUP_CHAT_TYPE {
		fileName = fmt.Sprintf("../GroupChatmessageStorage/%d_%s/%d", model.CurUsr.Usr.UserId, model.CurUsr.Usr.UserName, oppositeID)
	} else {
		return ""
	}
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := filepath.Join(currentDir, fileName)
	return filepath.Join(dir, "chatLog.json")
}

func (this *FileReadWrite) SaveGroupChatMes(mes *message.GroupChatMes) error {
	newMes := ForStoringMes{
		SendingTime: mes.SendingTime,
		SenderID:    mes.OriginId,
		Content:     mes.Content,
	}
	if mes.TXorRX == message.TX__ {
		newMes.SenderName = model.CurUsr.Usr.UserName
	} else {
		GC := GCMgr.GetGroupChatInfoByID(mes.GroupChatId)
		if GC == nil {
			return fmt.Errorf("获取群聊%d的信息时返回了一个空指针", mes.GroupChatId)
		}
		newMes.SenderName = GC.GroupMember[mes.OriginId].NickNameInGC
	}
	return this.saveMes(&newMes, mes.GroupChatId, GROUP_CHAT_TYPE)
}

func (this *FileReadWrite) SaveOneToOneMes(mes *message.OneToOneMes) error {
	newMes := ForStoringMes{
		SendingTime: mes.SendingTime,
		SenderID:    mes.OriginId,
		Content:     mes.Content,
	}
	if mes.TXorRX == message.TX__ {
		newMes.SenderName = model.CurUsr.Usr.UserName
		return this.saveMes(&newMes, mes.DesId, PRIVATE_MESSAGE_TYPE)
	} else {
		sdNm := FrdMgr.GetAFamilierName(mes.OriginId)
		newMes.SenderName = sdNm
		return this.saveMes(&newMes, mes.OriginId, PRIVATE_MESSAGE_TYPE)
	}
}

func (this *FileReadWrite) saveMes(newMes *ForStoringMes, oppositeID int, tp ChatMesType) (err error) {
	fileName := this.generateUserMesSavingPath(oppositeID, tp)
	if fileName == "" {
		return fmt.Errorf("生成信息存储路径错误")
	}
	dir := filepath.Dir(fileName)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("生成的信息存储路径不存在：%v", err)
	}

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("打开文件错误：%v", err)
	}
	defer file.Close()
	var messages []ForStoringMes
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&messages)
	if err != nil && err != io.EOF {
		return fmt.Errorf("json解码文件错误:%v", err)
	}
	newMessages := make([]ForStoringMes, len(messages)+1)
	newMessages[0] = *newMes
	copy(newMessages[1:], messages)
	file.Truncate(0)
	file.Seek(0, 0)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	return encoder.Encode(&newMessages)
}

func (this *FileReadWrite) ReadMess(offset int, oppositeID int, tp ChatMesType) (chatLog []ForStoringMes, err error) {
	fileName := this.generateUserMesSavingPath(oppositeID, tp)
	if fileName == "" {
		return nil, fmt.Errorf("生成信息存储路径错误")
	}
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("openning file error:%v", err)
	}
	defer file.Close()
	var messages []ForStoringMes
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&messages)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("json解码文件错误:%v", err)
	}
	sz := len(messages)
	if offset >= sz {
		return nil, nil
	} else if offset+batchSize > sz {
		return messages[offset:sz], nil
	} else {
		return messages[offset : offset+batchSize], nil
	}
}
