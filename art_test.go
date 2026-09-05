package artgo_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/convee/artgo"
	"github.com/stretchr/testify/assert"
)

func TestShutdownWaitsForInFlightRequest(t *testing.T) {
	engine := artgo.New()
	released := make(chan struct{})
	arrived := make(chan struct{})

	engine.GET("/slow", func(c *artgo.Context) {
		close(arrived)
		<-released
		c.String(http.StatusOK, "finished")
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := listener.Addr().String()
	assert.NoError(t, listener.Close())

	runErr := make(chan error, 1)
	go func() { runErr <- engine.Run(addr) }()

	// 等端口真正可连，避免竞态
	var conn net.Conn
	for i := 0; i < 50; i++ {
		if conn, err = net.Dial("tcp", addr); err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.NoError(t, err)

	bodies := make(chan string, 1)
	go func() {
		resp, reqErr := http.Get("http://" + addr + "/slow")
		if reqErr != nil {
			bodies <- "request failed: " + reqErr.Error()
			return
		}
		defer func() { _ = resp.Body.Close() }()
		payload, _ := io.ReadAll(resp.Body)
		bodies <- string(payload)
	}()

	<-arrived // 请求已进入 handler，此刻关闭才验证得了"等待在途请求"
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- engine.Shutdown(ctx)
	}()

	// 必须等 Shutdown 真的开始（listener 已关闭 = 新连接被拒）再放行 handler。
	// 否则 handler 可能在关闭动作发生前就返回，测试会对"是否等待在途请求"零鉴别力。
	closed := false
	for i := 0; i < 200; i++ {
		probe, dialErr := net.Dial("tcp", addr)
		if dialErr != nil {
			closed = true
			break
		}
		_ = probe.Close()
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, closed, "listener 未在超时内关闭，无法验证在途请求")

	close(released)

	assert.Equal(t, "finished", <-bodies, "in-flight request was cut off by shutdown")
	assert.NoError(t, <-shutdownDone)
	assert.True(t, errors.Is(<-runErr, http.ErrServerClosed))
}

func TestShutdownWithoutRunIsNoop(t *testing.T) {
	assert.NoError(t, artgo.New().Shutdown(context.Background()))
}
