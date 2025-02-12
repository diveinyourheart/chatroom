package processes

import (
	"chatroom/common/message"
	"fmt"
	"sync"
)

type FriendMgr struct {
	friendStatusMap          map[int]*message.Friend
	friendsUnreadMesCountMap map[int]int
}

var (
	FrdMgr             *FriendMgr
	unreadMsgCountChan = make(chan int)
	readMsgCountChan   = make(chan int)
)

func init() {
	FrdMgr = &FriendMgr{
		friendStatusMap:          make(map[int]*message.Friend, 1024),
		friendsUnreadMesCountMap: make(map[int]int, 1024),
	}
}

func (this *FriendMgr) NewFriendMgr() {
	mu.Lock()
	defer mu.Unlock()
	this.friendStatusMap = make(map[int]*message.Friend, 1024)
	this.friendsUnreadMesCountMap = make(map[int]int, 1024)
}

// 帮助获得各用户未读信息数量的协程
func (this *FriendMgr) AcquireUnreadMesCount(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-logOutChan:
			return
		case ID := <-unreadMsgCountChan:
			mu.Lock()
			FrdMgr.friendsUnreadMesCountMap[ID]++
			mu.Unlock()
		case ID := <-readMsgCountChan:
			mu.Lock()
			FrdMgr.friendsUnreadMesCountMap[ID] = 0
			mu.Unlock()
		}
	}
}

func (this *FriendMgr) outputFriendMesList() {
	mu.Lock()
	defer mu.Unlock()
	for ID, count := range this.friendsUnreadMesCountMap {
		mu.Unlock()
		name := this.GetAFamilierName(ID)
		mu.Lock()
		fmt.Printf("ID:%d\tname:%s\t\t%d条未读消息\n", ID, name, count)
	}
}

func (this *FriendMgr) GetUnreadMesCount(ID int) int {
	mu.Lock()
	defer mu.Unlock()
	return this.friendsUnreadMesCountMap[ID]
}

func (this *FriendMgr) SetNoteNameById(ID int, noteName string) {
	mu.Lock()
	defer mu.Unlock()
	FrdMgr.friendStatusMap[ID].FriendNoteName = noteName
}

func (this *FriendMgr) GetNoteNameById(ID int) string {
	mu.Lock()
	mu.Unlock()
	return FrdMgr.friendStatusMap[ID].FriendNoteName
}

func (this *FriendMgr) GetFriendNameById(ID int) (name string, exist bool) {
	mu.Lock()
	defer mu.Unlock()
	frd, exist := this.friendStatusMap[ID]
	if exist {
		name = frd.FriendName
	}
	return
}

func (this *FriendMgr) AddNewFriendToMap(f *message.Friend) {
	mu.Lock()
	defer mu.Unlock()
	this.friendStatusMap[f.FriendId] = f
	this.friendsUnreadMesCountMap[f.FriendId] = 0
}

func (this *FriendMgr) updateFriendStatus(notifyMes message.NotifyUserStatusMes) {
	mu.Lock()
	this.friendStatusMap[notifyMes.UserId].FriendStatus = notifyMes.Status
	mu.Unlock()
	nm := this.GetAFamilierName(notifyMes.UserId)
	if notifyMes.Status == message.UserOnline {
		fmt.Printf("%s上线啦！！！\n", nm)
		this.outputFriendsList()
	} else if notifyMes.Status == message.UserOffline {
		fmt.Printf("%s已经下线\n", nm)
		this.outputFriendsList()
	}
}

func (this *FriendMgr) outputFriendsList() {
	fmt.Println("好友列表：")
	mu.Lock()
	for _, friend := range this.friendStatusMap {
		fmt.Printf("用户id:%d\t用户昵称:%s\t用户状态:%s", friend.FriendId, friend.FriendName, this.intStatus2StringStatus(friend.FriendStatus))
		if friend.FriendNoteName != "" {
			fmt.Printf("\t备注名:%s", friend.FriendNoteName)
		}
		fmt.Printf("\n")
	}
	mu.Unlock()
}

func (this *FriendMgr) GetAFamilierName(ID int) (name string) {
	name = this.GetNoteNameById(ID)
	if name == "" {
		name, _ = this.GetFriendNameById(ID)
	}
	return
}

func (this *FriendMgr) intStatus2StringStatus(status int) string {
	switch status {
	case 0:
		return "离线"
	case 1:
		return "在线"
	default:
		return "未知状态"
	}
}
