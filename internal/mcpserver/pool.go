package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"gitee.com/MM-Q/fastlog"
)

// Pool 全局 MCP 连接池：按服务器 Name 持有已预热的会话（http/sse/stdio 均常驻复用）。
//
// 预热语义：
//   - Warmup：对传入服务器列表中的 enabled 服务器（http/sse/stdio）建连+发现工具并缓存；
//     已预热且配置指纹未变时直接复用（零网络开销），指纹变化时关闭旧会话后重连。
//   - Reconcile：先关闭池中「不在传入列表」的条目（服务器被停用/删除），再预热剩余，
//     供设置页任何 MCP 操作后调用，保证池与数据库配置一致。
//   - Session：发消息装配时取已预热会话；未命中返回 nil，调用方可选择 WarmupOne 兜底。
//
// 并发安全：多会话并发调用同一池连接的 CallTool 安全（go-sdk jsonrpc2 多路复用）；
// 池内部加锁保护条目增删；同名服务器的建连由 in-flight 信号串行化（并发 Warmup
// 不会对同一台服务器重复建连）。close 操作幂等（Session.Close 本身幂等）。
type Pool struct {
	mu       sync.Mutex
	entries  map[string]*poolEntry                                 // key = server.Name
	inflight map[string]chan struct{}                              // key = server.Name：同名建连串行化信号
	open     func(ctx context.Context, s Server) (*Session, error) // 注入，默认 OpenSession，测试可替换
	log      *fastlog.Logger                                       // 可选，nil 静默
}

// poolEntry 一台 MCP 服务器的池条目。
type poolEntry struct {
	fp   string   // 服务器配置指纹（json.Marshal(Server) 稳定序列化）
	sess *Session // 已预热的会话（含 Tools）
}

// SessionToolMeta 单条 MCP 工具元信息（供设置页展示与开关控制）。
type SessionToolMeta struct {
	ServerName string // 服务器名
	FullName   string // 完整工具名，格式 mcp_{serverName}_{toolName}
}

// WarmupResult 预热/同步结果汇总，供前端一条通知展示。
type WarmupResult struct {
	Total      int      `json:"total"`       // 本次处理的 enabled 服务器数
	Warmed     int      `json:"warmed"`      // 新预热成功数
	Reused     int      `json:"reused"`      // 已预热且指纹未变复用数
	Closed     int      `json:"closed"`      // 因停用/删除/指纹变更关闭数
	Failed     int      `json:"failed"`      // 预热失败数
	FailedMsgs []string `json:"failed_msgs"` // 失败详情（含服务器名+原因），前端展示
	ToolTotal  int      `json:"tool_total"`  // 全部可用会话发现的工具总数
}

// NewPool 创建全局 MCP 连接池。
func NewPool() *Pool {
	return &Pool{
		entries:  make(map[string]*poolEntry),
		inflight: make(map[string]chan struct{}),
		open:     OpenSession,
	}
}

// SetLogger 注入日志器（可选，nil 静默）。
func (p *Pool) SetLogger(l *fastlog.Logger) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log = l
}

// setOpen 注入连接函数（测试用）。
func (p *Pool) setOpen(fn func(ctx context.Context, s Server) (*Session, error)) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.open = fn
}

// openFn 返回当前注入的连接函数。
func (p *Pool) openFn() func(ctx context.Context, s Server) (*Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.open
}

// Warmup 对传入的 enabled 服务器（http/sse/stdio）预热（幂等）：
// 已入池且指纹一致 → 复用；指纹变化 → 关闭旧会话后重连；未入池 → 新建。
// 并发 3 槽位避免串行 10s×N；同名服务器由 getOrCreate 串行化，不会重复建连。
// 失败不缓存（下次 Warmup/兜底再试）。
// 返回汇总结果（不返回 error：部分失败通过 Failed/FailedMsgs 体现，避免调用方中断）。
func (p *Pool) Warmup(ctx context.Context, servers []Server) WarmupResult {
	if p == nil {
		return WarmupResult{}
	}
	targets := make([]Server, 0, len(servers))
	for _, s := range servers {
		if !s.Enabled {
			continue // 仅预热启用的服务器；disabled 不入池
		}
		targets = append(targets, s)
	}

	res := WarmupResult{Total: len(targets)}
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护 res 汇总
	for _, s := range targets {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sess, reused, replaced, err := p.getOrCreate(ctx, s)
			mu.Lock()
			if err != nil {
				res.Failed++
				res.FailedMsgs = append(res.FailedMsgs, err.Error())
				mu.Unlock()
				return
			}
			if reused {
				res.Reused++
			} else {
				res.Warmed++
			}
			if replaced {
				res.Closed++
			}
			res.ToolTotal += len(sess.Tools)
			mu.Unlock()
			if !reused && p.log != nil {
				p.log.Infow("MCP 服务器已预热",
					fastlog.String("server", s.Name),
					fastlog.Int("tools", len(sess.Tools)))
			}
		}(s)
	}
	wg.Wait()
	return res
}

// getOrCreate 取或建指定服务器的池会话（并发安全）：
// 已入池且指纹一致 → 直接复用（reused=true）；
// 否则持有 per-name in-flight 信号（同名同时只有一个建连者），关闭旧条目（指纹变化）后
// 调用 open 建连并回写；建连失败返回 err（不缓存）。
// 返回 (sess, reused, replaced, err)：replaced=true 表示因指纹变化关闭了旧会话。
// 等待者在信号释放后重查池，命中则复用，未命中（并发建连者失败）返回错误。
func (p *Pool) getOrCreate(ctx context.Context, s Server) (*Session, bool, bool, error) {
	fp := serverFingerprint(s)

	// 1. 快速路径：已入池且指纹一致
	p.mu.Lock()
	if e, ok := p.entries[s.Name]; ok && e.fp == fp {
		sess := e.sess
		p.mu.Unlock()
		return sess, true, false, nil
	}
	// 2. 已有同名建连在途：等待其完成后再查
	if ch, ok := p.inflight[s.Name]; ok {
		p.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return nil, false, false, ctx.Err()
		}
		p.mu.Lock()
		if e, ok := p.entries[s.Name]; ok && e.fp == fp {
			sess := e.sess
			p.mu.Unlock()
			return sess, true, false, nil
		}
		p.mu.Unlock()
		return nil, false, false, errors.New("MCP 服务器 " + s.Name + " 并发预热失败（其他建连者未成功）")
	}
	// 3. 注册 in-flight，成为建连者
	p.inflight[s.Name] = make(chan struct{})
	p.mu.Unlock()

	release := func() {
		p.mu.Lock()
		if ch, ok := p.inflight[s.Name]; ok {
			close(ch)
			delete(p.inflight, s.Name)
		}
		p.mu.Unlock()
	}
	defer release()

	// 4. 持有 in-flight：关闭指纹变化的旧条目（幂等）
	replaced := false
	p.mu.Lock()
	if e, ok := p.entries[s.Name]; ok && e.fp != fp {
		_ = e.sess.Close()
		delete(p.entries, s.Name)
		replaced = true
	}
	p.mu.Unlock()

	sess, err := p.openFn()(ctx, s)
	if err != nil {
		return nil, false, replaced, err
	}
	p.mu.Lock()
	p.entries[s.Name] = &poolEntry{fp: fp, sess: sess}
	p.mu.Unlock()
	return sess, false, replaced, nil
}

// Reconcile 同步池与服务器配置：先关闭池中不在 servers 列表里的条目（停用/删除），
// 再预热传入列表中的 enabled 服务器（新增/变更/复用）。
// 设置页任何 MCP 操作后调用，保证池与数据库一致。
func (p *Pool) Reconcile(ctx context.Context, servers []Server) WarmupResult {
	if p == nil {
		return WarmupResult{}
	}
	keep := make(map[string]bool, len(servers))
	for _, s := range servers {
		keep[s.Name] = true
	}

	// 收集需关闭的条目（不在传入列表中的）
	p.mu.Lock()
	var toClose []*Session
	for name, entry := range p.entries {
		if !keep[name] {
			toClose = append(toClose, entry.sess)
			delete(p.entries, name)
		}
	}
	p.mu.Unlock()
	for _, sess := range toClose {
		_ = sess.Close()
		if p.log != nil {
			p.log.Infow("MCP 服务器已从池中关闭（停用/删除）",
				fastlog.String("server", sess.ServerName))
		}
	}

	res := p.Warmup(ctx, servers)
	res.Closed += len(toClose)
	return res
}

// Session 取已预热会话（发消息装配用）；未命中返回 nil。
func (p *Pool) Session(name string) *Session {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.entries[name]; ok {
		return e.sess
	}
	return nil
}

// WarmupOne 单台服务器预热并入池（发消息装配兜底用：池中无该服务器时现场连接一次）。
// 失败不缓存；已入池且指纹一致时直接返回缓存会话；并发同名建连由 getOrCreate 串行化。
func (p *Pool) WarmupOne(ctx context.Context, s Server) (*Session, error) {
	if p == nil {
		return nil, errors.New("MCP 连接池未初始化")
	}
	sess, reused, _, err := p.getOrCreate(ctx, s)
	if err != nil {
		return nil, err
	}
	if !reused && p.log != nil {
		p.log.Infow("MCP 服务器已预热（兜底）",
			fastlog.String("server", s.Name),
			fastlog.Int("tools", len(sess.Tools)))
	}
	return sess, nil
}

// Close 关闭单台服务器的池连接（停用/删除时调用）；不存在时静默。
func (p *Pool) Close(name string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	entry, ok := p.entries[name]
	if ok {
		delete(p.entries, name)
	}
	p.mu.Unlock()
	if ok && entry != nil {
		_ = entry.sess.Close()
		if p.log != nil {
			p.log.Infow("MCP 服务器已从池中关闭",
				fastlog.String("server", name))
		}
	}
}

// CloseAll 关闭全部池连接并清空（应用退出/数据库还原时调用）；幂等。
func (p *Pool) CloseAll() {
	if p == nil {
		return
	}
	p.mu.Lock()
	entries := p.entries
	p.entries = make(map[string]*poolEntry)
	p.mu.Unlock()
	for _, e := range entries {
		if e != nil && e.sess != nil {
			_ = e.sess.Close()
		}
	}
}

// Count 返回池中条目数（调试/测试用）。
func (p *Pool) Count() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}

// ListToolMetas 返回池中所有已预热会话的工具元信息。
// 未预热的服务器不在此列；调用方（GetAgentTools）据此展示，不阻塞等待。
func (p *Pool) ListToolMetas() []SessionToolMeta {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	var metas []SessionToolMeta
	for _, entry := range p.entries {
		if entry.sess == nil {
			continue
		}
		for _, t := range entry.sess.Tools {
			info, err := t.Info(context.Background())
			if err != nil || info == nil {
				continue
			}
			metas = append(metas, SessionToolMeta{
				ServerName: entry.sess.ServerName,
				FullName:   info.Name,
			})
		}
	}
	return metas
}

// serverFingerprint 计算服务器配置指纹：json.Marshal 稳定序列化（map 按键排序），
// 用于判断配置是否变化（URL/Headers/Command 等任一变更都会触发重连）。
func serverFingerprint(s Server) string {
	b, err := json.Marshal(s)
	if err != nil {
		// 极端情况（理论不可能：Server 全为基本类型）：退化为名称+时间戳保证不误复用
		return s.Name + ":" + time.Now().Format("20060102150405.000000000")
	}
	return string(b)
}
