package usecaseApplicationCommand

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var aImageSuffixes = []string{".jpg", ".jpeg", ".png", ".bmp", ".webp"}

// AbstractUsecase 放各 usecase 共用的東西：列出圖片目錄。
// 各 usecase 自己宣告並接收所需的 output port，不引用其他 usecase。
type AbstractUsecase struct{}

func NewAbstractUsecase() *AbstractUsecase {
	return &AbstractUsecase{}
}

// ListImages 回傳 sWorkdir 底下（只讀第一層）的圖片，依檔名排序；沒有圖片就回錯誤。
func (oSelf *AbstractUsecase) ListImages(sWorkdir string) ([]string, error) {
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
