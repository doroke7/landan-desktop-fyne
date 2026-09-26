package pkgUtility

import (
	"context"
	"errors"
	"sync"
)

// Race 接收多個任務，只要有一個成功（err == nil）就立刻返回該結果。
// 並且會自動透過 Context 取消其他還在執行的任務，防止資源洩漏。
func Race[T any](oContext context.Context, aTasks ...func(ctx context.Context) (T, error)) (T, error) {
	// 建立一個可以手動取消的子 Context，用來通知其他慢的任務停下來
	oRaceContext, cRaceCancle := context.WithCancel(oContext)
	defer cRaceCancle() // 確保函數結束時，所有相關協程都會被通知取消

	// 建立一個有緩衝的 Channel，容量與任務數量一致，避免寫入時阻塞
	resultChan := make(chan T, len(aTasks))
	errChan := make(chan error, len(aTasks))

	var oWg sync.WaitGroup

	// 同時啟動所有任務進行「賽跑」
	for _, cTask := range aTasks {
		oWg.Add(1)
		go func(cFunc func(context.Context) (T, error)) {
			defer oWg.Done()

			// 執行任務，傳入我們可以用來取消的 raceCtx
			val, err := cFunc(oRaceContext)
			if err != nil {
				errChan <- err
				return
			}

			// 成功了！將結果塞入 Channel，並立刻取消其他任務
			select {
			case resultChan <- val:
				cRaceCancle() // 成功搶到第一名，通知其他隊友可以洗洗睡了
			case <-oRaceContext.Done():
				// 如果在塞入前，別人已經成功並取消了 Context，就直接退出
			}
		}(cTask)
	}

	// 另外開一個協程監控：如果所有人都失敗了，要關閉錯誤 Channel
	go func() {
		oWg.Wait()
		close(errChan)
	}()

	// 開始等待結果
	select {
	case <-oContext.Done(): // 外部整體的超時或取消
		var zero T
		return zero, oContext.Err()

	case res := <-resultChan: // 恭喜！有人成功跑贏了
		return res, nil

	case <-func() chan struct{} {
		// 這個 case 是用來判斷「是不是所有人都失敗了」
		// 當所有任務都寫入 errChan 且 oWg.Wait() 結束後，errChan 會被關閉
		ch := make(chan struct{})
		go func() {
			for range errChan {
				// 消耗掉所有錯誤
			}
			close(ch)
		}()
		return ch
	}():
		var zero T
		return zero, errors.New("all tasks failed")
	}
}
