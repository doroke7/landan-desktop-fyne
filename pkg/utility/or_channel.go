package pkgUtility

//  1. Or-channel 的例子提醒我們，
//
// 在go的世界裡面，
// 我們提供另外一種以 channel 的角度，
// 去看 race 併發行為。

func OrChannel(channels ...<-chan struct{}) <-chan struct{} {
	if len(channels) == 0 {
		return nil
	}
	if len(channels) == 1 {
		return channels[0]
	}

	orChan := make(chan struct{})

	go func() {
		defer close(orChan)

		if len(channels) == 2 {
			select {
			case <-channels[0]:
			case <-channels[1]:
			}
			return
		}

		// 遞迴核心：將前兩個與後面剩餘的所有 channel 展開組合
		select {
		case <-channels[0]:
		case <-channels[1]:
		case <-OrChannel(channels[2:]...):
		}
	}()

	return orChan
}
