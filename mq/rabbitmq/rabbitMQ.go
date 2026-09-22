package rabbitmq

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	log "log/slog"
	"math/rand"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	amqpGo "github.com/rabbitmq/amqp091-go"
)

const maxReconnectDelay = 60 * time.Second

type RabbitMQ struct {
	connected        atomic.Bool
	queueName        string
	consumerFunction func(consumer amqpGo.Delivery) bool
	//
	config     *ConfigRabbitMQ
	mu         sync.Mutex
	mqConnect  *amqpGo.Connection
	mqChannel  *amqpGo.Channel
	mqQueue    amqpGo.Queue
	mqMessages <-chan amqpGo.Delivery
	////////
	retryDelay     time.Duration
	confirmTimeout time.Duration
	args           amqpGo.Table
	ctx            context.Context
	cancel         context.CancelFunc
	parentCtx      context.Context // reserved
	stopped        bool
	confirms       bool
}
type MQFunc func(delivery amqpGo.Delivery) bool

func NewRabbitMQ(configRabbitMQ *ConfigRabbitMQ, queueName string, consumerFunction func(consumer amqpGo.Delivery) bool) *RabbitMQ {
	cfg := *configRabbitMQ
	args := amqpGo.Table{}
	if cfg.Quorum {
		args["x-queue-type"] = "quorum"
	}
	if cfg.DeadLetterExchange != "" {
		args["x-dead-letter-exchange"] = cfg.DeadLetterExchange
	}
	if len(args) == 0 {
		args = nil
	}
	var qName string
	if queueName == "" {
		qName = cfg.Queue
	} else {
		qName = queueName
	}
	retryD, err := time.ParseDuration(cfg.RetryDelay)
	if err != nil {
		log.Error("invalid RetryDelay", log.String("RetryDelay", cfg.RetryDelay), log.String("Message", err.Error()))
	}
	if retryD <= 0 {
		retryD = time.Second
	}
	confirmD, err := time.ParseDuration(cfg.ConfirmTimeout)
	if err != nil {
		log.Error("invalid ConfirmTimeout", log.String("ConfirmTimeout", cfg.ConfirmTimeout), log.String("Message", err.Error()))
	}
	if confirmD <= 0 {
		confirmD = 10 * time.Second
	}
	if cfg.ConcurrencyCount < 1 {
		log.Warn("ConcurrencyCount < 1, using 1", log.Int("ConcurrencyCount", cfg.ConcurrencyCount))
		cfg.ConcurrencyCount = 1
	}
	return &RabbitMQ{
		consumerFunction: consumerFunction,
		config:           &cfg,
		retryDelay:       retryD,
		confirmTimeout:   confirmD,
		queueName:        qName,
		args:             args,
	}
}

func (r *RabbitMQ) GetConf() *ConfigRabbitMQ {
	return r.config
}
func (r *RabbitMQ) GetName() string {
	return "RabbitMQ"
}

func (r *RabbitMQ) Start(parentCtx context.Context) (err error) {
	r.ctx, r.cancel = context.WithCancel(parentCtx)
	r.parentCtx = parentCtx
	r.mu.Lock()
	r.stopped = false
	r.mu.Unlock()
	if !r.config.Disabled {
		log.Debug("Start", log.String("Queue", r.queueName), log.Int("ConcurrencyCount", r.config.ConcurrencyCount), log.Bool("IsQuorum", r.config.Quorum))
		if r.config.Url != "" {
			r.connectAsync()
		} else {
			return errors.New("empty url")
		}
	} else {
		log.Warn("Rabbit MQ disabled")
	}
	return nil
}

func (r *RabbitMQ) Stop() (err error) {
	if r.cancel != nil {
		log.Debug("Cancelling")
		r.cancel()
	}
	if !r.config.Disabled {
		r.mu.Lock()
		r.stopped = true
		ch, conn := r.mqChannel, r.mqConnect
		r.mqChannel, r.mqConnect, r.mqMessages = nil, nil, nil
		r.mu.Unlock()
		if ch != nil {
			if cerr := ch.Close(); cerr != nil {
				log.Error("mqChannel.Close", log.String("Message", cerr.Error()))
				err = cerr
			}
		}
		if conn != nil {
			if cerr := conn.Close(); cerr != nil {
				log.Error("mqConnect.Close", log.String("Message", cerr.Error()))
				if err == nil {
					err = cerr
				}
			}
		}
	} else {
		log.Warn("Rabbit MQ disabled")
	}
	return nil
}

func (r *RabbitMQ) backoffDelay(attempt int) time.Duration {
	d := r.retryDelay
	for i := 0; i < attempt && d < maxReconnectDelay; i++ {
		d *= 2
	}
	if d > maxReconnectDelay {
		d = maxReconnectDelay
	}
	return time.Duration(float64(d) * (0.5 + rand.Float64()))
}

func (r *RabbitMQ) connectAsync() {
	log.Debug("connectAsync")
	go func() {
		attempt := 0
		for {
			select {
			case <-r.ctx.Done():
				log.Debug("Done.")
				return
			default:
			}
			if err := r.connect(); err != nil {
				log.Error("Connection", log.String("Message", err.Error()))
			} else {
				attempt = 0
			}
			if r.connected.Load() {
				notifyClose := make(chan *amqpGo.Error, 1)
				notifyBlocked := make(chan amqpGo.Blocking, 1)
				r.mu.Lock()
				conn := r.mqConnect
				r.mu.Unlock()
				if conn != nil {
					conn.NotifyClose(notifyClose)
					conn.NotifyBlocked(notifyBlocked)
				}
				select {
				case <-notifyClose:
					log.Error("NotifyClose....")
					r.connected.Store(false)
				case <-notifyBlocked:
					log.Error("NotifyBlocked....")
					r.connected.Store(false)
				case <-r.ctx.Done():
					log.Debug("Done.")
					return
				}
			}
			log.Debug("Trying to restart")
			delay := r.backoffDelay(attempt)
			attempt++
			select {
			case <-r.ctx.Done():
				log.Debug("Done.")
				return
			case <-time.After(delay):
			}
		}
	}()
}

func (r *RabbitMQ) connect() (err error) {
	if err = r.dial(); err == nil {
		if err = r.channel(); err == nil {
			r.setupReturns()
			r.enableConfirms()
			if err = r.queue(); err == nil {
				r.connected.Store(true)
				log.Debug("connected")
			}
		}
	} else {
		log.Error("connect", log.String("Message", err.Error()))
	}
	return err
}

func (r *RabbitMQ) closeConnLocked() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mqChannel != nil {
		if cerr := r.mqChannel.Close(); cerr != nil {
			log.Debug("mqChannel.Close", log.String("Message", cerr.Error()))
		}
		r.mqChannel = nil
	}
	if r.mqConnect != nil {
		if cerr := r.mqConnect.Close(); cerr != nil {
			log.Debug("mqConnect.Close", log.String("Message", cerr.Error()))
		}
		r.mqConnect = nil
	}
	r.mqMessages = nil
}

func (r *RabbitMQ) dial() (err error) {
	r.closeConnLocked()
	var conn *amqpGo.Connection
	if strings.HasPrefix(strings.ToLower(r.config.Url), "amqps") {
		log.Debug("DialTLS")
		conn, err = amqpGo.DialTLS(r.config.Url, new(tls.Config))
	} else {
		log.Debug("Dial NON TLS")
		conn, err = amqpGo.Dial(r.config.Url)
	}
	if err == nil {
		r.mu.Lock()
		if r.stopped {
			r.mu.Unlock()
			_ = conn.Close()
			return errors.New("rabbitmq is stopped")
		}
		r.mqConnect = conn
		r.mu.Unlock()
	}
	return err
}

func (r *RabbitMQ) channel() (err error) {
	log.Debug("attempting to open channel")
	r.mu.Lock()
	conn := r.mqConnect
	r.mu.Unlock()
	if conn == nil {
		return errors.New("no connection")
	}
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mqConnect != conn {
		_ = ch.Close()
		return errors.New("connection changed")
	}
	r.mqChannel = ch
	return nil
}

func (r *RabbitMQ) setupReturns() {
	r.mu.Lock()
	ch := r.mqChannel
	r.mu.Unlock()
	if ch == nil {
		return
	}
	retCh := ch.NotifyReturn(make(chan amqpGo.Return, 1))
	go func() {
		for ret := range retCh {
			log.Warn("Publish returned by broker",
				log.Int("ReplyCode", int(ret.ReplyCode)),
				log.String("ReplyText", ret.ReplyText),
				log.String("Exchange", ret.Exchange),
				log.String("RoutingKey", ret.RoutingKey))
		}
	}()
}

func (r *RabbitMQ) enableConfirms() {
	r.mu.Lock()
	ch := r.mqChannel
	r.mu.Unlock()
	if ch == nil {
		return
	}
	if err := ch.Confirm(false); err != nil {
		log.Warn("channel.Confirm failed, publishing without confirms", log.String("Message", err.Error()))
		return
	}
	r.mu.Lock()
	r.confirms = true
	r.mu.Unlock()
}

func (r *RabbitMQ) queue() (err error) {
	if r.queueName == "" {
		log.Info("No queue")
		return nil // in case if we didn't need channel
	}
	r.mu.Lock()
	ch := r.mqChannel
	r.mu.Unlock()
	if ch == nil {
		return errors.New("no channel")
	}
	q, err := ch.QueueDeclare(
		r.queueName, // name of the queue
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // noWait
		r.args,      // arguments
	)
	if err != nil {
		return err
	}
	r.mu.Lock()
	if r.mqChannel != ch {
		r.mu.Unlock()
		return errors.New("channel changed")
	}
	r.mqQueue = q
	r.mu.Unlock()
	log.Info("Queue is ready", log.String("Queue", r.queueName))
	return r.initConsumer(r.consumerFunction, r.config.ConcurrencyCount)
}

func (r *RabbitMQ) initConsumer(consumerFunction func(consumer amqpGo.Delivery) bool, concurrencyCount int) (err error) {
	log.Debug("initConsumer")
	if consumerFunction == nil {
		log.Warn("consumerFunction is null")
		return nil
	}
	if concurrencyCount < 1 {
		concurrencyCount = 1
	}
	r.mu.Lock()
	ch := r.mqChannel
	r.mu.Unlock()
	if ch == nil {
		return errors.New("no channel")
	}
	if err = ch.Qos(concurrencyCount, 0, false); err != nil {
		log.Error("Qos", log.String("message", err.Error()))
		return err
	}
	var messages <-chan amqpGo.Delivery
	messages, err = ch.Consume(
		r.mqQueue.Name,    // mqQueue
		r.config.Consumer, // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.mqMessages = messages
	r.mu.Unlock()
	log.Debug("Ready to consume", log.Int("Concurrency count", concurrencyCount))
	workCh := make(chan amqpGo.Delivery, concurrencyCount)
	for w := 0; w < concurrencyCount; w++ {
		go func() {
			for message := range workCh {
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							log.Error("consumer worker panic",
								log.Any("panic", rec),
								log.String("stack", string(debug.Stack())))
						}
					}()
					if consumerFunction(message) {
						if err := message.Ack(false); err != nil {
							log.Error("Ack", log.String("message", err.Error()))
						}
					} else {
						if err := message.Nack(false, false); err != nil {
							log.Error("Nack", log.String("message", err.Error()))
						}
					}
				}()
			}
		}()
	}
	go func() {
		for message := range r.mqMessages {
			workCh <- message
		}
		close(workCh)
	}()
	return nil
}

func (r *RabbitMQ) ProduceData(queueName string, inData interface{}) error {
	if !r.config.Disabled {
		if data, err := json.Marshal(inData); err == nil {
			// log.Debug("ProduceData", log.String("data", string(data)))
			return r.Produce(queueName, &data)
		} else {
			return err
		}
	} else {
		log.Warn("Rabbit MQ disabled")
		return nil
	}
}

func (r *RabbitMQ) Produce(queueName string, data *[]byte) error {
	if !r.connected.Load() {
		return errors.New("not connected")
	}
	r.mu.Lock()
	ch := r.mqChannel
	confirms := r.confirms
	r.mu.Unlock()
	if ch == nil {
		return errors.New("not connected")
	}
	publishing := amqpGo.Publishing{
		Priority:     0,                 // 0 to 9
		DeliveryMode: amqpGo.Persistent, // Transient (0 or 1) or Persistent (2)
		// ContentEncoding: "UTF-8",
		// ContentType:     "application/protobuf", //"application/json"
		Body: *data,
	}
	if !confirms {
		return ch.PublishWithContext(r.ctx,
			r.config.Exchange, // exchange
			queueName,         // routing key
			true,              // mandatory
			false,             // immediate
			publishing)
	}
	dc, err := ch.PublishWithDeferredConfirm(
		r.config.Exchange, // exchange
		queueName,         // routing key
		true,              // mandatory
		false,             // immediate
		publishing)
	if err != nil {
		return err
	}
	select {
	case <-dc.Done():
		if !dc.Acked() {
			return errors.New("publisher confirm nack")
		}
		return nil
	case <-time.After(r.confirmTimeout):
		return errors.New("publisher confirm timeout")
	case <-r.ctx.Done():
		return r.ctx.Err()
	}
}

func (r *RabbitMQ) IsConnected() bool {
	return r.connected.Load()
}

// https://medium.com/@eugenfedchenko/rpc-over-rabbitmq-golang-ff3d2b312a69
