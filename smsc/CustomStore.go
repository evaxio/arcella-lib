package smsc

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"strconv"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/linxGnu/gosmpp/pdu"

	log "log/slog"

	"github.com/linxGnu/gosmpp"
)

// This is a just an example how to implement a custom store.
//
// Your implementation must be concurrency safe
//
// In this example we use bigcache https://github.com/allegro/bigcache
// Warning:
//  - This is just an example and should be tested before using in production
//	- We are serializing with gob, some field cannot be serialized for simplicity
//  - We recommend you implement your own serialization/deserialization if you choose to use bigcache

type CustomStore struct {
	store *bigcache.BigCache
}

func NewCustomStore() CustomStore {
	cfg := bigcache.DefaultConfig(30 * time.Second)
	cfg.MaxEntrySize = 4096
	cache, err := bigcache.New(context.Background(), cfg)
	if err != nil {
		log.Error("NewCustomStore", log.String("message", err.Error()))
	}
	return CustomStore{
		store: cache,
	}
}

func (s CustomStore) Set(ctx context.Context, request gosmpp.Request) error {
	log.Debug("Set", log.Any("request", request))
	select {
	case <-ctx.Done():
		log.Debug("Task cancelled")
		return ctx.Err()
	default:
		if s.store == nil {
			return errors.New("custom store is not initialized")
		}
		b, err := serialize(request)
		if err != nil {
			return err
		}
		err = s.store.Set(strconv.Itoa(int(request.PDU.GetSequenceNumber())), b)
		if err != nil {
			return err
		}
		return nil
	}
}

func (s CustomStore) Get(ctx context.Context, sequenceNumber int32) (gosmpp.Request, bool) {
	log.Debug("Get", log.Any("sequenceNumber", sequenceNumber))
	select {
	case <-ctx.Done():
		log.Debug("Task cancelled")
		return gosmpp.Request{}, false
	default:
		if s.store == nil {
			return gosmpp.Request{}, false
		}
		bRequest, err := s.store.Get(strconv.Itoa(int(sequenceNumber)))
		if err != nil {
			return gosmpp.Request{}, false
		}
		request, err := deserialize(bRequest)
		if err != nil {
			return gosmpp.Request{}, false
		}
		return request, true
	}
}

func (s CustomStore) List(ctx context.Context) []gosmpp.Request {
	log.Debug("List")
	var requests []gosmpp.Request
	select {
	case <-ctx.Done():
		return requests
	default:
		if s.store == nil {
			return requests
		}
		it := s.store.Iterator()
		for it.SetNext() {
			value, err := it.Value()
			if err != nil {
				return requests
			}
			request, _ := deserialize(value.Value())
			requests = append(requests, request)
		}
		return requests
	}
}

func (s CustomStore) Delete(ctx context.Context, sequenceNumber int32) error {
	log.Debug("Get", log.Any("sequenceNumber", sequenceNumber))
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if s.store == nil {
			return errors.New("custom store is not initialized")
		}
		err := s.store.Delete(strconv.Itoa(int(sequenceNumber)))
		if err != nil {
			return err
		}
		return nil
	}
}

func (s CustomStore) Clear(ctx context.Context) error {
	log.Debug("Clear")
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if s.store == nil {
			return errors.New("custom store is not initialized")
		}
		err := s.store.Reset()
		if err != nil {
			return err
		}
		return nil
	}
}

func (s CustomStore) Length(ctx context.Context) (int, error) {
	log.Debug("Length")
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		if s.store == nil {
			return 0, errors.New("custom store is not initialized")
		}
		return s.store.Len(), nil
	}
}

func (s CustomStore) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

func serialize(request gosmpp.Request) ([]byte, error) {
	buf := pdu.NewBuffer(make([]byte, 0, 64))
	request.PDU.Marshal(buf)
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(requestGob{
		Pdu:      buf.Bytes(),
		TimeSent: request.TimeSent,
	})
	if err != nil {
		return b.Bytes()[:], errors.New("serialization failed")
	}
	return b.Bytes(), nil
}

func deserialize(bRequest []byte) (request gosmpp.Request, err error) {
	r := requestGob{}
	b := bytes.Buffer{}
	_, err = b.Write(bRequest)
	if err != nil {
		return request, errors.New("deserialization failed")
	}
	d := gob.NewDecoder(&b)
	err = d.Decode(&r)
	if err != nil {
		return request, errors.New("deserialization failed")
	}
	p, err := pdu.Parse(bytes.NewReader(r.Pdu))
	if err != nil {
		return gosmpp.Request{}, err
	}
	return gosmpp.Request{
		PDU:      p,
		TimeSent: r.TimeSent,
	}, nil
}

type requestGob struct {
	Pdu      []byte
	TimeSent time.Time
}
