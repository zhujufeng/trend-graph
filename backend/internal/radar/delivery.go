package radar

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/notify"
	"trend-graph/internal/store"
)

type deliveryStore interface {
	Begin(*store.DeliveryRun) (bool, error)
	Finish(id int64, status, failure string, sentAt *time.Time) error
	CountSentSince(kind string, since time.Time) (int, error)
}

type deliverySignalStore interface {
	ListRadarSignals(limit int) ([]store.RadarSignal, error)
}

type feishuSender interface {
	Notify(context.Context, any) error
}

type DeliveryService struct {
	deliveries deliveryStore
	signals    deliverySignalStore
	feishu     feishuSender
}

func NewDeliveryService(deliveries deliveryStore, signals deliverySignalStore, feishu feishuSender) *DeliveryService {
	return &DeliveryService{deliveries: deliveries, signals: signals, feishu: feishu}
}

func (s *DeliveryService) SendDigest(ctx context.Context, now time.Time) error {
	localNow, start, err := shanghaiDay(now)
	_ = start
	if err != nil {
		return err
	}
	items, err := s.signals.ListRadarSignals(100)
	if err != nil {
		return err
	}
	digest, err := BuildDigest(items, localNow)
	if err != nil {
		return err
	}
	if len(digest.Signals) == 0 {
		return nil
	}
	ids := qualifiedSignalIDs(items, 8)
	return s.send(ctx, store.DeliveryRun{
		Kind: "digest", IdempotencyKey: fmt.Sprintf("digest:%s:%02d", localNow.Format("2006-01-02"), localNow.Hour()),
		SignalIDsJSON: mustJSON(ids), Status: "running",
	}, digestPost(digest))
}

func (s *DeliveryService) SendMajorAlerts(ctx context.Context, now time.Time) error {
	_, start, err := shanghaiDay(now)
	if err != nil {
		return err
	}
	sent, err := s.deliveries.CountSentSince("major_alert", start)
	if err != nil {
		return err
	}
	remaining := 3 - sent
	if remaining <= 0 {
		return nil
	}
	items, err := s.signals.ListRadarSignals(100)
	if err != nil {
		return err
	}
	for _, item := range items {
		if remaining == 0 {
			break
		}
		alert, ok := majorAlert(item)
		if !ok {
			continue
		}
		run := store.DeliveryRun{
			Kind: "major_alert", IdempotencyKey: fmt.Sprintf("major:%d", item.Signal.ID),
			SignalIDsJSON: mustJSON([]int64{item.Signal.ID}), Status: "running",
		}
		if err := s.send(ctx, run, alert); err != nil {
			return err
		}
		remaining--
	}
	return nil
}

func (s *DeliveryService) send(ctx context.Context, run store.DeliveryRun, payload notify.FeishuPost) error {
	started, err := s.deliveries.Begin(&run)
	if err != nil || !started {
		return err
	}
	if err := s.feishu.Notify(ctx, payload); err != nil {
		_ = s.deliveries.Finish(run.ID, "failed", err.Error(), nil)
		return err
	}
	now := time.Now().UTC()
	return s.deliveries.Finish(run.ID, "sent", "", &now)
}

func majorAlert(item store.RadarSignal) (notify.FeishuPost, bool) {
	if item.Signal.Qualification != "qualified" || item.Analysis == nil {
		return notify.FeishuPost{}, false
	}
	var analysis struct {
		WhatChanged   string `json:"whatChanged"`
		Action        string `json:"action"`
		AlertEligible bool   `json:"alertEligible"`
		AlertCategory string `json:"alertCategory"`
		AlertReason   string `json:"alertReason"`
	}
	if json.Unmarshal([]byte(item.Analysis.AnalysisJSON), &analysis) != nil || !analysis.AlertEligible || !analyzer.ValidAlertCategory(analysis.AlertCategory) {
		return notify.FeishuPost{}, false
	}
	return notify.FeishuPost{
		Title: "AI 重磅信号 · " + item.Signal.OriginalTitle,
		Sections: []notify.FeishuSection{
			{Text: "事实：" + analysis.WhatChanged},
			{Text: "判断：" + analysis.AlertReason},
			{Text: "建议行动：" + analysis.Action},
			{Text: "来源：", LinkText: "查看原始信息", LinkURL: item.Signal.OriginalURL},
		},
	}, true
}

func digestPost(digest Digest) notify.FeishuPost {
	post := notify.FeishuPost{Title: digest.Title}
	for _, signal := range digest.Signals {
		post.Sections = append(post.Sections,
			notify.FeishuSection{Text: "事实：" + signal.WhatChanged},
			notify.FeishuSection{Text: "判断：" + signal.Interpretation},
			notify.FeishuSection{Text: "行动：" + signal.Action + "（证据：" + signal.EvidenceClass + "）", LinkText: signal.Title, LinkURL: signal.LinkURL},
		)
	}
	for _, opportunity := range digest.ContentOpportunities {
		post.Sections = append(post.Sections, notify.FeishuSection{Text: "内容机会：" + opportunity.Title + "｜" + opportunity.Angle})
	}
	return post
}

func qualifiedSignalIDs(items []store.RadarSignal, limit int) []int64 {
	ids := make([]int64, 0, limit)
	for _, item := range items {
		if item.Signal.Qualification == "qualified" && item.Analysis != nil {
			ids = append(ids, item.Signal.ID)
			if len(ids) == limit {
				break
			}
		}
	}
	return ids
}

func shanghaiDay(now time.Time) (time.Time, time.Time, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return local, start, nil
}

func mustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
