package raft

import (
	"6.824/raft/lib/logger"
	"6.824/raft/lib/my_panic"
	"fmt"
	"log"
	"os"
	"sync"
)

type Entry struct {
	Term    int32
	Command interface{}
}

type Logs struct {
	entries []Entry
	mu      sync.RWMutex
}

func MakeLogs() *Logs {
	return &Logs{
		entries: make([]Entry, 0),
		mu:      sync.RWMutex{},
	}
}

func (l *Logs) Append(entries ...Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range entries {
		l.entries = append(l.entries, entry)
	}
}

func (l *Logs) AppendLogs(logs ...*Logs) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, ls := range logs {
		func() {
			ls.mu.RLock()
			defer ls.mu.RUnlock()
			l.Append(ls.entries...)
		}()
	}
}

func (l *Logs) Get(index int) Entry {
	defer func() {
		if e := recover(); e != nil {
			my_panic.PrintStack()
			os.Exit(1)
		}
	}()
	if l.Len() == 0 || index == -1 {
		return Entry{Term: -1}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index > len(l.entries)-1 {
		logger.Error(fmt.Sprintf("The arguments of Logs.Get can't be '%d' because logs.Len='%d')", index, len(l.entries)))
		panic(fmt.Sprintf("The arguments of Logs.Get can't be '%d' because logs.Len='%d')", index, len(l.entries)))
	}
	return l.entries[index]
}

func (l *Logs) GetByRange(start int, stop int) []Entry {
	if l.Len() == 0 {
		return []Entry{}
	}
	if start < 0 || start > stop || stop > l.Len() {
		log.Fatal(fmt.Sprintf("The arguments of Logs.GetByRange can't be (%d, %d) when len is %d", start, stop, l.Len()))
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.entries[start:stop]
}

func (l *Logs) GetLast() Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.entries) == 0 {
		return Entry{Term: -1}
	}

	return l.entries[len(l.entries)-1]
}

func (l *Logs) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

func (l *Logs) LenInt32() int32 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return int32(len(l.entries))
}

func (l *Logs) LenFromStart(start int) int32 {
	return int32(len(l.GetByRange(start, l.Len()-1)))
}

func (l *Logs) Set(index int, entry Entry) {
	if index > len(l.entries)-1 {
		log.Fatal(fmt.Sprintf("The arguments of Logs.Set can't be (%d, some_entry)", index))
	}
	l.entries[index] = entry
}

func (l *Logs) FindLastLogByTerm(term int32) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := -1

	for index, entry := range l.entries {
		if entry.Term <= term {
			result = index
		} else {
			break
		}
	}

	return result
}

func (l *Logs) DeleteToRear(start int) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if start > len(l.entries) || start == -1 {
		return len(l.entries)
	}
	l.entries = l.entries[:start]
	return len(l.entries)
}

func (l *Logs) LastIndex() int {
	return l.Len() - 1
}

func (l *Logs) LastIndexInt32() int32 {
	return int32(l.LastIndex())
}

func (l *Logs) LastTerm() int {
	return int(l.GetLast().Term)
}

func (l *Logs) LastTermInt32() int32 {
	return l.GetLast().Term
}

func (l *Logs) Has(start int, entries []Entry) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(entries) == 0 || start+len(entries) > len(l.entries) {
		return false
	}
	for i := start; i < start+len(entries); i++ {
		if l.entries[i] != entries[i-start] {
			return false
		}
	}
	return true
}
