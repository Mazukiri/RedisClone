package core

import (
	"github.com/Mazukiri/RedisClone/internal/constant"
	"strconv"
)

func deduceTypeString(v string) (uint8, uint8) {
	oType := constant.ObjTypeString
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return oType, constant.ObjEncodingInt
	}
	return oType, constant.ObjEncodingRaw
}
