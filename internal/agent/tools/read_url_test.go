package tools

// 本文件覆盖 read_url 分页读取工具的核心行为：首段从开头读、按 offset 续读、
// 末段截到末尾不再提示、offset 越界报已读完、非法参数在抓取前被拒绝。测试经
// skipURLGuard 注入缝放行本机 httptest 地址（生产内网防护不受影响）。

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestReadURLPagination 验证 read_url 的 offset/length 分页协议：正文由 12000 个
// "甲" 接 12000 个 "乙" 组成（缺省单段长度 10000），按段读取时应能正确切片且
// 返回定位信息与续读提示。
func TestReadURLPagination(t *testing.T) {
	content := strings.Repeat("甲", 12000) + strings.Repeat("乙", 12000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><body>" + content + "</body></html>"))
	}))
	defer srv.Close()

	// setting 为 nil：getIntSetting 回退缺省单段长度 10000
	h := &readURLTool{skipURLGuard: true}

	t.Run("首段从开头读并提示续读", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), fmt.Sprintf(`{"url":%q}`, srv.URL))
		if err != nil {
			t.Fatalf("首段读取失败: %v", err)
		}
		if !strings.Contains(out, "第 1-10000 字符") {
			t.Errorf("首段应为第 1-10000 字符，实际:\n%.300s", out)
		}
		if !strings.Contains(out, "（内容未完，如需继续请以 offset=10000 调用）") {
			t.Errorf("首段未读完应提示续读 offset=10000，实际:\n%.300s", out)
		}
		if !strings.Contains(out, strings.Repeat("甲", 1000)) {
			t.Error("首段应包含正文开头的甲段")
		}
		if strings.Contains(out, "乙") {
			t.Error("首段（前 10000 字符）不应包含位于 12000 起的乙段")
		}
	})

	t.Run("按 offset 续读中间段", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), fmt.Sprintf(`{"url":%q,"offset":10000}`, srv.URL))
		if err != nil {
			t.Fatalf("续读失败: %v", err)
		}
		if !strings.Contains(out, "第 10001-20000 字符") {
			t.Errorf("续读段应为第 10001-20000 字符，实际:\n%.300s", out)
		}
		if !strings.Contains(out, "（内容未完，如需继续请以 offset=20000 调用）") {
			t.Errorf("中间段未读完应提示续读 offset=20000，实际:\n%.300s", out)
		}
		if !strings.Contains(out, strings.Repeat("甲", 1000)) || !strings.Contains(out, strings.Repeat("乙", 1000)) {
			t.Error("中间段应跨越甲/乙分界，同时包含两种字符")
		}
	})

	t.Run("末段截到末尾不再提示", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), fmt.Sprintf(`{"url":%q,"offset":20000}`, srv.URL))
		if err != nil {
			t.Fatalf("末段读取失败: %v", err)
		}
		if !strings.Contains(out, "第 20001-") {
			t.Errorf("末段应从第 20001 字符起，实际:\n%.300s", out)
		}
		if strings.Contains(out, "内容未完") {
			t.Errorf("末段已到末尾不应提示续读，实际:\n%.300s", out)
		}
		if strings.Contains(out, "甲") {
			t.Error("末段（20000 起）不应包含已结束的甲段")
		}
		if !strings.Contains(out, strings.Repeat("乙", 1000)) {
			t.Error("末段应包含乙段内容")
		}
	})

	t.Run("offset 越界报已读完", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), fmt.Sprintf(`{"url":%q,"offset":24000}`, srv.URL))
		if err == nil {
			t.Fatal("offset 等于总字符数时应报已读完")
		}
		if !strings.Contains(err.Error(), "read_url 的 offset 超出内容范围") || !strings.Contains(err.Error(), "已全部读取完毕") {
			t.Errorf("越界错误应含 read_url 前缀与已读完提示，实际: %v", err)
		}
	})

	t.Run("巨型 offset 溢出防护", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), fmt.Sprintf(`{"url":%q,"offset":1000000000000000000}`, srv.URL)); err == nil {
			t.Fatal("巨型 offset 应报已读完")
		}
	})

	t.Run("非法参数在抓取前被拒绝", func(t *testing.T) {
		var hits int32
		srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&hits, 1)
			_, _ = w.Write([]byte("<html><body>内容</body></html>"))
		}))
		defer srv2.Close()
		h2 := &readURLTool{skipURLGuard: true}

		cases := []string{
			fmt.Sprintf(`{"url":%q,"offset":-1}`, srv2.URL),
			fmt.Sprintf(`{"url":%q,"offset":1.5}`, srv2.URL),
			fmt.Sprintf(`{"url":%q,"length":-5}`, srv2.URL),
			fmt.Sprintf(`{"url":%q,"length":1.5}`, srv2.URL),
		}
		for _, args := range cases {
			if _, err := h2.InvokableRun(context.Background(), args); err == nil {
				t.Errorf("参数 %s 应报错", args)
			}
		}
		if got := atomic.LoadInt32(&hits); got != 0 {
			t.Errorf("非法参数不应触发抓取，服务端收到 %d 次请求", got)
		}
	})
}
