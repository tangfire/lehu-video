package idgen

import (
	"github.com/bwmarrin/snowflake"
)

// Generator 定义 ID 生成器接口
type Generator interface {
	NextID() int64
	NextIDString() string
}

type snowflakeGenerator struct {
	node *snowflake.Node
}

// NewGenerator 创建一个雪花 ID 生成器
func NewGenerator(workerID int64) Generator {
	node, err := snowflake.NewNode(workerID)
	if err != nil {
		return nil
	}
	return &snowflakeGenerator{node: node}
}

func (g *snowflakeGenerator) NextID() int64 {
	return g.node.Generate().Int64()
}

func (g *snowflakeGenerator) NextIDString() string {
	return g.node.Generate().String()
}
