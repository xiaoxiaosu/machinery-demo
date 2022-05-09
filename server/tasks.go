package server

import (
	"github.com/RichardKnop/machinery/v1/log"
)

func push(pushToken, pushContent string) (bool, error) {
	// 模拟push操作
	log.INFO.Printf("push to %s,the content is %s", pushToken, pushContent)

	return true, nil
}

func afterPush(pushResult ...bool) (bool, error) {
	for _, v := range pushResult {
		if v != true {
			log.INFO.Printf("not the all push get success,then the tx rollback")
			return false, nil
		}
	}

	log.INFO.Printf("all push get success,then the tx continue")

	return true, nil
}

func business(businessId int, auditResult bool) (bool, error) {
	if !auditResult {
		return false, nil
	}

	// 模拟处理交易
	log.INFO.Printf("deal business %v", businessId)

	return true, nil
}

func audit(businessId int) (bool, error) {
	// 模拟处理交易
	log.INFO.Printf("deal audit %v", businessId)

	return true, nil
}
