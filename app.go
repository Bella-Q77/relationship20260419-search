package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

const neuroPort = 18839

type App struct {
	ctx      context.Context
	graph    *GraphService
	history  *HistoryService
	neurodb  *NeuroDBClient
	neuroEmb *NeuroDBEmbed
}

func NewApp() *App {
	return &App{
		graph:    NewGraphService(),
		history:  NewHistoryService(),
		neurodb:  NewNeuroDBClient("127.0.0.1", neuroPort),
		neuroEmb: NewNeuroDBEmbed(neuroPort),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadFromFile()
	go a.startEmbeddedNeuroDB()
}

func (a *App) domReady(ctx context.Context) {}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	a.saveToFile()
	a.syncToNeuroDB()
	return false
}

func (a *App) shutdown(ctx context.Context) {
	a.saveToFile()
	if a.neurodb.IsConnected() {
		a.neurodb.SaveDB()
		a.neurodb.Close()
	}
	a.neuroEmb.Stop()
}

func (a *App) startEmbeddedNeuroDB() {
	if err := a.neuroEmb.Start(); err != nil {
		log.Printf("NeuroDB 嵌入启动失败: %v", err)
		return
	}

	if err := a.neurodb.Connect(); err != nil {
		log.Printf("NeuroDB 连接失败: %v", err)
		return
	}
	log.Println("NeuroDB 嵌入式连接成功")
	a.syncToNeuroDB()
}

func (a *App) syncToNeuroDB() {
	if !a.neurodb.IsConnected() {
		return
	}
	data := a.graph.GetGraphData()
	for _, e := range data.Entities {
		props := map[string]string{
			"_uid":    e.ID,
			"_typeId": e.TypeID,
			"label":   e.Label,
		}
		for _, p := range e.Properties {
			props["p_"+p.Key] = p.Value
		}
		a.neurodb.CreateNode(e.TypeID, props)
	}
	for _, r := range data.Relationships {
		props := map[string]string{
			"_uid":    r.ID,
			"_typeId": r.TypeID,
			"label":   r.Label,
		}
		if r.Directed {
			props["directed"] = "true"
		}
		for _, p := range r.Properties {
			props["p_"+p.Key] = p.Value
		}
		a.neurodb.CreateRelation(r.Source, r.Target, r.TypeID, props)
	}
	a.neurodb.SaveDB()
}

// ==================== Persistence ====================

func (a *App) getDataFilePath() string {
	homeDir, _ := os.UserHomeDir()
	dir := filepath.Join(homeDir, ".relationship-analyzer")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "graph_data.json")
}

func (a *App) saveToFile() {
	data := a.graph.ExportJSON()
	os.WriteFile(a.getDataFilePath(), []byte(data), 0644)
}

func (a *App) loadFromFile() {
	filePath := a.getDataFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	a.graph.ImportJSON(string(data))
}

// ==================== Graph Data CRUD ====================

func (a *App) GetGraphData() GraphData {
	return a.graph.GetGraphData()
}

func (a *App) AddEntity(entityJSON string) Entity {
	var e Entity
	json.Unmarshal([]byte(entityJSON), &e)
	result := a.graph.AddEntity(e)
	a.saveToFile()
	a.history.Record("add_entity", "entity", result.ID, result.Label,
		fmt.Sprintf("%s - %s", actionLabel("add_entity"), fmtEntityDetail(result)))

	if a.neurodb.IsConnected() {
		props := map[string]string{"_uid": result.ID, "_typeId": result.TypeID, "label": result.Label}
		for _, p := range result.Properties {
			props["p_"+p.Key] = p.Value
		}
		a.neurodb.CreateNode(result.TypeID, props)
	}
	return result
}

func (a *App) UpdateEntity(entityJSON string) Entity {
	var e Entity
	json.Unmarshal([]byte(entityJSON), &e)
	result := a.graph.UpdateEntity(e)
	a.saveToFile()
	a.history.Record("update_entity", "entity", result.ID, result.Label,
		fmt.Sprintf("%s - %s", actionLabel("update_entity"), fmtEntityDetail(result)))

	if a.neurodb.IsConnected() {
		props := map[string]string{"_typeId": result.TypeID, "label": result.Label}
		for _, p := range result.Properties {
			props["p_"+p.Key] = p.Value
		}
		a.neurodb.UpdateNodeProps(result.ID, props)
	}
	return result
}

func (a *App) DeleteEntity(id string) bool {
	data := a.graph.GetGraphData()
	var label string
	var typeID string
	for _, e := range data.Entities {
		if e.ID == id {
			label = e.Label
			typeID = e.TypeID
			break
		}
	}

	result := a.graph.DeleteEntity(id)
	a.saveToFile()
	if result {
		a.history.Record("delete_entity", "entity", id, label,
			fmt.Sprintf("%s - 类型:%s, 标签:%s", actionLabel("delete_entity"), typeID, label))
		if a.neurodb.IsConnected() {
			a.neurodb.DeleteNodeByUID(id)
		}
	}
	return result
}

func (a *App) AddRelationship(relJSON string) Relationship {
	var r Relationship
	json.Unmarshal([]byte(relJSON), &r)
	result := a.graph.AddRelationship(r)
	a.saveToFile()
	a.history.Record("add_relationship", "relationship", result.ID, result.Label,
		fmt.Sprintf("%s - %s", actionLabel("add_relationship"), fmtRelDetail(result)))

	if a.neurodb.IsConnected() {
		props := map[string]string{"_uid": result.ID, "_typeId": result.TypeID, "label": result.Label}
		if result.Directed {
			props["directed"] = "true"
		}
		for _, p := range result.Properties {
			props["p_"+p.Key] = p.Value
		}
		a.neurodb.CreateRelation(result.Source, result.Target, result.TypeID, props)
	}
	return result
}

func (a *App) UpdateRelationship(relJSON string) Relationship {
	var r Relationship
	json.Unmarshal([]byte(relJSON), &r)
	result := a.graph.UpdateRelationship(r)
	a.saveToFile()
	a.history.Record("update_relationship", "relationship", result.ID, result.Label,
		fmt.Sprintf("%s - %s", actionLabel("update_relationship"), fmtRelDetail(result)))
	return result
}

func (a *App) DeleteRelationship(id string) bool {
	data := a.graph.GetGraphData()
	var label string
	for _, r := range data.Relationships {
		if r.ID == id {
			label = r.Label
			break
		}
	}

	result := a.graph.DeleteRelationship(id)
	a.saveToFile()
	if result {
		a.history.Record("delete_relationship", "relationship", id, label,
			fmt.Sprintf("%s - 标签:%s", actionLabel("delete_relationship"), label))
	}
	return result
}

// ==================== Types ====================

func (a *App) GetEntityTypes() []EntityType {
	return a.graph.GetEntityTypes()
}

func (a *App) GetRelationTypes() []RelationType {
	return a.graph.GetRelationTypes()
}

func (a *App) AddEntityType(typeJSON string) EntityType {
	var et EntityType
	json.Unmarshal([]byte(typeJSON), &et)
	result := a.graph.AddEntityType(et)
	a.history.Record("add_entity_type", "type", result.ID, result.Name,
		fmt.Sprintf("%s - %s", actionLabel("add_entity_type"), result.Name))
	return result
}

func (a *App) AddRelationType(typeJSON string) RelationType {
	var rt RelationType
	json.Unmarshal([]byte(typeJSON), &rt)
	result := a.graph.AddRelationType(rt)
	a.history.Record("add_relation_type", "type", result.ID, result.Name,
		fmt.Sprintf("%s - %s", actionLabel("add_relation_type"), result.Name))
	return result
}

// ==================== Import/Export ====================

func (a *App) ImportJSONData(jsonStr string) GraphData {
	data, _ := a.graph.ImportJSON(jsonStr)
	a.saveToFile()
	a.history.Record("import_data", "graph", "", "",
		fmt.Sprintf("%s - %d个实体, %d条关系",
			actionLabel("import_data"), len(data.Entities), len(data.Relationships)))
	return data
}

func (a *App) ExportJSONData() string {
	return a.graph.ExportJSON()
}

func (a *App) ImportCSVEntities(csvStr string) GraphData {
	data, _ := a.graph.ImportCSVEntities(csvStr)
	a.saveToFile()
	a.history.Record("import_data", "graph", "", "",
		fmt.Sprintf("导入CSV实体 - 共%d个实体", len(data.Entities)))
	return data
}

func (a *App) ImportCSVRelationships(csvStr string) GraphData {
	data, _ := a.graph.ImportCSVRelationships(csvStr)
	a.saveToFile()
	a.history.Record("import_data", "graph", "", "",
		fmt.Sprintf("导入CSV关系 - 共%d条关系", len(data.Relationships)))
	return data
}

func (a *App) ClearData() {
	a.graph.ClearData()
	a.saveToFile()
	a.history.Record("clear_data", "graph", "", "", actionLabel("clear_data"))
}

// ==================== History ====================

func (a *App) GetHistory() []HistoryEntry {
	return a.history.GetAll()
}

func (a *App) GetRecentHistory(count int) []HistoryEntry {
	return a.history.GetRecent(count)
}

func (a *App) ClearHistory() {
	a.history.Clear()
}

// ==================== NeuroDB ====================

func (a *App) GetNeuroDBStatus() NeuroDBStatus {
	st := NeuroDBStatus{
		Connected:  a.neurodb.IsConnected(),
		Host:       a.neurodb.host,
		Port:       a.neurodb.port,
		Embedded:   a.neuroEmb.IsRunning(),
		BinaryPath: a.neuroEmb.findServerBinary(),
		InstallDir: a.neuroEmb.GetInstallPath(),
	}
	if st.Connected {
		if res, err := a.neurodb.Execute("match (n) call result.count()"); err == nil && res.ResultCount > 0 && len(res.RawEntries) > 0 {
			fmt.Sscanf(res.RawEntries[0], "%d", &st.NodeCount)
		}
	}
	return st
}

func (a *App) ConnectNeuroDB(host string, port int) string {
	a.neurodb.Close()
	a.neurodb = NewNeuroDBClient(host, port)
	if err := a.neurodb.Connect(); err != nil {
		return fmt.Sprintf("连接失败: %v", err)
	}
	a.history.Record("neurodb_sync", "neurodb", "", "",
		fmt.Sprintf("连接到 NeuroDB %s:%d", host, port))
	return "OK"
}

func (a *App) StartNeuroDB() string {
	if a.neuroEmb.IsRunning() {
		return "NeuroDB 已在运行"
	}
	if err := a.neuroEmb.Start(); err != nil {
		return fmt.Sprintf("启动失败: %v", err)
	}
	if err := a.neurodb.Connect(); err != nil {
		return fmt.Sprintf("连接失败: %v", err)
	}
	return "OK"
}

func (a *App) StopNeuroDB() string {
	if a.neurodb.IsConnected() {
		a.neurodb.SaveDB()
		a.neurodb.Close()
	}
	a.neuroEmb.Stop()
	return "OK"
}

func (a *App) GetNeuroDBInfo() string {
	return a.neuroEmb.StatusInfo()
}

func (a *App) SyncToNeuroDB() string {
	if !a.neurodb.IsConnected() {
		return "NeuroDB 未连接"
	}
	a.syncToNeuroDB()
	data := a.graph.GetGraphData()
	a.history.Record("neurodb_sync", "neurodb", "", "",
		fmt.Sprintf("同步到NeuroDB: %d个节点, %d条关系", len(data.Entities), len(data.Relationships)))
	return "OK"
}

// ==================== File Dialogs ====================

func (a *App) OpenFileDialog(title string, filters []runtime.FileFilter) string {
	file, _ := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
	return file
}

func (a *App) SaveFileDialog(title string, defaultFilename string, filters []runtime.FileFilter) string {
	file, _ := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters:         filters,
	})
	return file
}

func (a *App) ReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (a *App) WriteFile(path string, content string) bool {
	return os.WriteFile(path, []byte(content), 0644) == nil
}

func (a *App) WriteBase64File(path string, base64Data string) bool {
	data, err := base64Decode(base64Data)
	if err != nil {
		log.Printf("WriteBase64File decode error: %v", err)
		return false
	}
	return os.WriteFile(path, data, 0644) == nil
}

// ==================== Analysis ====================

func (a *App) LinkAnalysis(entityID string, depth int) LinkAnalysisResult {
	return a.graph.LinkAnalysis(entityID, depth)
}

func (a *App) PathAnalysis(sourceID string, targetID string) PathResult {
	return a.graph.PathAnalysis(sourceID, targetID)
}

func (a *App) ClusterAnalysis() ClusterResult {
	return a.graph.ClusterAnalysis()
}

func (a *App) SocialNetworkAnalysis() SNAResult {
	return a.graph.SocialNetworkAnalysis()
}
