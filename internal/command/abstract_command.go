package internalCommand

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var aImageSuffixes = []string{".jpg", ".jpeg", ".png", ".bmp", ".webp"}

// AbstractCommand 放各 command 共用的東西：列出圖片目錄、限制同時執行的數量。
// 各 command 自己宣告並接收所需的 usecase，彼此不共用實作。
type AbstractCommand struct{}

func NewAbstractCommand() *AbstractCommand {
	return &AbstractCommand{}
}

// ListImages 回傳 sWorkdir 底下（只讀第一層）的圖片，依檔名排序；沒有圖片就回錯誤。
func (oSelf *AbstractCommand) ListImages(sWorkdir string) ([]string, error) {
	aEntries, err := os.ReadDir(sWorkdir)
	if err != nil {
		return nil, fmt.Errorf("read workdir: %w", err)
	}

	var aImages []string
	for _, oEntry := range aEntries {
		if oEntry.IsDir() {
			continue
		}
		sSuffix := strings.ToLower(filepath.Ext(oEntry.Name()))
		for _, sImageSuffix := range aImageSuffixes {
			if sSuffix == sImageSuffix {
				aImages = append(aImages, filepath.Join(sWorkdir, oEntry.Name()))
				break
			}
		}
	}
	if len(aImages) == 0 {
		return nil, fmt.Errorf("%s 底下沒有圖片（%s）", sWorkdir, strings.Join(aImageSuffixes, ", "))
	}
	sort.Strings(aImages)

	return aImages, nil
}

// RunParallel 對 0 ~ nCount-1 各呼叫一次 fnRun，同時最多跑 iThreads 個，全部跑完才回來。
func (oSelf *AbstractCommand) RunParallel(nCount int, iThreads int, fnRun func(iIndex int)) {
	var oWaitGroup sync.WaitGroup
	aSlots := make(chan struct{}, iThreads)
	for iIndex := 0; iIndex < nCount; iIndex++ {
		oWaitGroup.Add(1)
		aSlots <- struct{}{}
		go func() {
			defer oWaitGroup.Done()
			defer func() { <-aSlots }()
			fnRun(iIndex)
		}()
	}
	oWaitGroup.Wait()
}
