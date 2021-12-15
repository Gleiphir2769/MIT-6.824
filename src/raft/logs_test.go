package raft

import (
	"fmt"
	"testing"
)

func TestLogs(t *testing.T) {
	l := MakeLogs()
	l.Append(Entry{
		Term:    0,
		Command: nil,
	})
	l.Append(Entry{
		Term:    1,
		Command: nil,
	})
	l.Append(Entry{
		Term:    2,
		Command: nil,
	})
	//nl := l.GetByRange(3, 3)
	l.DeleteToRear(-1)
	fmt.Println(l.entries)
}

func TestHas(t *testing.T) {
	l := MakeLogs()
	entry0 := Entry{
		Term:    0,
		Command: nil,
	}
	entry1 := Entry{
		Term:    1,
		Command: nil,
	}
	entry2 := Entry{
		Term:    2,
		Command: nil,
	}
	l.Append(entry0)
	l.Append(entry1)
	l.Append(entry2)

	entries := make([]Entry, 0)
	entries = append(entries, entry1)
	entries = append(entries, entry0)
	entries = append(entries, entry2)

	fmt.Println(l.Has(0, entries))
}
