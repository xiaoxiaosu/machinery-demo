

# 业务架构

![image](https://user-images.githubusercontent.com/5572372/167400120-6b84824c-e503-45a9-9098-b46b503b06e4.png)


```mermaid
sequenceDiagram

participant producer  as producer
participant cserver  as 发布端server
participant broker  as broker
participant backend as backend
participant wserver as 消费端server
participant worker  as worker

worker ->> wserver: 消费(lanuch)
producer ->> cserver: 发布任务(publish)
cserver ->> broker: 将任务转为json，发送一条消息到broker
cserver ->> backend: 将任务状态置为PENDING
cserver -->> producer: asyncResult
broker ->> wserver: 将消息推送给server
wserver ->> backend: 将任务状态置为RECEIVED
wserver ->> wserver: 获取到消息从registerdTasks中获取到任务
wserver ->> backend: 将任务状态置为STARTED
wserver ->> wserver: 执行任务
wserver ->> backend: 将任务状态置为SUCCESS/FAILED
producer ->> cserver: 获取asyncResult
cserver ->> backend: 阻塞获取result
backend -->> cserver: return result
cserver -->> producer: return result
```

# 任务类型

## 普通任务

**使用场景**

适用于需要对业务做异步解耦的场景，例：文件的导出，消息的推送等

**示例代码**

```go
package main

import ...

func main() {
	// 初始化配置
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
	
	// 实例化servere
	server, _ := machinery.NewServer(cnf)
	
	// 发送推送任务
	server.SendTask(&tasks.Signature{
		Name: "push",
		Args: []tasks.Arg{
			{Name: "pushToken", Type: "string", Value: "acc01xxxcfu1"},
			{Name: "pushContent", Type: "string", Value: "hello,world"},
		},
	})
}
```

## 并发任务

**使用场景**

适用于多个任务之间没有依赖关系的场景，例：同时发送多条推送

**示例代码**

```go
package main

import ...

func main() {
	
	... // init cnf
	server, _ := machinery.NewServer(cnf)

	group, _ := tasks.NewGroup(
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "acc01xxxcfu1"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "x00c1acufl1h"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "c1s20cgalkj"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
	)

	server.SendGroup(group, len(group.Tasks))
}
```

## 回调任务

**使用场景**

适用于某一个任务依赖于一批任务结果的场景，例：5个审批都完成以后触发交易状态的维护

**示例代码**

```go
package main

import ...

func main() {
	
	... // init cnf

	server, _ := machinery.NewServer(cnf)

	group, _ := tasks.NewGroup(
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "acc01xxxcfu1"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "x00c1acufl1h"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "c1s20cgalkj"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
	)

	chord, _ := tasks.NewChord(group, &tasks.Signature{
		Name: "afterPush",
	})

	server.SendChord(chord, len(group.Tasks))
}
```

## 链式任务

**使用场景**

适用于后一个任务依赖于前一个任务结果的场景，例：审批完成后触发交易

**示例代码**

```go
package main

import "fmt"

func main() {
	
	... // init cnf
	server, _ := machinery.NewServer(cnf)

	chain, _ := tasks.NewChain(
		&tasks.Signature{
			Name: "audit",
			Args: []tasks.Arg{
				{Name: "business_id", Type: "int", Value: 1},
			},
		},
		&tasks.Signature{
			Name: "business",
			Args: []tasks.Arg{
				{Name: "business_id", Type: "int", Value: 1},
			},
		},
	)

	server.SendChain(chain)
}
```

# 源码分析

## 核心组件

### server

> machinery服务器
> 

```go
type Server struct {
	config            *config.Config // 配置信息
	registeredTasks   *sync.Map // 注册的任务
	broker            brokersiface.Broker // 消息存储器
	backend           backendsiface.Backend // 任务维护期
	lock              lockiface.Lock // 锁
	scheduler         *cron.Cron // machinery调度器
	prePublishHandler func(*tasks.Signature) // 任务发布前置触发函数
}
```

### broker

> 实现了broker接口的对象，即可用于发布任务的操作
> 

```go
type Broker interface {
	GetConfig() *config.Config // 获取broker配置
	SetRegisteredTaskNames(names []string) // 注册任务
	IsTaskRegistered(name string) bool // 判断任务是否注册
	StartConsuming(consumerTag string, concurrency int, p TaskProcessor) (bool, error) // 开始消费
	StopConsuming() // 停止消费
	Publish(ctx context.Context, task *tasks.Signature) error // 发布任务
	GetPendingTasks(queue string) ([]*tasks.Signature, error) // 获取等待中的任务
	GetDelayedTasks() ([]*tasks.Signature, error) // 获取延迟任务
	AdjustRoutingKey(s *tasks.Signature) // 校准routingkey
}
```

### backend

> 实现了backend接口的对象，即可用于维护任务状态的操作
> 

```go
type Backend interface {
	InitGroup(groupUUID string, taskUUIDs []string) error // 用于存储group的原信息
	GroupCompleted(groupUUID string, groupTaskCount int) (bool, error) // group完成状态
	GroupTaskStates(groupUUID string, groupTaskCount int) ([]*tasks.TaskState, error) // group任务完成状态
	TriggerChord(groupUUID string) (bool, error) // 是否触发回调

	SetStatePending(signature *tasks.Signature) error // 将任务状态设置为pending
	SetStateReceived(signature *tasks.Signature) error // 将任务状态设置为received
	SetStateStarted(signature *tasks.Signature) error // 将任务状态设置为started
	SetStateRetry(signature *tasks.Signature) error // 将任务状态设置为retry
	SetStateSuccess(signature *tasks.Signature, results []*tasks.TaskResult) error // 将任务状态设置为success
	SetStateFailure(signature *tasks.Signature, err string) error // 将任务状态设置为failure
	GetState(taskUUID string) (*tasks.TaskState, error) // 获取任务状态
}
```

### singature

> 任务结构体
> 

```go
// Signature represents a single task invocation
type Signature struct {
	UUID           string // 位移id
	Name           string // 任务名称
	RoutingKey     string // routingkey 用于和队列做绑定
	ETA            *time.Time // 执行时间
	GroupUUID      string // 组唯一id
	GroupTaskCount int // 组任务数量
	Args           []Arg // 参数
	Headers        Headers // 头信息
	Priority       uint8 // 优先级[仅适用于rabbitMq]
	Immutable      bool // 可变性 默认false，执行回调时判断
	RetryCount     int // 重试次数
	RetryTimeout   int // 重试时间
	OnSuccess      []*Signature // 成功时触发任务
	OnError        []*Signature // 失败时触发任务
	ChordCallback  *Signature // 回调任务
}
```

### worker

> 消费者结构体
> 

```go
type Worker struct {
	server            *Server // 关联的server
	ConsumerTag       string // 消费者唯一标识
	Concurrency       int // 消费时最高并发数控制
	Queue             string // 消费队列
	errorHandler      func(err error) // 错误时触发函数
	preTaskHandler    func(*tasks.Signature) // 任务执行前置触发函数
	postTaskHandler   func(*tasks.Signature) // 任务执行完成触发函数
}
```

## 创建服务

**NewServer**

```go
func NewServer(cnf *config.Config) (*Server, error) {
	// 实例化broker
	broker, err := BrokerFactory(cnf)
	if err != nil {
		return nil, err
	}

	// 实例化backend
	backend, _ := BackendFactory(cnf)

	...

	// 实例化server
	srv := NewServerWithBrokerBackendLock(cnf, broker, backend, lock)

	...

	return srv, nil
}
```

**BrokerFactory**

```go
func BrokerFactory(cnf *config.Config) (brokeriface.Broker, error) {
	if strings.HasPrefix(cnf.Broker, "amqp://") {
		return amqpbroker.New(cnf), nil
	}
	...

	return nil, fmt.Errorf("Factory failed with broker URL: %v", cnf.Broker)
}
```

**amqpbroker.New**

```go
func New(cnf *config.Config) iface.Broker {
	return &Broker{Broker: common.NewBroker(cnf), AMQPConnector: common.AMQPConnector{}, connections: make(map[string]*AMQPConnection)}
}
```

**NewServerWithBrokerBackendLock**

```go
func NewServerWithBrokerBackendLock(cnf *config.Config, brokerServer brokersiface.Broker, backendServer backendsiface.Backend, lock lockiface.Lock) *Server {
	srv := &Server{
		config:          cnf,
		registeredTasks: new(sync.Map),
		broker:          brokerServer,
		backend:         backendServer,
		lock:            lock,
		scheduler:       cron.New(),
	}

	// Run scheduler job
	go srv.scheduler.Run()

	return srv
}
```

## 注册任务

**RegisterTasks**

```go
func (server *Server) RegisterTasks(namedTaskFuncs map[string]interface{}) error {
	for _, task := range namedTaskFuncs {
		if err := tasks.ValidateTask(task); err != nil {
			return err
		}
	}

	for k, v := range namedTaskFuncs {
		server.registeredTasks.Store(k, v)
	}

	server.broker.SetRegisteredTaskNames(server.GetRegisteredTaskNames())
	return nil
}
```

**broker.SetRegisteredTaskNames**

```go
func (b *Broker) SetRegisteredTaskNames(names []string) {
	b.registeredTaskNames.Lock()
	defer b.registeredTaskNames.Unlock()
	b.registeredTaskNames.items = names
}
```

## 发布任务

### 普通任务

**SendTask**

```go
func (server *Server) SendTask(signature *tasks.Signature) (*result.AsyncResult, error) {
	return server.SendTaskWithContext(context.Background(), signature)
}
```

**SendTaskWithContext**

```go
func (server *Server) SendTaskWithContext(ctx context.Context, signature *tasks.Signature) (*result.AsyncResult, error) {

	...

	// 给任务创建一个唯一标识
	if signature.UUID == "" {
		taskID := uuid.New().String()
		signature.UUID = fmt.Sprintf("task_%v", taskID)
	}

	...
	
	// 执行前置方法
	if server.prePublishHandler != nil {
		server.prePublishHandler(signature)
	}
	
	// 通过broker发布一个task
	if err := server.broker.Publish(ctx, signature); err != nil {
		return nil, fmt.Errorf("Publish message error: %s", err)
	}
	
	// 返回一个异步的result对象
	return result.NewAsyncResult(signature, server.backend), nil
}
```

**amqp.Publish**

```go
func (b *Broker) Publish(ctx context.Context, signature *tasks.Signature) error {
	// 确保消息有routing key，没有的话分两种情况，exchange是direct时就赋值为config.BindingKey，否则赋值为config.defaultQueue
	b.AdjustRoutingKey(signature)
	
	// 将任务转为json
	msg, err := json.Marshal(signature)
	if err != nil {
		return fmt.Errorf("JSON marshal error: %s", err)
	}

	// 如果是延迟消息，调用broker.delay延迟发送
	if signature.ETA != nil {
		now := time.Now().UTC()

		if signature.ETA.After(now) {
			delayMs := int64(signature.ETA.Sub(now) / time.Millisecond)

			return b.delay(signature, delayMs)
		}
	}
	
	// 通过配置获取到队列
	queue := b.GetConfig().DefaultQueue
	// 获取bindingKey
	bindingKey := b.GetConfig().AMQP.BindingKey

	if b.isDirectExchange() { //对于directExchange，queue和bindingKey都为config.BindingKey
		queue = signature.RoutingKey
		bindingKey = signature.RoutingKey
	}
	
	// 创建连接
	connection, err := b.GetOrOpenConnection(
		queue,
		bindingKey, // queue binding key
		nil,        // exchange declare args
		amqp.Table(b.GetConfig().AMQP.QueueDeclareArgs), // queue declare args
		amqp.Table(b.GetConfig().AMQP.QueueBindingArgs), // queue binding args
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to get a connection for queue %s", queue)
	}

	channel := connection.channel
	confirmsChan := connection.confirmation
	
	// 发布消息
	if err := channel.Publish(
		b.GetConfig().AMQP.Exchange, // exchange name
		signature.RoutingKey,        // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			Headers:      amqp.Table(signature.Headers),
			ContentType:  "application/json",
			Body:         msg,
			Priority:     signature.Priority,
			DeliveryMode: amqp.Persistent,
		},
	); err != nil {
		return errors.Wrap(err, "Failed to publish task")
	}

	confirmed := <-confirmsChan
	// ack机制，保证消息已经收到
	if confirmed.Ack {
		return nil
	}

	return fmt.Errorf("Failed delivery of delivery tag: %v", confirmed.DeliveryTag)
}
```

**result.NewAsyncResult**

```go
func NewAsyncResult(signature *tasks.Signature, backend iface.Backend) *AsyncResult {
	return &AsyncResult{
		Signature: signature,
		taskState: new(tasks.TaskState),
		backend:   backend,
	}
}
```

### 并发任务

```go
func sendParallelTask() {
	s, err := server.StartServer()
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	group, err := tasks.NewGroup(
		server.CreatePushTask("zhangsan", "hell world"),
		server.CreatePushTask("lisi", "hell world"),
		server.CreatePushTask("wangwu", "hell world"),
	)
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	asyncResults, err := s.SendGroup(group, 3)
	if err != nil {
		log.ERROR.Printf("%v", err)
	}

	for _, results := range asyncResults {
		result, err := results.Get(time.Duration(time.Millisecond * 5))
		if err != nil {
			log.ERROR.Printf("%v", err)
		}

		log.INFO.Printf("%v", tasks.HumanReadableResults(result))
	}
}
```

**NewGroup**

```go
func NewGroup(signatures ...*Signature) (*Group, error) {

	groupUUID := uuid.New().String()
	groupID := fmt.Sprintf("group_%v", groupUUID)

	for _, signature := range signatures {
		if signature.UUID == "" {
			signatureID := uuid.New().String()
			signature.UUID = fmt.Sprintf("task_%v", signatureID)
		}
		signature.GroupUUID = groupID
		signature.GroupTaskCount = len(signatures)
	}

	return &Group{
		GroupUUID: groupID,
		Tasks:     signatures,
	}, nil
}
```

**sendGroupContext**

```go
func (server *Server) SendGroupWithContext(ctx context.Context, group *tasks.Group, sendConcurrency int) ([]*result.AsyncResult, error) {
	
	...
	
	// 创建一个切片用于存储任务结果
	asyncResults := make([]*result.AsyncResult, len(group.Tasks))

	var wg sync.WaitGroup
	wg.Add(len(group.Tasks))
	errorsChan := make(chan error, len(group.Tasks)*2)

	// 初始化任务组
	server.backend.InitGroup(group.GroupUUID, group.GetUUIDs())

	// 将任务状态置为pendding
	for _, signature := range group.Tasks {
		if err := server.backend.SetStatePending(signature); err != nil {
			errorsChan <- err
			continue
		}
	}
	
	// 根据配置的并发数，创建一个令牌池，允许每次最多n个任务同时发布
	pool := make(chan struct{}, sendConcurrency)
	go func() {
		for i := 0; i < sendConcurrency; i++ {
			pool <- struct{}{}
		}
	}()
	
	for i, signature := range group.Tasks {

		if sendConcurrency > 0 {
			<-pool
		}
		
		// 创建一个gorotuine
		go func(s *tasks.Signature, index int) {
			defer wg.Done()
			err := server.broker.Publish(ctx, s)
			
			// 任务发布完成，向令牌池中放回一个令牌
			if sendConcurrency > 0 {
				pool <- struct{}{}
			}

			if err != nil {
				errorsChan <- fmt.Errorf("Publish message error: %s", err)
				return
			}

			asyncResults[index] = result.NewAsyncResult(s, server.backend)
		}(signature, i)
	}

	done := make(chan int)
	go func() {
		wg.Wait()
		done <- 1
	}()
	
	
	select {
	case err := <-errorsChan:
		return asyncResults, err
	case <-done: // 全部执行完成以后返回
		return asyncResults, nil
	}
}
```

### 回调任务

**tasks.NewChord**

```go
func NewChord(group *Group, callback *Signature) (*Chord, error) {
	// 设置唯一id
	if callback.UUID == "" {
		// Generate a UUID for the chord callback
		callbackUUID := uuid.New().String()
		callback.UUID = fmt.Sprintf("chord_%v", callbackUUID)
	}

	// 给每个任务都加上一个回调任务,由worker去调用
	for _, signature := range group.Tasks {
		signature.ChordCallback = callback
	}

	return &Chord{Group: group, Callback: callback}, nil
}
```

**server.sendChord**

```go
func (server *Server) SendChordWithContext(ctx context.Context, chord *tasks.Chord, sendConcurrency int) (*result.ChordAsyncResult, error) {
	...

	_, err := server.SendGroupWithContext(ctx, chord.Group, sendConcurrency)
	if err != nil {
		return nil, err
	}

	return result.NewChordAsyncResult(
		chord.Group.Tasks,
		chord.Callback,
		server.backend,
	), nil
}
```

### 链式任务

**tasks.NewChain**

```go
func NewChain(signatures ...*Signature) (*Chain, error) {
	// 为每个任务创建唯一id
	for _, signature := range signatures {
		if signature.UUID == "" {
			signatureID := uuid.New().String()
			signature.UUID = fmt.Sprintf("task_%v", signatureID)
		}
	}
	
	// 将后一个任务放入前一个任务的OnSuccess中
	for i := len(signatures) - 1; i > 0; i-- {
		if i > 0 {
			signatures[i-1].OnSuccess = []*Signature{signatures[i]}
		}
	}

	chain := &Chain{Tasks: signatures}

	return chain, nil
}
```

**server.SendChain**

```go
func (server *Server) SendChain(chain *tasks.Chain) (*result.ChainAsyncResult, error) {
	_, err := server.SendTask(chain.Tasks[0])
	if err != nil {
		return nil, err
	}

	return result.NewChainAsyncResult(chain.Tasks, server.backend), nil
}
```

### 延迟任务

**amqp.delay**

```go
func (b *Broker) delay(signature *tasks.Signature, delayMs int64) error {
	
	... // 校验工作

	// 创建一个延迟队列
	queueName := fmt.Sprintf(
		"delay.%d.%s.%s",
		delayMs, // delay duration in mileseconds
		b.GetConfig().AMQP.Exchange,
		signature.RoutingKey, // routing key
	)
	declareQueueArgs := amqp.Table{
		// Exchange where to send messages after TTL expiration.
		"x-dead-letter-exchange": b.GetConfig().AMQP.Exchange,
		// Routing key which use when resending expired messages.
		"x-dead-letter-routing-key": signature.RoutingKey,
		// Time in milliseconds
		// after that message will expire and be sent to destination.
		"x-message-ttl": delayMs,
		// Time after that the queue will be deleted.
		"x-expires": delayMs * 2,
	}
	conn, channel, _, _, _, err := b.Connect(
		b.GetConfig().Broker,
		b.GetConfig().MultipleBrokerSeparator,
		b.GetConfig().TLSConfig,
		b.GetConfig().AMQP.Exchange,     // exchange name
		b.GetConfig().AMQP.ExchangeType, // exchange type
		queueName,                       // queue name
		true,                            // queue durable
		b.GetConfig().AMQP.AutoDelete,   // queue delete when unused
		queueName,                       // queue binding key
		nil,                             // exchange declare args
		declareQueueArgs,                // queue declare args
		amqp.Table(b.GetConfig().AMQP.QueueBindingArgs), // queue binding args
	)
	if err != nil {
		return err
	}

	defer b.Close(channel, conn)
	
	// 发布消息到延迟队列
	if err := channel.Publish(
		b.GetConfig().AMQP.Exchange, // exchange
		queueName,                   // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			Headers:      amqp.Table(signature.Headers),
			ContentType:  "application/json",
			Body:         message,
			DeliveryMode: amqp.Persistent,
		},
	); err != nil {
		return err
	}

	return nil
}
```

## 消费任务

### 创建消费者

**server.NewWorker**

```go
func (server *Server) NewWorker(consumerTag string, concurrency int) *Worker {
	return &Worker{
		server:      server,
		ConsumerTag: consumerTag,
		Concurrency: concurrency,
		Queue:       "",
	}
}
```

### 执行消费

**worker.Lanuch**

```go
func (worker *Worker) LaunchAsync(errorsChan chan<- error) {
	// 获取配置和broker
	cnf := worker.server.GetConfig()
	broker := worker.server.GetBroker()

	... // 记录一些log

	var signalWG sync.WaitGroup
	// 开一个gorotuine让broker执行消费
	go func() {
		for {
			retry, err := broker.StartConsuming(worker.ConsumerTag, worker.Concurrency, worker)
			if retry { 
				if worker.errorHandler != nil {
					worker.errorHandler(err)
				} else {
					log.WARNING.Printf("Broker failed with error: %s", err)
				}
			} else {
				signalWG.Wait()
				errorsChan <- err // stop the goroutine
				return
			}
		}
	}()
	... // 处理信号，用于结束消费
}
```

**amqp.StartConsuming**

```go
func (b *Broker) StartConsuming(consumerTag string, concurrency int, taskProcessor iface.TaskProcessor) (bool, error) {

	// 获取消费的队列名，如果没有指定就用配置里的队列名
	queueName := taskProcessor.CustomQueue()
	if queueName == "" {
		queueName = b.GetConfig().DefaultQueue
	}
	
  // 连接broker
	conn, channel, queue, _, amqpCloseChan, err := b.Connect(
		b.GetConfig().Broker,
		b.GetConfig().MultipleBrokerSeparator,
		b.GetConfig().TLSConfig,
		b.GetConfig().AMQP.Exchange,     // exchange name
		b.GetConfig().AMQP.ExchangeType, // exchange type
		queueName,                       // queue name
		true,                            // queue durable
		false,                           // queue delete when unused
		b.GetConfig().AMQP.BindingKey,   // queue binding key
		nil,                             // exchange declare args
		amqp.Table(b.GetConfig().AMQP.QueueDeclareArgs), // queue declare args
		amqp.Table(b.GetConfig().AMQP.QueueBindingArgs), // queue binding args
	)
	if err != nil {
		b.GetRetryFunc()(b.GetRetryStopChan())
		return b.GetRetry(), err
	}
	defer b.Close(channel, conn)
	
	...	
	
	// rabbitMq执行消费
	deliveries, err := channel.Consume(
		queue.Name,  // queue
		consumerTag, // consumer tag
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // arguments
	)

	// rabbitMq消费失败时，进行重试
	if err != nil {
		return b.GetRetry(), fmt.Errorf("Queue consume error: %s", err)
	}

	log.INFO.Print("[*] Waiting for messages. To exit press CTRL+C")
	
	// 拿到消费的结果，broker进行消费
	if err := b.consume(deliveries, concurrency, taskProcessor, amqpCloseChan); err != nil {
		return b.GetRetry(), err
	}

	// Waiting for any tasks being processed to finish
	b.processingWG.Wait()

	return b.GetRetry(), nil
}
```

**amqp.consume**

```go
func (b *Broker) consume(deliveries <-chan amqp.Delivery, concurrency int, taskProcessor iface.TaskProcessor, amqpCloseChan <-chan *amqp.Error) error {
	pool := make(chan struct{}, concurrency)

	// 初始化一个令牌池，用于并发消费
	go func() {
		for i := 0; i < concurrency; i++ {
			pool <- struct{}{}
		}
	}()

	errorsChan := make(chan error, 1)

	for {
		select {
		case amqpErr := <-amqpCloseChan:
			return amqpErr
		case err := <-errorsChan:
			return err
		case d := <-deliveries:
			if concurrency > 0 {
				// 获取令牌
				<-pool
			}

			b.processingWG.Add(1)

			// Consume the task inside a gotourine so multiple tasks
			// can be processed concurrently
			go func() {
				// 执行消费
				if err := b.consumeOne(d, taskProcessor, true); err != nil {
					errorsChan <- err
				}

				b.processingWG.Done()

				if concurrency > 0 {
					// give worker back to pool
					pool <- struct{}{}
				}
			}()
		case <-b.GetStopChan():
			return nil
		}
	}
}
```

**amqp.consumeOne**

```go
func (b *Broker) consumeOne(delivery amqp.Delivery, taskProcessor iface.TaskProcessor, ack bool) error {
	... //校验消息

	var multiple, requeue = false, false

	// 将消息转为task signature
	signature := new(tasks.Signature)
	decoder := json.NewDecoder(bytes.NewReader(delivery.Body))
	decoder.UseNumber()
	if err := decoder.Decode(signature); err != nil {
		delivery.Nack(multiple, requeue)
		return errs.NewErrCouldNotUnmarshalTaskSignature(delivery.Body, err)
	}

	... //判断任务是否注册
	
	// 执行任务
	err := taskProcessor.Process(signature)
	if ack {
		delivery.Ack(multiple)
	}
	return err
}
```

**worker.Process**

```go
func (worker *Worker) Process(signature *tasks.Signature) error {
	
	... // 参数校验
	
	// 获取任务对应的方法
	taskFunc, err := worker.server.GetRegisteredTask(signature.Name)
	if err != nil {
		return nil
	}

	// 将任务状态置为received
	if err = worker.server.GetBackend().SetStateReceived(signature); err != nil {
		return fmt.Errorf("Set state to 'received' for task %s returned error: %s", signature.UUID, err)
	}

	... //  一些tracing初始化的操作

	// 将任务状态置为started
	if err = worker.server.GetBackend().SetStateStarted(signature); err != nil {
		return fmt.Errorf("Set state to 'started' for task %s returned error: %s", signature.UUID, err)
	}

	// 如果设置了preTaskHandler就先执行preTaskHandler
	if worker.preTaskHandler != nil {
		worker.preTaskHandler(signature)
	}

	// 如果设置了postTaskHandler就在最后通过defer执行postTaskHandler
	if worker.postTaskHandler != nil {
		defer worker.postTaskHandler(signature)
	}

	// 开始执行任务
	results, err := task.Call()
	if err != nil {
		... // 执行失败，根据配置的重试次数进行重试，依然失败则返回失败了

		return worker.taskFailed(signature, err)
	}
	
	// 返回成功
	return worker.taskSucceeded(signature, results)
}
```

**task.Call**

```go
func (t *Task) Call() (taskResults []*TaskResult, err error) {
	
	...

	defer func() {
		... 
		}
	}()

	args := t.Args

	if t.UseContext { // 如果任务携带了context，就把值拿出来合并到参数列表中
		ctxValue := reflect.ValueOf(t.Context)
		args = append([]reflect.Value{ctxValue}, args...)
	}

	// 执行任务
	results := t.TaskFunc.Call(args)

	// 任务至少需要含有一个返回值，否则返回错误
	if len(results) == 0 {
		return nil, ErrTaskReturnsNoValue
	}

	lastResult := results[len(results)-1]
	
	// 最后一个返回值不为空，认为是有error
	if !lastResult.IsNil() {
		... // error的反射处理，并返回error
	}

	// 将返回值组装为taskResult返回
	taskResults = make([]*TaskResult, len(results)-1)
	for i := 0; i < len(results)-1; i++ {
		val := results[i].Interface()
		typeStr := reflect.TypeOf(val).String()
		taskResults[i] = &TaskResult{
			Type:  typeStr,
			Value: val,
		}
	}

	return taskResults, err
}
```

**worker.taskSucceeded**

```go
func (worker *Worker) taskSucceeded(signature *tasks.Signature, taskResults []*tasks.TaskResult) error {
	// 将任务状态置位success
	if err := worker.server.GetBackend().SetStateSuccess(signature, taskResults); err != nil {
		return fmt.Errorf("Set state to 'success' for task %s returned error: %s", signature.UUID, err)
	}

	... // 日志记录

	
	// 遍历OnSuccess中的任务进行投递
	for _, successTask := range signature.OnSuccess {
		if signature.Immutable == false {
			// Pass results of the task to success callbacks
			for _, taskResult := range taskResults {
				successTask.Args = append(successTask.Args, tasks.Arg{
					Type:  taskResult.Type,
					Value: taskResult.Value,
				})
			}
		}

		worker.server.SendTask(successTask)
	}

	// 如果不存在任务组，直接返回
	if signature.GroupUUID == "" {
		return nil
	}

	// 如果不存在回调直接返回
	if signature.ChordCallback == nil {
		return nil
	}

	... // 任务组中所有任务状态检测，如果没有全部完成，直接返回

	// 从backend中获取回调任务是否已经被触发
	shouldTrigger, err := worker.server.GetBackend().TriggerChord(signature.GroupUUID)
	if err != nil {
		return fmt.Errorf("Triggering chord for group %s returned error: %s", signature.GroupUUID, err)
	}

	// 已经被触发则直接返回
	if !shouldTrigger {
		return nil
	}

	... // group中任务状态检查

	for _, taskState := range taskStates {
		if !taskState.IsSuccess() {
			return nil
		}

		if signature.ChordCallback.Immutable == false {
			 // 将当前任务的返回值合并到回调任务的参数中
			for _, taskResult := range taskState.Results {
				signature.ChordCallback.Args = append(signature.ChordCallback.Args, tasks.Arg{
					Type:  taskResult.Type,
					Value: taskResult.Value,
				})
			}
		}
	}

	// 投递回调任务
	_, err = worker.server.SendTask(signature.ChordCallback)
	if err != nil {
		return err
	}

	return nil
}
```
