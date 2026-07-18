package radar

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"trend-graph/internal/store"
)

func TestDeliveryServiceCapsMajorAlertsAndKeepsDigestIdempotent(t *testing.T) {
	items := make([]store.RadarSignal, 0, 5)
	for index := 1; index <= 5; index++ {
		items = append(items, store.RadarSignal{
			Signal:   store.Signal{ID: int64(index), OriginalTitle: fmt.Sprintf("重大更新 %d", index), OriginalURL: fmt.Sprintf("https://example.com/release/%d", index), Qualification: "qualified", LifecycleState: store.LifecycleInbox},
			Analysis: &store.SignalAnalysis{AnalysisJSON: fmt.Sprintf(`{"matchedTopics":["主题%d"],"valueScore":5,"whatChanged":"模型发布","action":"阅读迁移文档","contentOpportunity":"迁移清单","alertEligible":true,"alertCategory":"major_release","alertReason":"核心模型正式发布"}`, index)},
		})
	}
	deliveries := &fakeDeliveryStore{keys: map[string]bool{}}
	sender := &fakeFeishuSender{}
	service := NewDeliveryService(deliveries, &fakeDeliverySignals{items: items}, sender)
	now := time.Date(2026, 7, 16, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	if err := service.SendMajorAlerts(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 3 {
		t.Fatalf("major alert calls = %d, want 3", sender.calls)
	}
	if err := service.SendDigest(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if err := service.SendDigest(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 4 {
		t.Fatalf("total calls = %d, duplicate digest should be skipped", sender.calls)
	}
}

func TestDeliveryMarksSignalsSoAlertsAndLaterDigestsDoNotRepeat(t *testing.T) {
	item := store.RadarSignal{
		Signal:   store.Signal{ID: 9, OriginalTitle: "Release 9", OriginalURL: "https://example.com/9", Qualification: "qualified", LifecycleState: store.LifecycleInbox},
		Analysis: &store.SignalAnalysis{AnalysisJSON: `{"matchedTopics":["AI"],"valueScore":5,"whatChanged":"正式发布","action":"阅读说明","alertEligible":true,"alertCategory":"major_release","alertReason":"重大版本"}`},
	}
	signals := &fakeDeliverySignals{items: []store.RadarSignal{item}}
	deliveries := &fakeDeliveryStore{keys: map[string]bool{}}
	deliveries.onComplete = func(ids []int64, deliveredAt time.Time) {
		for index := range signals.items {
			for _, id := range ids {
				if signals.items[index].Signal.ID == id {
					signals.items[index].Signal.LastDeliveredAt = &deliveredAt
				}
			}
		}
	}
	sender := &fakeFeishuSender{}
	service := NewDeliveryService(deliveries, signals, sender)
	now := time.Date(2026, 7, 16, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	if err := service.SendMajorAlerts(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if err := service.SendDigest(context.Background(), now.Add(12*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 || signals.items[0].Signal.LastDeliveredAt == nil {
		t.Fatalf("sender calls = %d, deliveredAt = %v", sender.calls, signals.items[0].Signal.LastDeliveredAt)
	}
}

func TestDeliveryFailureRemainsRetryable(t *testing.T) {
	item := store.RadarSignal{
		Signal:   store.Signal{ID: 12, OriginalTitle: "Useful update", OriginalURL: "https://example.com/12", Qualification: "qualified", LifecycleState: store.LifecycleInbox},
		Analysis: &store.SignalAnalysis{AnalysisJSON: `{"matchedTopics":["AI"],"valueScore":4,"whatChanged":"更新","action":"阅读"}`},
	}
	deliveries := &fakeDeliveryStore{keys: map[string]bool{}}
	sender := &fakeFeishuSender{err: errors.New("webhook unavailable")}
	service := NewDeliveryService(deliveries, &fakeDeliverySignals{items: []store.RadarSignal{item}}, sender)
	now := time.Date(2026, 7, 16, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	if err := service.SendDigest(context.Background(), now); err == nil {
		t.Fatal("expected webhook failure")
	}
	if err := service.SendDigest(context.Background(), now); err == nil {
		t.Fatal("expected retry to reach the webhook and fail again")
	}
	if deliveries.sent != 0 || deliveries.failed != 2 || sender.calls != 2 {
		t.Fatalf("sent = %d, failed = %d, sender calls = %d", deliveries.sent, deliveries.failed, sender.calls)
	}
}

type fakeDeliverySignals struct{ items []store.RadarSignal }

func (f *fakeDeliverySignals) ListRadarSignals(int) ([]store.RadarSignal, error) { return f.items, nil }

type fakeDeliveryStore struct {
	keys       map[string]bool
	next       int64
	sent       int
	failed     int
	onComplete func([]int64, time.Time)
	idKeys     map[int64]string
}

func (f *fakeDeliveryStore) Begin(run *store.DeliveryRun) (bool, error) {
	if f.keys[run.IdempotencyKey] {
		return false, nil
	}
	f.keys[run.IdempotencyKey] = true
	f.next++
	run.ID = f.next
	if f.idKeys == nil {
		f.idKeys = make(map[int64]string)
	}
	f.idKeys[run.ID] = run.IdempotencyKey
	return true, nil
}
func (f *fakeDeliveryStore) Finish(id int64, status, _ string, _ *time.Time) error {
	if status == "sent" {
		f.sent++
	}
	if status == "failed" {
		f.failed++
		delete(f.keys, f.idKeys[id])
	}
	return nil
}
func (f *fakeDeliveryStore) Complete(_ int64, ids []int64, sentAt time.Time) error {
	f.sent++
	if f.onComplete != nil {
		f.onComplete(ids, sentAt)
	}
	return nil
}
func (f *fakeDeliveryStore) CountSentSince(kind string, _ time.Time) (int, error) {
	if kind == "major_alert" {
		return f.sent, nil
	}
	return 0, nil
}

type fakeFeishuSender struct {
	calls int
	err   error
}

func (f *fakeFeishuSender) Notify(context.Context, any) error {
	f.calls++
	return f.err
}
