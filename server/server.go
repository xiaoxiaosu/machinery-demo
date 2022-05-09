package server

import (
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
)

func StartServer() (*machinery.Server, error) {
	cnf := &config.Config{
		Broker:        "amqp://guest:guest@localhost:5672", // 消息存储
		DefaultQueue:  "tasks",
		ResultBackend: "redis://localhost:6379",
		AMQP: &config.AMQPConfig{
			Exchange:     "tasks",
			ExchangeType: "direct",
		},
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			NormalTasksPollPeriod:  1000,
			DelayedTasksPollPeriod: 500,
		},
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	tasks := map[string]interface{}{
		"push":      push,
		"afterPush": afterPush,
		"audit":     audit,
		"business":  business,
	}

	return server, server.RegisterTasks(tasks)
}
