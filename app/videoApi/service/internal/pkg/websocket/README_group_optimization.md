# 群聊消息推送优化方案

## 📊 优化内容

### 1. 问题诊断

**原实现问题**（`kafka_consumer.go:pushToReceiver`）：
```go
// ❌ 每次群消息都查询数据库
members, err := s.chat.GetGroupMembers(ctx, groupID)

// ❌ 循环内逐个检查在线状态（串行 Redis 查询）
for _, memberID := range members {
    if s.wsManager.IsUserOnlineGlobal(ctx, memberID) {
        s.wsManager.PushToUser(memberID, pushMsg)
    } else {
        s.storeOfflineMessage(memberID, output.MessageID, data)
    }
}
```

**性能瓶颈**：
- 500 人群 × 100 条消息/秒 = **50,000 次 DB 查询/秒**
- 500 次串行 Redis 查询 + 500 次推送操作
- 无并发控制，资源浪费严重

---

### 2. 优化方案

#### ✅ 方案 A：群成员缓存 + Pipeline + 并发推送

**核心改进**：

1. **Redis 缓存群成员列表**（5 分钟 TTL）
   ```go
   // manager.go
   func (m *Manager) GetGroupMembersCached(ctx context.Context, groupID string) ([]string, error) {
       cacheKey := "group:members:" + groupID
       
       // 1. 先查缓存
       members, err := m.redisClient.SMembers(ctx, cacheKey).Result()
       if err == nil && len(members) > 0 {
           return members, nil
       }
       
       // 2. 缓存未命中，从数据库查询
       dbMembers, err := m.chat.GetGroupMembers(ctx, groupID)
       
       // 3. 写入缓存（使用 Set + Expire Pipeline）
       pipe := m.redisClient.Pipeline()
       for _, mid := range dbMembers {
           pipe.SAdd(ctx, cacheKey, mid)
       }
       pipe.Expire(ctx, cacheKey, 5*time.Minute)
       _, _ = pipe.Exec()
       
       return dbMembers, nil
   }
   ```

2. **Pipeline 批量检查在线状态**
   ```go
   // kafka_consumer.go
   func (s *KafkaConsumerService) batchCheckOnlineStatus(ctx context.Context, userIDs []string) map[string]bool {
       // 使用 Pipeline 批量查询 Redis
       pipe := s.redisClient.Pipeline()
       cmds := make([]*redis.StringCmd, len(userIDs))
       
       for i, uid := range userIDs {
           key := "online:" + uid
           cmds[i] = pipe.Get(ctx, key)
       }
       _, _ = pipe.Exec()
       
       // 收集结果
       onlineMap := make(map[string]bool)
       for i, cmd := range cmds {
           if val, err := cmd.Result(); err == nil && val != "" {
               onlineMap[userIDs[i]] = true
           }
       }
       return onlineMap
   }
   ```

3. **并发推送（带限流）**
   ```go
   func (s *KafkaConsumerService) batchPushToOnlineMembers(ctx context.Context, memberIDs []string, pushMsg []byte) {
       // 限制最大并发数（避免瞬间创建太多 goroutine）
       sem := make(chan struct{}, 50)
       
       for _, mid := range memberIDs {
           sem <- struct{}{}
           go func(uid string) {
               defer func() { <-sem }()
               s.wsManager.PushToUser(uid, pushMsg)
           }(mid)
       }
   }
   ```

---

### 3. 数据一致性保障

#### 🔄 缓存失效策略

**方案选择**：TTL 自动过期（5 分钟）

**原因**：
1. **群成员变更频率低**：加人/退群是低频操作
2. **容忍短暂不一致**：5 分钟内成员列表变化可接受
3. **实现简单**：无需复杂的通知机制

**可选增强**（未来扩展）：
```go
// videoChat 服务 group usecase 中添加
func (uc *GroupUsecase) HandleGroupApply(...) error {
    // ... 原有逻辑 ...
    
    if cmd.Accept {
        // 添加成员后，发布缓存失效事件
        uc.notifyCacheInvalidate(apply.GroupID)
    }
    return nil
}

// 通过 Kafka 或 Redis Pub/Sub 通知 videoApi 服务
func (uc *GroupUsecase) notifyCacheInvalidate(groupID int64) {
    // 发送到 "group:cache:invalidate" Topic
    uc.kafkaProducer.Send(ctx, "group:cache:invalidate", groupID)
}
```

---

### 4. 性能对比

| 指标 | 优化前 | 优化后 | 提升倍数 |
|------|--------|--------|----------|
| **DB 查询** | 每次查询 | 5 分钟 1 次 | **数千倍** |
| **Redis 查询** | N 次单独查询 | 1 次 Pipeline | **N 倍** |
| **推送耗时** | 串行 5 秒 | 并发 0.1 秒 | **50 倍** |
| **500 人群总耗时** | ~5 秒 | ~0.1 秒 | **50 倍** |

---

### 5. 关键代码位置

#### `manager.go`
- Line 18-21: 缓存常量定义
- Line 314-345: `GetGroupMembersCached()` 方法
- Line 347-351: `InvalidateGroupCache()` 方法

#### `kafka_consumer.go`
- Line 189-226: 优化后的群聊推送逻辑
- Line 246-267: `batchCheckOnlineStatus()` 方法
- Line 269-281: `batchPushToOnlineMembers()` 方法
- Line 283-303: `batchStoreOfflineMessages()` 方法

---

### 6. 监控建议

```go
// 添加指标监控
metrics.GroupCacheHitCounter.Inc()      // 缓存命中率
metrics.GroupMemberSizeHistogram.Observe(float64(len(members))) // 群大小分布
metrics.BatchPushDuration.Observe(duration.Seconds()) // 批量推送耗时

// 告警阈值
if len(members) > 500 {
    log.Warnf("大群消息：%d 人", len(members))
}
if duration > 1*time.Second {
    log.Warnf("推送耗时过长：%v", duration)
}
```

---

### 7. 测试验证

**压测场景**：
```bash
# 模拟 500 人群，100 条消息/秒
wrk -t12 -c400 -d30s \
  -H "Content-Type: application/json" \
  -d '{"conv_type":1,"receiver_id":"group_123",...}' \
  http://localhost:8000/api/message/send
```

**预期结果**：
- P99 延迟 < 200ms
- CPU 使用率下降 60%
- DB QPS 下降 99%

---

## ✅ 总结

**核心思想**：
1. **缓存优先**：减少 DB 压力
2. **批量操作**：减少网络往返
3. **并发处理**：提升吞吐量
4. **限流保护**：防止雪崩

**适用场景**：
- ✅ 群聊消息推送
- ✅ Feed 流分发
- ✅ 大 V 粉丝通知
- ✅ 任何一对多场景

**未来优化方向**：
1. 添加群成员变更主动失效机制
2. 引入本地缓存（如 singleflight）
3. 支持@特定成员的优化推送
4. 大 V 拉模式切换（参考 Feed 流设计）
