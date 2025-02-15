package processes

import (
	"chatroom/common/message"
	"fmt"
	"sync"
)

type GroupChatMgr struct {
	groupChatMap                map[int]*message.GroupChat
	groupChatsUnreadMesCountMap map[int]int
}

var (
	GCMgr                *GroupChatMgr
	readGCMsgCountChan   = make(chan int)
	unreadGCMsgCountChan = make(chan int)
)

func init() {
	GCMgr = &GroupChatMgr{
		groupChatMap:                make(map[int]*message.GroupChat, 1024),
		groupChatsUnreadMesCountMap: make(map[int]int, 1024),
	}
}

func (this *GroupChatMgr) ModifyGCMemberRole(GCID int, userID int, newRole message.RoleInGroupChat) {
	mu.Lock()
	defer mu.Unlock()
	member := this.groupChatMap[GCID].GroupMember[userID]
	member.Role = newRole
	this.groupChatMap[GCID].GroupMember[userID] = member
}

func (this *GroupChatMgr) GetGCLeader(GCID int) int {
	mu.Lock()
	defer mu.Unlock()
	return this.groupChatMap[GCID].GroupLeader
}

func (this *GroupChatMgr) OutputGCMembersByID(ID int) (err error) {
	mu.Lock()
	defer mu.Unlock()
	GC, exist := this.groupChatMap[ID]
	if !exist {
		return fmt.Errorf("该群聊不在你的群聊列表中的")
	}
	this.OutputGCMembers(GC)
	return
}

func (this *GroupChatMgr) OutputGCMembers(GC *message.GroupChat) {
	for member, info := range GC.GroupMember {
		fmt.Printf("群用户ID:%d\t群昵称:%s\t%s\n", member, info.NickNameInGC, info.Role.Visualize())
	}
}

func (this *GroupChatMgr) AddNewMember2GC(usrID int, GCID int, info message.UserInfoInGroupChat) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := this.groupChatMap[GCID]; exist {
		this.groupChatMap[GCID].GroupMember[usrID] = info
	}
}

func (this *GroupChatMgr) GetGroupChatInfoByID(ID int) (GC *message.GroupChat) {
	mu.Lock()
	defer mu.Unlock()
	GC = this.groupChatMap[ID]
	return
}

func (this *GroupChatMgr) GetGCNameById(ID int) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	GC, exist := this.groupChatMap[ID]
	if !exist {
		return "", false
	}
	return GC.GroupName, true
}

func (this *GroupChatMgr) outputGCMesList() {
	mu.Lock()
	defer mu.Unlock()
	for _, GC := range this.groupChatMap {
		fmt.Printf("群ID:%d\t%s\t%d条未读消息\n", GC.GroupID, GC.GroupName, this.groupChatsUnreadMesCountMap[GC.GroupID])
	}
}

func (this *GroupChatMgr) NewGroupChatMgr() {
	mu.Lock()
	defer mu.Unlock()
	this.groupChatMap = make(map[int]*message.GroupChat, 1024)
	this.groupChatsUnreadMesCountMap = make(map[int]int, 1024)
}

func (this *GroupChatMgr) GetGCUnreadMesCountByID(GCID int) (count int, exist bool) {
	mu.Lock()
	defer mu.Unlock()
	count, exist = this.groupChatsUnreadMesCountMap[GCID]
	return
}

func (this *GroupChatMgr) AcquireUnreadMesCount(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-logOutChan:
			return
		case ID := <-unreadGCMsgCountChan:
			mu.Lock()
			this.groupChatsUnreadMesCountMap[ID]++
			mu.Unlock()
			updateUnreadGCMesDone <- struct{}{}
		case ID := <-readGCMsgCountChan:
			mu.Lock()
			this.groupChatsUnreadMesCountMap[ID] = 0
			mu.Unlock()
		}
	}
}

func (this *GroupChatMgr) AddGroupChatToMap(GC *message.GroupChat) {
	mu.Lock()
	defer mu.Unlock()
	this.groupChatMap[GC.GroupID] = GC
	this.groupChatsUnreadMesCountMap[GC.GroupID] = 0
}
