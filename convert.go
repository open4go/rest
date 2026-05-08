package rest

import (
	"fmt"
	"strings"
)

// ToString 封装一个转换函数（推荐）
func ToString(v interface{}) (string, error) {
	if v == nil {
		return "", fmt.Errorf("输入不能为空")
	}

	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val), nil
	case []byte:
		return strings.TrimSpace(string(val)), nil
	case fmt.Stringer:
		return strings.TrimSpace(val.String()), nil
	case float64, float32, int, int64, int32:
		return fmt.Sprintf("%v", val), nil
	default:
		return "", fmt.Errorf("不支持的数据类型: %T", v)
	}
}
