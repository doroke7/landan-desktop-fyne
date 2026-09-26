package pkgUtility

/*
	    思維： Or-Done 是為了解決「用 只用 for-range 思維解決取消 」的結束問題；

		用法：

	   	done := make(chan struct{})
		oChannel := make(chan int)

		。。。


		for iVal := range OrDone(done, oChannel) {
			fmt.Printf("👉 成功處理資料: %d\n", iVal)
		}
*/
func OrDone[T any](done <-chan struct{}, ch <-chan T) <-chan T {
	valChan := make(chan T)

	go func() {
		defer close(valChan) // 確保協程結束時，關閉輸出的 Channel
		for {
			select {
			case <-done:
				// 1. 收到外部取消訊號，立刻退出
				return
			case v, ok := <-ch:
				if !ok {
					// 2. 來源 Channel 已被關閉，正常結束
					return
				}
				// 3. 將讀到的資料轉發出去
				//    這裡必須再次搭配 select，防止在轉發阻塞時，外部突然發出 done 訊號而卡死
				select {
				case valChan <- v:
				case <-done:
					return
				}
			}
		}
	}()

	return valChan
}
