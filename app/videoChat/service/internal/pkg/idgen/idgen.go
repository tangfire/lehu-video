package idgen

import (
	"errors"
	"lehu-video/app/videoChat/service/internal/biz"
	"sync"
	"time"
)

const (
	epoch          int64 = 1609459200000 // 2021-01-01 00:00:00 UTC
	machineIDBits  uint8 = 10
	sequenceBits   uint8 = 12
	machineIDShift uint8 = sequenceBits
	timestampShift uint8 = sequenceBits + machineIDBits
	maxMachineID   int64 = -1 ^ (-1 << machineIDBits)
	maxSequence    int64 = -1 ^ (-1 << sequenceBits)
)

// SnowflakeIDGenerator 雪花算法ID生成器
type SnowflakeIDGenerator struct {
	mutex     sync.Mutex
	machineID int64
	lastStamp int64
	sequence  int64
}

// NewSnowflakeIDGenerator 创建新的雪花ID生成器
func NewSnowflakeIDGenerator(machineID int64) (*SnowflakeIDGenerator, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, errors.New("machine ID out of range")
	}

	return &SnowflakeIDGenerator{
		machineID: machineID,
		lastStamp: -1,
		sequence:  0,
	}, nil
}

// NewIDGenerator 创建ID生成器
func NewIDGenerator() (biz.IDGenerator, error) {
	// 使用机器ID 1（实际应根据部署环境配置）
	return NewSnowflakeIDGenerator(1)
}

// Generate 生成唯一ID
func (g *SnowflakeIDGenerator) Generate() int64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	currentStamp := g.getCurrentStamp()
	if currentStamp < g.lastStamp {
		panic("clock moved backwards")
	}

	if currentStamp == g.lastStamp {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			currentStamp = g.getNextMillis()
		}
	} else {
		g.sequence = 0
	}

	g.lastStamp = currentStamp

	return ((currentStamp - epoch) << timestampShift) |
		(g.machineID << machineIDShift) |
		g.sequence
}

func (g *SnowflakeIDGenerator) getCurrentStamp() int64 {
	return time.Now().UnixNano() / 1e6
}

func (g *SnowflakeIDGenerator) getNextMillis() int64 {
	millis := g.getCurrentStamp()
	for millis <= g.lastStamp {
		millis = g.getCurrentStamp()
	}
	return millis
}
