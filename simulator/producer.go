package simulator

import (
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

type Producer interface {
	TopicWriter(topic string) io.Writer
	Close() error
}

type FranzProducer struct {
	client        *kgo.Client
	closed        int32 // nonzero if the producer has started closing. accessed via atomics
	pendingWrites sync.WaitGroup
}

var (
	kpromMetrics = kprom.NewMetrics("franzgo", kprom.Registry(prometheus.DefaultRegisterer.(*prometheus.Registry)))
)

func NewFranzProducer(config TopicsConfig) (Producer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(config.Brokers...),
		kgo.WithHooks(kpromMetrics),
		kgo.MaxBufferedRecords(1e7),
		kgo.BatchMaxBytes(int32(config.BatchMaxBytes)),
	}

	if config.Compression {
		opts = append(opts, kgo.BatchCompression(kgo.Lz4Compression()))
	} else {
		opts = append(opts, kgo.BatchCompression(kgo.NoCompression()))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &FranzProducer{
		client: client,
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

	p.pendingWrites.Wait()

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

	w.p.pendingWrites.Add(1)
	w.p.client.Produce(context.Background(), r, func(r *kgo.Record, err error) {
		defer w.p.pendingWrites.Done()
		if err != nil && err != kgo.ErrClientClosed {
			log.Printf("FranzProducer error on topic %s: %+v", w.topic, err)
		}
	})

	return len(d), nil
}
