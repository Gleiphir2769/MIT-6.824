package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   Start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"6.824/raft/lib/logger"
	"6.824/raft/lib/sync/atomic"
	"6.824/raft/lib/sync/wait"
	"6.824/raft/lib/timewheel"
	"6.824/raft/utils"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	//	"bytes"
	"sync"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	Null int = iota
	LeaderFalse
	EntryLengthLag
	EntryTermLag
	Success
	HeartBeat
	Outdated
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      atomic.Int32        // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	term            atomic.Int32
	leader          atomic.Int32
	voted           atomic.Int32
	refreshed       atomic.Boolean
	refreshCh       chan struct{}
	campaigning     atomic.Boolean
	terminateCampCh chan struct{}
	//allowHeartbeat atomic.Boolean

	logs            *Logs
	committedIndex  atomic.Int32
	appliedIndex    atomic.Int32
	applyMu         sync.Mutex
	applyCh         chan ApplyMsg
	matchIndex      []int32
	matchIndexMutex sync.RWMutex
}

func (rf *Raft) isLeader() bool {
	return rf.me == int(rf.leader.Get())
}

func (rf *Raft) updateLeader(nLeader int32) {
	if nLeader != int32(rf.me) {
		if rf.campaigning.Get() && nLeader != -1 {
			go func() {
				rf.terminateCampCh <- struct{}{}
			}()
		}
	}
	rf.leader.Set(nLeader)
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	return int(rf.term.Get()), rf.isLeader()
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

//
// example RequestVote RPC arguments structure.
// field names must Start with capital letters!
//
type RequestVoteArgs struct {
	Term         int32
	LastLogTerm  int32
	LastLogIndex int32
	CandidateId  int
}

//
// example RequestVote RPC reply structure.
// field names must Start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Agree bool
	Term  int32
}

type AppendEntriesArgs struct {
	LogEntries []Entry
	Term       int32
	LeaderId   int

	PrevLogIndex int32
	PrevLogTerm  int32

	LeaderCommit int32
}

type AppendEntriesReply struct {
	Success       bool
	State         int
	Term          int32
	EntryLagTerm  int32
	EntryLagIndex int32
}

type AcknowledgeCommitArgs struct {
	CommitTerm  int32
	CommitIndex int32
}

type AcknowledgeCommitReply struct {
	Success bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {

	// Your code here (2A, 2B).
	if args.Term >= rf.term.Get() {
		func() {
			if rf.voted.Get() >= args.Term {
				logger.Debug(fmt.Sprintf("node %d disagree vote %d, becouse node %d has voted in term %d", rf.me, args.CandidateId, rf.me, rf.voted.Get()))
				reply.Agree = false
				reply.Term = rf.term.Get()
				return
			}
			if args.LastLogTerm < rf.logs.LastTermInt32() || (args.LastLogTerm == rf.logs.LastTermInt32() && args.LastLogIndex < rf.logs.LastIndexInt32()) {
				logger.Debug(fmt.Sprintf("node %d disagree vote %d, args: {Term: %d, LastLogTerm: %d, logsize: %d} follower: {Term: %d, LastLogTerm %d, logSize: %d}", rf.me, args.CandidateId, args.Term, args.LastLogTerm, args.LastLogIndex, rf.term.Get(), rf.logs.LastTermInt32(), rf.logs.LenInt32()))
				if args.Term > rf.term.Get() {
					rf.term.Set(args.Term)
				}
				reply.Agree = false
				reply.Term = rf.term.Get()
				return
			}
			// 加锁避免被两个节点同时请求投票导致一个任期内投出两张票
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if rf.voted.Get() >= args.Term {
				logger.Debug(fmt.Sprintf("node %d disagree vote %d, becouse node %d has voted in term %d", rf.me, args.CandidateId, rf.me, rf.voted.Get()))
				reply.Term = rf.term.Get()
				reply.Agree = false
				return
			}
			if args.LastLogTerm < rf.logs.LastTermInt32() || (args.LastLogTerm == rf.logs.LastTermInt32() && args.LastLogIndex < rf.logs.LastIndexInt32()) {
				logger.Debug(fmt.Sprintf("node %d disagree vote %d, args: {Term: %d, LastLogTerm: %d, logsize: %d} follower: {Term: %d, LastLogTerm %d, logSize: %d}", rf.me, args.CandidateId, args.Term, args.LastLogTerm, args.LastLogIndex, rf.term.Get(), rf.logs.LastTermInt32(), rf.logs.LenInt32()))
				reply.Agree = false
				reply.Term = rf.term.Get()
				return
			}
			if rf.isLeader() {
				rf.updateLeader(-1)
			}
			rf.refreshVoteTimer()
			logger.Debug(fmt.Sprintf("node %d agree vote %d in term %d (args.logsize %d), follower: voted is %d, term is %d, logSize: %d", rf.me, args.CandidateId, args.Term, args.LastLogIndex, rf.voted.Get(), rf.term.Get(), rf.logs.LenInt32()))
			reply.Term = rf.term.Get()
			reply.Agree = true
			rf.voted.Set(args.Term)
			rf.term.Set(args.Term)

		}()
	} else {
		logger.Debug(fmt.Sprintf("node %d disagree vote %d, args: {term: %d, logSize: %d} follower: {term %d, logSize: %d}", rf.me, args.CandidateId, args.Term, args.LastLogIndex, rf.term.Get(), rf.logs.LenInt32()))
		reply.Agree = false
	}
}

func (rf *Raft) campaignLeader() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.campaigning.Set(true)
	defer rf.campaigning.Set(false)

	if rf.refreshed.Get() {
		rf.refreshed.Set(false)
		return
	}

	if rf.isLeader() {
		rf.updateLeader(-1)
	}

	rf.term.Incr()
	if rf.voted.Get() >= rf.term.Get() {
		logger.Debug(fmt.Sprintf("node %d break leader campaign because has voted to another node", rf.me))
		return
	}
	rf.voted.Set(rf.term.Get())

	major := len(rf.peers)/2 + 1
	peersNum := len(rf.peers)

	wg := wait.Wait{}
	wg.Add(peersNum)
	support := atomic.Int32(1)

	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		arg := &RequestVoteArgs{
			Term:         rf.term.Get(),
			LastLogTerm:  rf.logs.LastTermInt32(),
			LastLogIndex: rf.logs.LastIndexInt32(),
			CandidateId:  rf.me,
		}
		reply := &RequestVoteReply{}
		go func(server int) {
			rf.sendRequestVote(server, arg, reply)
			if reply.Agree {
				support.Incr()
			}
			if reply.Term > rf.term.Get() {
				rf.term.Set(reply.Term)
			}
			wg.Done()
		}(i)
	}

	finishCh := make(chan struct{}, 1)
	termWaitCh := make(chan struct{}, 1)
	defer close(finishCh)
	go func() {
		// 等待时间略小于election time
		wg.WaitWithTimeout(time.Millisecond * 150)
		select {
		case <-termWaitCh:
			close(termWaitCh)
			return
		default:
			finishCh <- struct{}{}
		}
	}()

	select {
	case <-finishCh:
		if int(support.Get()) < major {
			logger.Debug(fmt.Sprintf("node %d campaign leader failed (term %d), get %d agrees", rf.me, rf.term, support.Get()))
			rf.updateLeader(-1)
			return
		}
		rf.updateLeader(int32(rf.me))
		logger.Info(fmt.Sprintf("node %d become leader (term %d)", rf.me, rf.term.Get()))
		rf.refreshVoteTimer()
	case <-rf.terminateCampCh:
		logger.Debug(fmt.Sprintf("node %d campaign leader has been terminated because new leader %d come", rf.me, rf.leader))
		termWaitCh <- struct{}{}
		rf.refreshVoteTimer()
		return
	}
}

func (rf *Raft) apply() {
	for {
		for rf.appliedIndex.Get() < rf.committedIndex.Get() {
			rf.appliedIndex.Incr()
			am := ApplyMsg{
				CommandValid: true,
				Command:      rf.logs.Get(int(rf.appliedIndex.Get())).Command,
				CommandIndex: int(rf.appliedIndex.Get()),
			}
			//logger.Debug(fmt.Sprintf("node %d commit entry (index %d, {term: %d, cmd: %v})", rf.me, rf.appliedIndex, rf.logs.Get(int(rf.appliedIndex.Get())).Term, am.Command))
			rf.applyCh <- am
		}
		time.Sleep(time.Millisecond * 50)
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.term.Get() {
		reply.Success = false
		reply.State = LeaderFalse
		reply.Term = rf.term.Get()
		return
	}

	if len(args.LogEntries) == 0 {
		rf.refreshVoteTimer()

		reply.Success = true
		reply.State = HeartBeat
		reply.Term = rf.term.Get()
		//logger.Debug(fmt.Sprintf("follower %d get HeartBeat from %d in term %d, leader committed index %d, follower commited %d, follower lastIndex %d, agree", rf.me, args.LeaderId, rf.term.Get(), args.LeaderCommit, rf.committedIndex.Get(), rf.logs.LastIndexInt32()))
		return
	}

	if rf.logs.LastIndexInt32() < args.PrevLogIndex {
		logger.Debug(fmt.Sprintf("server %d caused EntryLengthLag error {rf.LastLogIndex: %d, args.PrevLogIndex: %d}", rf.me, rf.logs.LastIndexInt32(), args.PrevLogTerm))
		rf.refreshVoteTimer()
		reply.Success = false
		reply.State = EntryLengthLag
		reply.Term = rf.term.Get()
		reply.EntryLagIndex = rf.logs.LastIndexInt32()
		return
	}

	if rf.logs.Has(int(args.PrevLogIndex+1), args.LogEntries) && rf.logs.Len() != int(args.PrevLogIndex)+len(args.LogEntries) {
		reply.Success = false
		reply.State = Outdated
		reply.Term = rf.term.Get()
		return
	}

	if rf.logs.Get(int(args.PrevLogIndex)).Term != args.PrevLogTerm {
		rf.refreshVoteTimer()
		reply.Success = false
		reply.State = EntryTermLag
		reply.Term = rf.term.Get()
		reply.EntryLagTerm = rf.logs.Get(int(args.PrevLogIndex)).Term
		return
	}

	rf.logs.DeleteToRear(int(args.PrevLogIndex + 1))
	rf.logs.Append(args.LogEntries...)
	rf.persist()

	rf.refreshVoteTimer()

	reply.Success = true
	reply.State = Success
	reply.Term = rf.term.Get()
	return
}

func (rf *Raft) AcknowledgeCommit(args *AcknowledgeCommitArgs, reply *AcknowledgeCommitReply) {
	if rf.logs.LastIndexInt32() >= args.CommitIndex && rf.logs.Get(int(args.CommitIndex)).Term == args.CommitTerm && rf.committedIndex.Get() < args.CommitIndex {
		rf.committedIndex.Set(args.CommitIndex)
	}
	reply.Success = true
}

func (rf *Raft) sendRequestAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)

	// heartbeat
	if len(args.LogEntries) == 0 {
		return ok
	}

	for !ok {
		nReply := &AppendEntriesReply{}
		ok = rf.peers[server].Call("Raft.AppendEntries", args, nReply)
		reply = nReply
		time.Sleep(100 * time.Millisecond)
	}

	for rf.isLeader() {
		switch reply.State {
		case LeaderFalse:
			rf.updateLeader(-1)
			rf.term.Set(reply.Term)
			return false
		case EntryLengthLag:
			nIndex := reply.EntryLagIndex
			nTerm := rf.logs.Get(int(nIndex)).Term
			entries := rf.logs.GetByRange(int(nIndex+1), rf.logs.Len())
			nArgs := &AppendEntriesArgs{
				LogEntries:   entries,
				Term:         rf.term.Get(),
				LeaderId:     rf.me,
				PrevLogIndex: nIndex,
				PrevLogTerm:  nTerm,
				LeaderCommit: rf.committedIndex.Get(),
			}
			nReply := &AppendEntriesReply{}
			ok1 := false
			for !ok1 {
				ok1 = rf.peers[server].Call("Raft.AppendEntries", nArgs, nReply)
				time.Sleep(100 * time.Millisecond)
			}
			if ok1 {
				reply = nReply
				args = nArgs
				rf.refreshVoteTimer()
			}

		case EntryTermLag:
			nIndex := rf.logs.FindLastLogByTerm(reply.EntryLagTerm)
			nTerm := rf.logs.Get(nIndex).Term
			entries := rf.logs.GetByRange(nIndex+1, rf.logs.Len())
			nArgs := &AppendEntriesArgs{
				LogEntries:   entries,
				Term:         rf.term.Get(),
				LeaderId:     rf.me,
				PrevLogIndex: int32(nIndex),
				PrevLogTerm:  nTerm,
				LeaderCommit: rf.committedIndex.Get(),
			}
			nReply := &AppendEntriesReply{}
			ok2 := false
			for !ok2 {
				ok2 = rf.peers[server].Call("Raft.AppendEntries", nArgs, nReply)
				time.Sleep(100 * time.Millisecond)
			}
			if ok2 {
				reply = nReply
				args = nArgs
				rf.refreshVoteTimer()
			}
		case Success:
			rf.matchIndexMutex.Lock()
			rf.matchIndex[server] = args.PrevLogIndex + int32(len(args.LogEntries))
			rf.matchIndex[rf.me] = rf.logs.LastIndexInt32()
			// 根据matchIndex的中位数设置commit index（trick）
			committed := utils.GetMiddleNum(rf.matchIndex)
			rf.matchIndexMutex.Unlock()

			if rf.committedIndex.Get() < committed && rf.logs.Get(int(committed)).Term == rf.term.Get() {
				rf.committedIndex.Set(committed)
				logger.Debug(fmt.Sprintf("leader %d commited log of index %d, command %v", rf.me, committed, rf.logs.Get(int(committed)).Command))
			}

			ackArgs := &AcknowledgeCommitArgs{CommitTerm: rf.logs.Get(int(rf.committedIndex.Get())).Term, CommitIndex: rf.committedIndex.Get()}
			ackReply := &AcknowledgeCommitReply{}
			for i := range rf.peers {
				if i == rf.me {
					continue
				}
				go func(server int) {
					ack := false
					for !ack {
						ack = rf.peers[server].Call("Raft.AcknowledgeCommit", ackArgs, ackReply)
					}
				}(i)
			}

			return true
		case Outdated:
			return false
		case Null:
			logger.Error(fmt.Sprintf("reply: {Success: %v, State: %d, Term: %d, EntryLagTerm %d, EntryLagIndex %d}", reply.Success, reply.State, reply.Term, reply.EntryLagTerm, reply.EntryLagIndex))
			log.Fatal("program can't go here")
		}
	}
	return false
}

// refreshVoteTimer 可能会发生阻塞，务必用协程运行
func (rf *Raft) refreshVoteTimer() {
	rf.refreshed.Set(true)
	go func() { rf.refreshCh <- struct{}{} }()
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to Start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise Start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	index := -1
	term := -1

	if !rf.isLeader() {
		return index, term, false
	}

	entry := Entry{Term: rf.term.Get(), Command: command}

	arg := &AppendEntriesArgs{
		LogEntries:   []Entry{entry},
		Term:         rf.term.Get(),
		LeaderId:     rf.me,
		PrevLogIndex: rf.logs.LastIndexInt32(),
		PrevLogTerm:  rf.logs.LastTermInt32(),
		LeaderCommit: rf.committedIndex.Get(),
	}
	reply := &AppendEntriesReply{}

	rf.logs.Append(entry)
	rf.persist()

	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go rf.sendRequestAppendEntries(i, arg, reply)
	}

	entryIndex := arg.PrevLogIndex + 1
	entryTerm := entry.Term

	return int(entryIndex), int(entryTerm), rf.isLeader()
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	rf.dead.Set(1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := rf.dead.Get()
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {
		if !rf.isLeader() {
			func() {
				startNextElect := make(chan struct{}, 1)
				termStartElect := make(chan struct{}, 1)
				n, _ := rand.Int(rand.Reader, big.NewInt(3))
				electionTime := time.Duration(100*(n.Int64()+4)) * time.Millisecond
				jobKey := fmt.Sprintf("elect-node%d", rf.me)
				go func() {
					time.Sleep(electionTime)
					select {
					case <-termStartElect:
						close(termStartElect)
						return
					default:
						startNextElect <- struct{}{}
					}
				}()
				select {
				case <-startNextElect:
					electAt := time.Now().Add(electionTime)
					timewheel.At(electAt, jobKey, rf.campaignLeader)
					close(startNextElect)
				case <-rf.refreshCh:
					termStartElect <- struct{}{}
					electAt := time.Now().Add(electionTime)
					timewheel.RefreshAt(electAt, jobKey, rf.campaignLeader)
				}
			}()
		} else {
			heartbeatTime := 100 * time.Millisecond
			jobKey := fmt.Sprintf("heartbeat-node%d-term%d-time%d", rf.me, rf.term.Get(), time.Now().UnixNano())
			heartbeatAt := time.Now().Add(heartbeatTime)
			timewheel.At(heartbeatAt, jobKey, rf.heartbeat)
			time.Sleep(heartbeatTime)
		}
	}
}

func (rf *Raft) heartbeat() {
	wg := wait.Wait{}
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		wg.Add(1)
		arg := &AppendEntriesArgs{
			LogEntries:   []Entry{},
			Term:         rf.term.Get(),
			LeaderId:     rf.me,
			PrevLogTerm:  rf.logs.LastTermInt32(),
			PrevLogIndex: rf.logs.LastIndexInt32(),
			LeaderCommit: rf.committedIndex.Get(),
		}
		reply := &AppendEntriesReply{}
		go func(server int) {
			rf.sendRequestAppendEntries(server, arg, reply)
			if reply.Term > rf.term.Get() {
				rf.updateLeader(-1)
				rf.term.Set(reply.Term)
			}
			wg.Done()
		}(i)
	}
	wg.WaitWithTimeout(time.Millisecond * 80)
	rf.refreshVoteTimer()
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should Start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.leader.Set(-1)
	rf.voted = -1
	rf.logs = MakeLogs()
	rf.matchIndex = make([]int32, len(rf.peers))
	for i := 0; i < len(rf.peers); i++ {
		rf.matchIndex[i] = -1
	}
	rf.committedIndex.Set(-1)
	rf.applyCh = applyCh
	rf.terminateCampCh = make(chan struct{}, 1)
	rf.refreshCh = make(chan struct{}, 1)
	rf.appliedIndex.Set(-1)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// Start ticker goroutine to Start elections
	go rf.ticker()
	go rf.apply()

	//rf.Start("start")

	return rf
}
