package server

import (
	"github.com/RichardKnop/machinery/v1/log"
	"github.com/RichardKnop/machinery/v1/tasks"
)

func push(token, content string) (bool, error) {
	// 模拟push操作
	log.INFO.Printf("push to %s,the content is %s", token, content)

	return true, nil
}

func CreatePushTask(token, content string) *tasks.Signature {
	return &tasks.Signature{
		Name: "push",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: token,
			},
			{
				Type:  "string",
				Value: content,
			},
		},
	}
}

func audit(userId uint64, businessId uint64) error {
	// 模拟审批操作
	log.INFO.Printf("there is a new audit,userid:%d,businessId:%d", userId, businessId)

	return nil
}

func CreateAuditTask(userId, businessId uint64) *tasks.Signature {
	return &tasks.Signature{
		Name: "audit",
		Args: []tasks.Arg{
			{
				Type:  "uint64",
				Value: userId,
			},
			{
				Type:  "uint64",
				Value: businessId,
			},
		},
	}
}

func auditCallback(args ...bool) (bool, error) {
	for _, v := range args {
		if v == false {
			return false, nil
		}
	}

	return true, nil
}

func CreateAuditCallback() *tasks.Signature {
	return &tasks.Signature{
		Name: "auditCallback",
	}
}
