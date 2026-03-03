package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	epoch          int64 = 1704067200000 // 起始时间戳：2024-01-01 00:00:00 UTC
	timestampBits        = 41
	machineIDBits        = 10
	sequenceBits         = 12
	maxMachineID         = -1 ^ (-1 << machineIDBits) // 1023
	maxSequence          = -1 ^ (-1 << sequenceBits)  // 4095
	machineIDShift       = sequenceBits
	timestampShift       = sequenceBits + machineIDBits
)

type Snowflake struct {
	mu        sync.Mutex
	timestamp int64 // 上次生成ID的时间戳
	machineID int64 // 机器ID
	sequence  int64 // 序列号
}

func New(machineID int64) (*Snowflake, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, errors.New("machine id must be between 0 and 1023")
	}
	return &Snowflake{machineID: machineID}, nil
}

func (s *Snowflake) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli() - epoch

	if now == s.timestamp {
		// 同一毫秒内，序列号递增
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// 序列号溢出，等待下一毫秒
			for now <= s.timestamp {
				now = time.Now().UnixMilli() - epoch
			}
		}
	} else if now > s.timestamp {
		// 新的毫秒，重置序列号
		s.sequence = 0
	} else {
		// 时钟回拨，等待追上
		for now < s.timestamp {
			time.Sleep(time.Millisecond)
			now = time.Now().UnixMilli() - epoch
		}
		s.sequence = 0
	}

	s.timestamp = now

	// 组装ID
	return (now << timestampShift) | (s.machineID << machineIDShift) | s.sequence
}

// Parse 解析ID
func Parse(id int64) (timestamp time.Time, machineID int64, sequence int64) {
	ts := (id >> timestampShift) + epoch
	machineID = (id >> machineIDShift) & maxMachineID
	sequence = id & maxSequence
	return time.UnixMilli(ts), machineID, sequence
}
