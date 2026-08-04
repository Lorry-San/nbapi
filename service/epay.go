package service

import (
	"github.com/Lorry-San/nbapi/setting/operation_setting"
	"github.com/Lorry-San/nbapi/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
