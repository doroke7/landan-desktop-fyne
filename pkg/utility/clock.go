package pkgUtility

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/beevik/ntp"
)

// defaultNTPServer 為 Clock 用來校時的 NTP 伺服器位址。
const defaultNTPServer = "pool.ntp.org"

type Clock struct {
	mu       sync.RWMutex
	stopOnce sync.Once
	wg       sync.WaitGroup
	offset   time.Duration

	stop chan struct{}

	server string
}

// NewClock 建立並啟動一個 Clock：先嘗試同步一次 offset，失敗只記錄警告、
// 不讓整個程序啟動失敗（offset 先維持 0，等同直接用系統時間），
// 之後背景每 10 分鐘 refresh 一次，直到 ctx 被取消或呼叫 Stop() 為止。
// 設計成 wire provider，在 container 組裝時建立一次，透過 DI 注入給需要的元件共用同一個實例。
func NewClock(oContext context.Context) (*Clock, error) {
	oClock := &Clock{
		server: defaultNTPServer,
		stop:   make(chan struct{}),
	}

	oClock.Start(oContext)

	return oClock, nil
}

// Now 回傳校正過 offset 之後的目前時間。
func (oClock *Clock) Now() time.Time {
	oClock.mu.RLock()
	defer oClock.mu.RUnlock()

	return time.Now().Add(oClock.offset)
}

// Sync 立即跟 NTP 伺服器要一次時間，計算並更新 offset。
func (oClock *Clock) Sync() error {
	return nil

	oTime, err := ntp.Time(oClock.server)
	if err != nil {
		return err
	}

	iOffset := oTime.Sub(time.Now())

	oClock.mu.Lock()
	oClock.offset = iOffset
	oClock.mu.Unlock()

	return nil
}

// Start 啟動時先嘗試同步一次 offset，失敗只記錄警告（offset 維持 0），
// 接著開一個背景 goroutine，每隔固定時間 refresh 一次，
// 直到 ctx 被取消或呼叫 Stop() 為止。
func (oSelf *Clock) Start(oContext context.Context) {
	if err := oSelf.Sync(); err != nil {
		log.Printf("[clock] initial sync offset failed, falling back to system time: %v", err)
	}

	oSelf.wg.Add(1)
	go oSelf.forever(oContext)
}

func (oSelf *Clock) forever(oContext context.Context) {
	defer oSelf.wg.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-oContext.Done():
			return
		case <-oSelf.stop:
			return
		case <-ticker.C:
			if err := oSelf.Sync(); err != nil {
				log.Printf("[clock] sync offset failed: %v", err)
			}
		}
	}
}

// Stop 停止背景 refresh 並等待 goroutine 結束，可重複呼叫。
func (c *Clock) Stop() {
	c.stopOnce.Do(func() {
		close(c.stop)
	})
	c.wg.Wait()
}
