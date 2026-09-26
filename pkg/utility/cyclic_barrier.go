package pkgUtility

import (
	"sync"
)

/*

CyclicBarrier 就是把原本 水平切分 N 分的 任務
然後各自任務 再切為 m 段 step，
每個協程 1段 step 完成才能往下一step

*/

// CyclicBarrier 結構體
type CyclicBarrier struct {
	mu      sync.Mutex
	parties int           // 觸發屏障所需的目標總人數
	count   int           // 當前已經到達的人數
	barrier chan struct{} // 用來讓協程等待的廣播通道
}

// NewCyclicBarrier 初始化一個循環屏障，傳入需要等待的協程數量
func NewCyclicBarrier(parties int) *CyclicBarrier {
	if parties <= 0 {
		panic("parties 必須大於 0")
	}
	return &CyclicBarrier{
		parties: parties,
		barrier: make(chan struct{}),
	}
}

// Await 協程呼叫此方法表示「我到了，我開始等大家」
func (oSelf *CyclicBarrier) Await() {
	oSelf.mu.Lock()
	oSelf.count++

	// 暫存當前的 barrier 通道。
	// 因為最後一個人到的時候會替換掉 cb.barrier，我們必須保留舊通道的引用來進行 close 廣播。
	currentBarrier := oSelf.barrier

	if oSelf.count == oSelf.parties {
		// 【情況 A】最後一個人到了！

		oSelf.count = 0                     // 1. 重置計數器
		oSelf.barrier = make(chan struct{}) // 2. 建立全新的通道（這就是 Cyclic 循環重複使用的關鍵）

		close(currentBarrier) // 3. 一個 close 會喚醒所有 取通道的人。
		oSelf.mu.Unlock()
	} else {
		// 【情況 B】人還沒到齊
		oSelf.mu.Unlock() // 先解鎖，讓其他隊友也能進來 Await()
		<-currentBarrier  // 卡在舊通道這裡，死等最後一個人來 close 它
	}
}
