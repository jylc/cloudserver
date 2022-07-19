package mq

import (
	"encoding/gob"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/aria2/rpc"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	TriggeredBy string
	Event       string
	Content     interface{}
}

type CallbackFunc func(message Message)

type MQ interface {
	rpc.Notifier

	Publish(string, Message)

	Subscribe(string, int) <-chan Message

	SubscribeCallback(string, CallbackFunc)

	Unsubscribe(string, <-chan Message)
}

var GlobalMQ = NewMQ()

func NewMQ() MQ {
	return &inMemoryMQ{
		topics:    make(map[string][]chan Message),
		callbacks: make(map[string][]CallbackFunc),
	}
}

func init() {
	gob.Register(Message{})
	gob.Register([]rpc.Event{})
}

type inMemoryMQ struct {
	topics    map[string][]chan Message
	callbacks map[string][]CallbackFunc
	sync.RWMutex
}

func (i *inMemoryMQ) OnDownloadStart(events []rpc.Event) {
	i.Aria2Notify(events, common.Downloading)
}

func (i *inMemoryMQ) OnDownloadPause(events []rpc.Event) {
	i.Aria2Notify(events, common.Paused)
}

func (i *inMemoryMQ) OnDownloadStop(events []rpc.Event) {
	i.Aria2Notify(events, common.Canceled)
}

func (i *inMemoryMQ) OnDownloadComplete(events []rpc.Event) {
	i.Aria2Notify(events, common.Complete)
}

func (i *inMemoryMQ) OnDownloadError(events []rpc.Event) {
	i.Aria2Notify(events, common.Error)
}

func (i *inMemoryMQ) OnBtDownloadComplete(events []rpc.Event) {
	i.Aria2Notify(events, common.Complete)
}

func (i *inMemoryMQ) Publish(topic string, message Message) {
	i.RLock()
	subscribersChan, okChan := i.topics[topic]
	subscribersCallback, okCallback := i.callbacks[topic]
	i.RUnlock()
	if okChan {
		go func(subscribersChan []chan Message) {
			for i := 0; i < len(subscribersChan); i++ {
				select {
				case subscribersChan[i] <- message:
				case <-time.After(time.Millisecond * 500):
				}
			}
		}(subscribersChan)
	}

	if okCallback {
		for i := 0; i < len(subscribersCallback); i++ {
			go subscribersCallback[i](message)
		}
	}
}

func (i *inMemoryMQ) Subscribe(topic string, buffer int) <-chan Message {
	ch := make(chan Message, buffer)
	i.Lock()
	i.topics[topic] = append(i.topics[topic], ch)
	i.Unlock()
	return ch
}

func (i *inMemoryMQ) SubscribeCallback(topic string, callbackFunc CallbackFunc) {
	i.Lock()
	i.callbacks[topic] = append(i.callbacks[topic], callbackFunc)
	i.Unlock()
}

func (i *inMemoryMQ) Unsubscribe(topic string, sub <-chan Message) {
	i.Lock()
	defer i.Unlock()

	subscribers, ok := i.topics[topic]
	if !ok {
		return
	}

	var newSubs []chan Message
	for _, subscriber := range subscribers {
		if subscriber == sub {
			continue
		}
		newSubs = append(newSubs, subscriber)
	}

	i.topics[topic] = newSubs
}

func (i *inMemoryMQ) Aria2Notify(events []rpc.Event, status int) {
	for _, event := range events {
		i.Publish(event.Gid, Message{
			TriggeredBy: event.Gid,
			Event:       strconv.FormatInt(int64(status), 10),
			Content:     events,
		})
	}
}
