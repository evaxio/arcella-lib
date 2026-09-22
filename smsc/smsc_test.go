package smsc

import (
	"context"
	"testing"
	"time"

	"github.com/linxGnu/gosmpp"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

func newTestSubmitSM(t *testing.T) *pdu.SubmitSM {
	t.Helper()
	submit := pdu.NewSubmitSM().(*pdu.SubmitSM)
	if err := submit.Message.SetMessageWithEncoding("test", data.GSM7BIT); err != nil {
		t.Fatalf("SetMessageWithEncoding: %v", err)
	}
	return submit
}

func TestCustomStoreRoundTrip(t *testing.T) {
	store := NewCustomStore()
	ctx := context.Background()
	submit := newTestSubmitSM(t)
	req := gosmpp.Request{PDU: submit, TimeSent: time.Now()}
	if err := store.Set(ctx, req); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok := store.Get(ctx, submit.GetSequenceNumber())
	if !ok {
		t.Fatal("Get: not found")
	}
	if got.TimeSent.IsZero() {
		t.Error("TimeSent was lost on round-trip")
	}
	if !got.TimeSent.Equal(req.TimeSent) {
		t.Errorf("TimeSent = %v, want %v", got.TimeSent, req.TimeSent)
	}
	gotSubmit, isSubmit := got.PDU.(*pdu.SubmitSM)
	if !isSubmit {
		t.Fatalf("PDU type = %T", got.PDU)
	}
	msg, err := gotSubmit.Message.GetMessage()
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if msg != "test" {
		t.Errorf("message = %q, want %q", msg, "test")
	}
	if n, err := store.Length(ctx); err != nil || n != 1 {
		t.Errorf("Length = %d, %v; want 1, nil", n, err)
	}
	list := store.List(ctx)
	if len(list) != 1 {
		t.Errorf("List len = %d, want 1", len(list))
	}
	if err := store.Delete(ctx, submit.GetSequenceNumber()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok = store.Get(ctx, submit.GetSequenceNumber()); ok {
		t.Error("Get after Delete: still present")
	}
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
}

func TestCustomStoreNilStore(t *testing.T) {
	var store CustomStore
	ctx := context.Background()
	submit := newTestSubmitSM(t)
	if err := store.Set(ctx, gosmpp.Request{PDU: submit}); err == nil {
		t.Error("Set on nil store: expected error")
	}
	if _, ok := store.Get(ctx, 1); ok {
		t.Error("Get on nil store: expected not found")
	}
	if err := store.Delete(ctx, 1); err == nil {
		t.Error("Delete on nil store: expected error")
	}
	if err := store.Clear(ctx); err == nil {
		t.Error("Clear on nil store: expected error")
	}
	if _, err := store.Length(ctx); err == nil {
		t.Error("Length on nil store: expected error")
	}
	if l := store.List(ctx); len(l) != 0 {
		t.Errorf("List on nil store len = %d, want 0", len(l))
	}
}

func TestSubmitSMResponseCorrelation(t *testing.T) {
	s := NewSMSCenter(&ConfigSMSC{DryRun: true})
	submit := newTestSubmitSM(t)
	resp := pdu.NewSubmitSMRespFromReq(submit).(*pdu.SubmitSMResp)
	resp.MessageID = "12345"

	ch := make(chan string, 1)
	s.waitChs[submit] = ch
	s.handleSubmitSMResponse(gosmpp.Response{PDU: resp, OriginalRequest: gosmpp.Request{PDU: submit}})

	select {
	case id := <-ch:
		if id != "12345" {
			t.Errorf("messageID = %q, want %q", id, "12345")
		}
	case <-time.After(time.Second):
		t.Fatal("no response delivered")
	}
	if len(s.waitChs) != 0 {
		t.Errorf("waitChs = %d entries, want 0", len(s.waitChs))
	}
}

func TestSubmitSMResponseUnknownRequest(t *testing.T) {
	s := NewSMSCenter(&ConfigSMSC{DryRun: true})
	other := newTestSubmitSM(t)
	resp := pdu.NewSubmitSMRespFromReq(other).(*pdu.SubmitSMResp)
	resp.MessageID = "999"

	done := make(chan struct{})
	go func() {
		s.handleSubmitSMResponse(gosmpp.Response{PDU: resp, OriginalRequest: gosmpp.Request{PDU: other}})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler blocked on unknown request")
	}
}

func TestSendMessageDryRun(t *testing.T) {
	s := NewSMSCenter(&ConfigSMSC{DryRun: true})
	id, err := s.SendMessage("from", "to", "text")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if id != "DryRun" {
		t.Errorf("id = %q, want %q", id, "DryRun")
	}
}
