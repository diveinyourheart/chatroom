package processes

import "fmt"

var (
	UsrMgr *UserMgr
)

type UserMgr struct {
	onlineUsers      map[int]*UserProcess
	onlineSmsProcess map[int]*SmsProcess
}

func init() {
	UsrMgr = &UserMgr{
		onlineUsers:      make(map[int]*UserProcess, 1024),
		onlineSmsProcess: make(map[int]*SmsProcess, 1024),
	}
}

func NewUserMgr() (usrMgr *UserMgr) {
	usrMgr = &UserMgr{
		onlineUsers: make(map[int]*UserProcess, 1024),
	}
	return
}

func (this *UserMgr) GetOnlineUserSmsPrcsByUserPrcs(up *UserProcess) (smsPrcs *SmsProcess, err error) {
	if up == nil {
		return nil, fmt.Errorf("错误：试图为一个不在线的用户创建一个发送消息的线程")
	}
	smsPrcs, exist := this.onlineSmsProcess[up.UsrId]
	if !exist {
		smsPrcs = &SmsProcess{
			Conn:  up.Conn,
			UsrId: up.UsrId,
		}
	}
	return
}

func (this *UserMgr) AddOnlineUser(up *UserProcess) {
	this.onlineUsers[up.UsrId] = up
}

func (this *UserMgr) DelOnlineUser(usrId int) {
	_, exist := this.onlineUsers[usrId]
	if exist {
		delete(this.onlineUsers, usrId)
	}
	_, exist = this.onlineSmsProcess[usrId]
	if exist {
		delete(this.onlineSmsProcess, usrId)
	}
}

func (this *UserMgr) GetAllOnlineUser() map[int]*UserProcess {
	return this.onlineUsers
}

func (this *UserMgr) GetOnlineUserByID(usrId int) (up *UserProcess, err error) {
	up, ok := this.onlineUsers[usrId]
	if !ok {
		err = fmt.Errorf("用户%d当前不在线", usrId)
		return
	}
	return
}
