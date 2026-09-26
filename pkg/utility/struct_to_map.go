package pkgUtility

import (
	"encoding/json"
	"fmt"
)

func StructToMap[T any](oValue T) (map[string]any, error) {

	fmt.Printf("%#v \n", oValue)

	aByteValue, err := json.Marshal(oValue)
	// go 的 json.Marshal 相當 js 的 JSON.stringfy. 只不過 輸出是 byte[]
	if err != nil {
		return nil, err
	}

	var oResult map[string]any
	if err := json.Unmarshal(aByteValue, &oResult); err != nil {
		return nil, err
	}

	return oResult, nil
}
