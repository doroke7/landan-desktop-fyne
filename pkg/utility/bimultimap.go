package pkgUtility

import "github.com/cornelk/hashmap"

// Hashable 是 cornelk/hashmap 接受的 key 型別集合（數值與字串）。
// 它自己的 key constraint 沒有匯出，這裡列一份等價的給 BiMultiMap 當型別參數約束用。
type Hashable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

type BiMultiMap[L, R Hashable] struct {
	leftToRights *hashmap.Map[L, *hashmap.Map[R, struct{}]]
	rightToLefts *hashmap.Map[R, *hashmap.Map[L, struct{}]]
}

func NewBiMultiMap[L, R Hashable]() *BiMultiMap[L, R] {
	return &BiMultiMap[L, R]{
		leftToRights: hashmap.New[L, *hashmap.Map[R, struct{}]](),
		rightToLefts: hashmap.New[R, *hashmap.Map[L, struct{}]](),
	}
}

/*

{
	"conn-1": {
		"#room-1": true,
		"#room-2": false,
	},
	"conn-2": {
		"#room-1": {},
		"#room-5": {},
	}
}


*/

// Insert 建立 oLeft/oRight 的雙向關聯，重複呼叫同一組 oLeft/oRight 不會有副作用。
func (oSelf *BiMultiMap[L, R]) Insert(oLeft L, oRight R) {
	oRights, _ := oSelf.leftToRights.GetOrInsert(oLeft, hashmap.New[R, struct{}]())
	oRights.Set(oRight, struct{}{})

	oLefts, _ := oSelf.rightToLefts.GetOrInsert(oRight, hashmap.New[L, struct{}]())
	oLefts.Set(oLeft, struct{}{})
}

// Left 回傳這個 left 目前關聯到的所有 right。
func (oSelf *BiMultiMap[L, R]) Left(oLeft L) []R {
	oRights, bGotten := oSelf.leftToRights.Get(oLeft)
	if !bGotten {
		return nil
	}

	aResult := make([]R, 0, oRights.Len())
	oRights.Range(func(oRight R, _ struct{}) bool {
		aResult = append(aResult, oRight)
		return true
	})

	return aResult
}

// Right 回傳這個 right 目前關聯到的所有 left。
func (oSelf *BiMultiMap[L, R]) Right(oRight R) []L {
	oLefts, bGotten := oSelf.rightToLefts.Get(oRight)
	if !bGotten {
		return nil
	}

	aResult := make([]L, 0, oLefts.Len())
	oLefts.Range(func(oLeft L, _ struct{}) bool {
		aResult = append(aResult, oLeft)
		return true
	})

	return aResult
}

// Remove 拆掉單一一組 oLeft/oRight 的關聯，其他跟 oLeft 或 oRight 相關的關聯不受影響。
func (oSelf *BiMultiMap[L, R]) Remove(oLeft L, oRight R) {
	if oRights, bGotten := oSelf.leftToRights.Get(oLeft); bGotten {
		oRights.Del(oRight)
		if oRights.Len() == 0 {
			oSelf.leftToRights.Del(oLeft)
		}
	}

	if oLefts, bGotten := oSelf.rightToLefts.Get(oRight); bGotten {
		oLefts.Del(oLeft)
		if oLefts.Len() == 0 {
			oSelf.rightToLefts.Del(oRight)
		}
	}
}

// RemoveLeft 拆掉這個 left 的全部關聯，連帶清掉對面每個 right 記著這個 left 的紀錄。
func (oSelf *BiMultiMap[L, R]) RemoveLeft(oLeft L) {
	oRights, bGotten := oSelf.leftToRights.Get(oLeft)
	if !bGotten {
		return
	}

	oRights.Range(func(oRight R, _ struct{}) bool {
		if oLefts, bLefts := oSelf.rightToLefts.Get(oRight); bLefts {
			oLefts.Del(oLeft)
			if oLefts.Len() == 0 {
				oSelf.rightToLefts.Del(oRight)
			}
		}
		return true
	})

	oSelf.leftToRights.Del(oLeft)
}

// RemoveRight 拆掉這個 right 的全部關聯，連帶清掉對面每個 left 記著這個 right 的紀錄。
func (oSelf *BiMultiMap[L, R]) RemoveRight(oRight R) {
	oLefts, bGotten := oSelf.rightToLefts.Get(oRight)
	if !bGotten {
		return
	}

	oLefts.Range(func(oLeft L, _ struct{}) bool {
		if oRights, bRights := oSelf.leftToRights.Get(oLeft); bRights {
			oRights.Del(oRight)
			if oRights.Len() == 0 {
				oSelf.leftToRights.Del(oLeft)
			}
		}
		return true
	})

	oSelf.rightToLefts.Del(oRight)
}
