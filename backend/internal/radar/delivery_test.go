package radar

import (
	"context"
	"testing"
	"time"

	"trend-graph/internal/store"
)

func TestDeliveryServiceCapsMajorAlertsAndKeepsDigestIdempotent(t *testing.T) {
	items := make([]store.RadarSignal, 0, 5)
	for index := 1; index <= 5; index++ {
		items = append(items, store.RadarSignal{
			Signal:   store.Signal{ID: int64(index), OriginalTitle: "重大更新", OriginalURL: "https://example.com/release", Qualification: "qualified"},
			Analysis: &store.SignalAnalysis{AnalysisJSON: `{"whatChanged":"模型发布","action":"阅读迁移文档","contentOpportunity":"迁移清单","alertEligible":true,"alertCategory":"major_release","alertReason":"核心模型正式发布"}`},
		})
	}
	deliveries := &fakeDeliveryStore{keys: map[string]bool{}}
	sender := &fakeFeishuSender{}
	service := NewDeliveryService(deliveries, fakeDeliverySignals{items: items}, sender)
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

type fakeDeliverySignals struct{ items []store.RadarSignal }

func (f fakeDeliverySignals) ListRadarSignals(int) ([]store.RadarSignal, error) { return f.items, nil }

type fakeDeliveryStore struct {
	keys map[string]bool
	next int64
	sent int
}

func (f *fakeDeliveryStore) Begin(run *store.DeliveryRun) (bool, error) {
	if f.keys[run.IdempotencyKey] {
		return false, nil
	}
	f.keys[run.IdempotencyKey] = true
	f.next++
	run.ID = f.next
	return true, nil
}
func (f *fakeDeliveryStore) Finish(_ int64, status, _ string, _ *time.Time) error {
	if status == "sent" {
		f.sent++
	}
	return nil
}
func (f *fakeDeliveryStore) CountSentSince(kind string, _ time.Time) (int, error) {
	if kind == "major_alert" {
		return f.sent, nil
	}
	return 0, nil
}

type fakeFeishuSender struct{ calls int }

func (f *fakeFeishuSender) Notify(context.Context, any) error {
	f.calls++
	return nil
}
