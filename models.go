package main

type EntityType struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

type RelationType struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Property struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Entity struct {
	ID         string     `json:"id"`
	TypeID     string     `json:"typeId"`
	Label      string     `json:"label"`
	Properties []Property `json:"properties"`
	X          float64    `json:"x,omitempty"`
	Y          float64    `json:"y,omitempty"`
}

type Relationship struct {
	ID         string     `json:"id"`
	Source     string     `json:"source"`
	Target     string     `json:"target"`
	TypeID     string     `json:"typeId"`
	Label      string     `json:"label"`
	Properties []Property `json:"properties"`
	Directed   bool       `json:"directed"`
}

type GraphData struct {
	Entities      []Entity       `json:"entities"`
	Relationships []Relationship `json:"relationships"`
	EntityTypes   []EntityType   `json:"entityTypes"`
	RelationTypes []RelationType `json:"relationTypes"`
}

type LinkAnalysisResult struct {
	Entities      []Entity       `json:"entities"`
	Relationships []Relationship `json:"relationships"`
}

type PathResult struct {
	Paths [][]string `json:"paths"`
}

type ClusterResult struct {
	Clusters map[string]int `json:"clusters"`
}

type SNAMetrics struct {
	EntityID            string  `json:"entityId"`
	DegreeCentrality    float64 `json:"degreeCentrality"`
	BetweennessCentrality float64 `json:"betweennessCentrality"`
	ClosenessCentrality float64 `json:"closenessCentrality"`
}

type SNAResult struct {
	Metrics []SNAMetrics `json:"metrics"`
}

func DefaultEntityTypes() []EntityType {
	return []EntityType{
		{ID: "person", Name: "人物", Color: "#1890ff", Icon: "user"},
		{ID: "organization", Name: "组织", Color: "#52c41a", Icon: "bank"},
		{ID: "event", Name: "事件", Color: "#faad14", Icon: "calendar"},
		{ID: "location", Name: "地点", Color: "#eb2f96", Icon: "environment"},
		{ID: "phone", Name: "电话", Color: "#722ed1", Icon: "phone"},
		{ID: "account", Name: "账户", Color: "#13c2c2", Icon: "credit-card"},
		{ID: "vehicle", Name: "车辆", Color: "#fa541c", Icon: "car"},
		{ID: "document", Name: "文档", Color: "#2f54eb", Icon: "file"},
	}
}

// ==================== History ====================

type HistoryEntry struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	Action      string `json:"action"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	TargetLabel string `json:"targetLabel"`
	Detail      string `json:"detail"`
	Snapshot    string `json:"snapshot,omitempty"`
}

// ==================== NeuroDB Status ====================

type NeuroDBStatus struct {
	Connected  bool   `json:"connected"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	NodeCount  int    `json:"nodeCount"`
	LinkCount  int    `json:"linkCount"`
	Embedded   bool   `json:"embedded"`
	BinaryPath string `json:"binaryPath"`
	InstallDir string `json:"installDir"`
}

func DefaultRelationTypes() []RelationType {
	return []RelationType{
		{ID: "associate", Name: "关联", Color: "#999999"},
		{ID: "family", Name: "亲属", Color: "#f5222d"},
		{ID: "colleague", Name: "同事", Color: "#1890ff"},
		{ID: "transaction", Name: "交易", Color: "#52c41a"},
		{ID: "communication", Name: "通讯", Color: "#722ed1"},
		{ID: "ownership", Name: "所有权", Color: "#fa8c16"},
		{ID: "membership", Name: "成员", Color: "#13c2c2"},
		{ID: "travel", Name: "出行", Color: "#eb2f96"},
	}
}
