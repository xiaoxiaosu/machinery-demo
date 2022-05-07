package server

import (
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
)

func StartServer() (*machinery.Server, error) {
	cnf := &config.Config{
		Broker:        "amqp://guest:guest@localhost:5672", // 消息存储
		DefaultQueue:  "tasks",
		ResultBackend: "amqp://guest:guest@localhost:5672",
		AMQP: &config.AMQPConfig{
			Exchange:     "tasks",
			ExchangeType: "direct",
		},
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	tasks := map[string]interface{}{
		"push":          push,
		"audit":         audit,
		"auditCallback": auditCallback,
	}

	return server, server.RegisterTasks(tasks)
}
