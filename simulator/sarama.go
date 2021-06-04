package simulator

import (
	"io"
	"log"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/Shopify/sarama.v1"
)

func NewSaramaConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V2_1_0_0

	timeout := 2 * time.Second

	config.Admin.Timeout = timeout
	config.Producer.Retry.Backoff = 2 * time.Second
	config.Producer.Retry.Max = 3
	config.Producer.Timeout = timeout

	config.Producer.Return.Successes = false
	config.Producer.Return.Errors = true

	return config
}

type SaramaProducer struct {
	producer      sarama.AsyncProducer
	closed        int32 // nonzero if the producer has started closing. accessed via atomics
	closeCh       chan struct{}
	pendingWrites sync.WaitGroup
}

func NewSaramaProducer(brokers []string) (*SaramaProducer, error) {
	producer, err := sarama.NewAsyncProducer(brokers, NewSaramaConfig())
	if err != nil {
		return nil, err
	}

	closeCh := make(chan struct{})

	go func() {
		for {
			select {
			case err := <-producer.Errors():
				log.Printf("SaramaProducer error: %+v", err)
			case <-closeCh:
				return
			}
		}
	}()

	return &SaramaProducer{
		producer:      producer,
		closeCh:       closeCh,
		pendingWrites: sync.WaitGroup{},
	}, nil
}

func (p *SaramaProducer) TopicWriter(topic string) io.Writer {
	if p.Closed() {
		panic("closed")
	}
	return &SaramaWriter{p: p, topic: topic}
}

func (p *SaramaProducer) Closed() bool {
	return atomic.LoadInt32(&p.closed) != 0
}

func (p *SaramaProducer) Close() error {
	if p.Closed() {
		return errors.New("already closed")
	}

	atomic.StoreInt32(&p.closed, 1)

	p.pendingWrites.Wait()
	close(p.closeCh)

	return p.producer.Close()
}

type SaramaWriter struct {
	p     *SaramaProducer
	topic string
}

func (w *SaramaWriter) Write(d []byte) (n int, err error) {
	if w.p.Closed() {
		return 0, syscall.EINVAL
	}

	w.p.pendingWrites.Add(1)
	defer w.p.pendingWrites.Done()

	w.p.producer.Input() <- &sarama.ProducerMessage{
		Topic: w.topic,
		Key:   nil,
		Value: sarama.ByteEncoder(d),
	}

	return len(d), nil
}
