package simulator

import (
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

type Producer interface {
	TopicWriter(topic string) io.Writer
	Close() error
}

func NewSaramaConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V2_4_0_0

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

func NewSaramaProducer(brokers []string) (Producer, error) {
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

func (w *SaramaWriter) Write(d []byte) (int, error) {
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

type FranzProducer struct {
	client       *kgo.Client
	closed       int32 // nonzero if the producer has started closing. accessed via atomics
	writeContext context.Context
	cancelWrites context.CancelFunc
}

func NewFranzProducer(brokers []string) (Producer, error) {
	reg := prometheus.DefaultRegisterer.(*prometheus.Registry)
	m := kprom.NewMetrics("franzgo", kprom.Registry(reg))

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.WithHooks(m),
		kgo.MetadataMinAge(time.Minute),
		kgo.BatchCompression(kgo.Lz4Compression(), kgo.SnappyCompression(), kgo.NoCompression()),
		kgo.MaxBufferedRecords(1000000),
	)
	if err != nil {
		return nil, err
	}

	wc, cancel := context.WithCancel(context.Background())

	return &FranzProducer{
		client:       client,
		writeContext: wc,
		cancelWrites: cancel,
	}, nil
}

func (p *FranzProducer) TopicWriter(topic string) io.Writer {
	if p.Closed() {
		panic("closed")
	}
	return &FranzWriter{p: p, topic: topic}
}

func (p *FranzProducer) Closed() bool {
	return atomic.LoadInt32(&p.closed) != 0
}

func (p *FranzProducer) Close() error {
	if p.Closed() {
		return errors.New("already closed")
	}
	atomic.StoreInt32(&p.closed, 1)

	p.cancelWrites()
	p.client.Close()

	return nil
}

type FranzWriter struct {
	p     *FranzProducer
	topic string
}

func (w *FranzWriter) Write(d []byte) (int, error) {
	if w.p.Closed() {
		return 0, syscall.EINVAL
	}

	r := kgo.SliceRecord(d)
	r.Topic = w.topic

	w.p.client.Produce(w.p.writeContext, r, func(r *kgo.Record, err error) {
		if err != nil && err != context.Canceled && err != kgo.ErrClientClosed {
			log.Printf("FranzProducer error on topic %s: %+v", w.topic, err)
		}
	})

	return len(d), nil
}
