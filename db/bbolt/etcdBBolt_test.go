package bbolt

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestBBoltLifecycle(t *testing.T) {
	bb := NewBBolt(&ConfigEtcdBBolt{FileName: filepath.Join(t.TempDir(), "test.db")})

	if err := bb.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := bb.Start(context.Background()); err == nil {
		t.Fatal("second Start should fail")
	}

	bucket := []byte("test")

	b, err := bb.CreateBucket(bucket)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	if b != nil {
		t.Error("CreateBucket should return a nil bucket handle")
	}
	if _, err := bb.CreateBucket(bucket); err != nil {
		t.Fatalf("CreateBucket on existing bucket: %v", err)
	}

	if err := bb.Put([]byte("missing"), []byte("k"), []byte("v")); err == nil {
		t.Error("Put into missing bucket should fail")
	}
	if err := bb.AddWithSeq([]byte("missing"), []byte("v")); err == nil {
		t.Error("AddWithSeq into missing bucket should fail")
	}

	if err := bb.Put(bucket, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := bb.Put(bucket, []byte("k2"), []byte("v2")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	val, err := bb.Get(bucket, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(val, []byte("v1")) {
		t.Errorf("Get(k1) = %q, want v1", val)
	}

	val, err = bb.Get(bucket, []byte("missing-key"))
	if err != nil {
		t.Fatalf("Get(missing key) error: %v", err)
	}
	if len(val) != 0 {
		t.Errorf("Get(missing key) = %q, want empty", val)
	}

	if _, err := bb.Get([]byte("missing"), []byte("k1")); err == nil {
		t.Error("Get from missing bucket should fail")
	}

	if err := bb.AddWithSeq(bucket, []byte("s1")); err != nil {
		t.Fatalf("AddWithSeq: %v", err)
	}
	if err := bb.AddWithSeq(bucket, []byte("s2")); err != nil {
		t.Fatalf("AddWithSeq: %v", err)
	}
	id, err := bb.AddWithSeqID(bucket, []byte("s3"))
	if err != nil {
		t.Fatalf("AddWithSeqID: %v", err)
	}
	// bbolt sequences start at 1
	if id != 3 {
		t.Errorf("AddWithSeqID = %d, want 3 (sequence must increment)", id)
	}

	if got, err := bb.Get(bucket, bb.Itob(1)); err != nil || !bytes.Equal(got, []byte("s1")) {
		t.Errorf("Get(seq 1) = %q, err %v; want s1", got, err)
	}
	if got, err := bb.Get(bucket, bb.Itob(2)); err != nil || !bytes.Equal(got, []byte("s2")) {
		t.Errorf("Get(seq 2) = %q, err %v; want s2", got, err)
	}
	if got, err := bb.Get(bucket, bb.Itob(3)); err != nil || !bytes.Equal(got, []byte("s3")) {
		t.Errorf("Get(seq 3) = %q, err %v; want s3", got, err)
	}

	all, err := bb.GetAll(bucket)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(*all) != 5 {
		t.Errorf("GetAll len = %d, want 5", len(*all))
	}

	if _, err := bb.GetAll([]byte("missing")); err == nil {
		t.Error("GetAll from missing bucket should fail")
	}

	if err := bb.DelByKey(bucket, []byte("k1")); err != nil {
		t.Fatalf("DelByKey: %v", err)
	}
	if val, err := bb.Get(bucket, []byte("k1")); err != nil || len(val) != 0 {
		t.Errorf("Get(deleted) = %q, err %v; want empty", val, err)
	}

	if err := bb.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := bb.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}

	if err := bb.Start(context.Background()); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	defer func() {
		if err := bb.Stop(); err != nil {
			t.Errorf("final Stop: %v", err)
		}
	}()
	if val, err := bb.Get(bucket, []byte("k2")); err != nil || !bytes.Equal(val, []byte("v2")) {
		t.Errorf("Get(k2) after restart = %q, err %v; want v2", val, err)
	}
}

func TestBBoltBatchPut(t *testing.T) {
	bb := NewBBolt(&ConfigEtcdBBolt{FileName: filepath.Join(t.TempDir(), "test.db")})
	if err := bb.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if err := bb.Stop(); err != nil {
			t.Errorf("Stop: %v", err)
		}
	}()

	bucket := []byte("batch")
	if _, err := bb.CreateBucket(bucket); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	if err := bb.BatchPut([]byte("missing"), &[]KV{}); err == nil {
		t.Error("BatchPut into missing bucket should fail")
	}

	items := &[]KV{
		{Key: []byte("a"), Value: []byte("1")},
		{Key: []byte("b"), Value: []byte("2")},
		{Key: []byte("c"), Value: []byte("3")},
	}
	if err := bb.BatchPut(bucket, items); err != nil {
		t.Fatalf("BatchPut: %v", err)
	}
	all, err := bb.GetAll(bucket)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(*all) != 3 {
		t.Errorf("GetAll len = %d, want 3", len(*all))
	}
	if val, err := bb.Get(bucket, []byte("b")); err != nil || !bytes.Equal(val, []byte("2")) {
		t.Errorf("Get(b) = %q, err %v; want 2", val, err)
	}
}
