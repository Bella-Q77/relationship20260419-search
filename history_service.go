package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type HistoryService struct {
	mu      sync.RWMutex
	entries []HistoryEntry
	maxSize int
	file    string
}

func NewHistoryService() *HistoryService {
	homeDir, _ := os.UserHomeDir()
	dir := filepath.Join(homeDir, ".relationship-analyzer")
	os.MkdirAll(dir, 0755)
	hs := &HistoryService{
		entries: []HistoryEntry{},
		maxSize: 500,
		file:    filepath.Join(dir, "history.json"),
	}
	hs.load()
	return hs
}

func (h *HistoryService) Record(action, targetType, targetID, targetLabel, detail string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry := HistoryEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		TargetLabel: targetLabel,
		Detail:      detail,
	}

	h.entries = append([]HistoryEntry{entry}, h.entries...)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[:h.maxSize]
	}
	h.save()
}

func (h *HistoryService) GetAll() []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]HistoryEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

func (h *HistoryService) GetRecent(count int) []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if count > len(h.entries) {
		count = len(h.entries)
	}
	result := make([]HistoryEntry, count)
	copy(result, h.entries[:count])
	return result
}

func (h *HistoryService) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = []HistoryEntry{}
	h.save()
}

func (h *HistoryService) save() {
	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(h.file, data, 0644)
}

func (h *HistoryService) load() {
	data, err := os.ReadFile(h.file)
	if err != nil {
		return
	}
	json.Unmarshal(data, &h.entries)
}

func actionLabel(action string) string {
	labels := map[string]string{
		"add_entity":          "添加实体",
		"update_entity":       "更新实体",
		"delete_entity":       "删除实体",
		"add_relationship":    "添加关系",
		"update_relationship": "更新关系",
		"delete_relationship": "删除关系",
		"import_data":         "导入数据",
		"clear_data":          "清空数据",
		"add_entity_type":     "添加实体类型",
		"add_relation_type":   "添加关系类型",
		"neurodb_sync":        "同步到NeuroDB",
		"neurodb_load":        "从NeuroDB加载",
	}
	if l, ok := labels[action]; ok {
		return l
	}
	return action
}

func fmtEntityDetail(e Entity) string {
	return fmt.Sprintf("类型:%s, 标签:%s", e.TypeID, e.Label)
}

func fmtRelDetail(r Relationship) string {
	return fmt.Sprintf("类型:%s, 标签:%s, %s→%s", r.TypeID, r.Label, r.Source, r.Target)
}
