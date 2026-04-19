package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type GraphService struct {
	mu   sync.RWMutex
	data GraphData
}

func NewGraphService() *GraphService {
	return &GraphService{
		data: GraphData{
			Entities:      []Entity{},
			Relationships: []Relationship{},
			EntityTypes:   DefaultEntityTypes(),
			RelationTypes: DefaultRelationTypes(),
		},
	}
}

// ==================== CRUD ====================

func (s *GraphService) GetGraphData() GraphData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *GraphService) AddEntity(e Entity) Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.Properties == nil {
		e.Properties = []Property{}
	}
	s.data.Entities = append(s.data.Entities, e)
	return e
}

func (s *GraphService) UpdateEntity(e Entity) Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, ent := range s.data.Entities {
		if ent.ID == e.ID {
			s.data.Entities[i] = e
			return e
		}
	}
	return e
}

func (s *GraphService) DeleteEntity(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	entities := make([]Entity, 0, len(s.data.Entities))
	for _, e := range s.data.Entities {
		if e.ID == id {
			found = true
			continue
		}
		entities = append(entities, e)
	}
	s.data.Entities = entities

	rels := make([]Relationship, 0, len(s.data.Relationships))
	for _, r := range s.data.Relationships {
		if r.Source == id || r.Target == id {
			continue
		}
		rels = append(rels, r)
	}
	s.data.Relationships = rels
	return found
}

func (s *GraphService) AddRelationship(r Relationship) Relationship {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.Properties == nil {
		r.Properties = []Property{}
	}
	s.data.Relationships = append(s.data.Relationships, r)
	return r
}

func (s *GraphService) UpdateRelationship(r Relationship) Relationship {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, rel := range s.data.Relationships {
		if rel.ID == r.ID {
			s.data.Relationships[i] = r
			return r
		}
	}
	return r
}

func (s *GraphService) DeleteRelationship(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	rels := make([]Relationship, 0, len(s.data.Relationships))
	for _, r := range s.data.Relationships {
		if r.ID == id {
			found = true
			continue
		}
		rels = append(rels, r)
	}
	s.data.Relationships = rels
	return found
}

// ==================== Types ====================

func (s *GraphService) GetEntityTypes() []EntityType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.EntityTypes
}

func (s *GraphService) GetRelationTypes() []RelationType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.RelationTypes
}

func (s *GraphService) AddEntityType(et EntityType) EntityType {
	s.mu.Lock()
	defer s.mu.Unlock()
	if et.ID == "" {
		et.ID = uuid.New().String()
	}
	s.data.EntityTypes = append(s.data.EntityTypes, et)
	return et
}

func (s *GraphService) AddRelationType(rt RelationType) RelationType {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rt.ID == "" {
		rt.ID = uuid.New().String()
	}
	s.data.RelationTypes = append(s.data.RelationTypes, rt)
	return rt
}

// ==================== Import/Export ====================

func (s *GraphService) ImportJSON(jsonStr string) (GraphData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var imported GraphData
	if err := json.Unmarshal([]byte(jsonStr), &imported); err != nil {
		return s.data, fmt.Errorf("JSON解析失败: %w", err)
	}

	for _, e := range imported.Entities {
		if e.ID == "" {
			e.ID = uuid.New().String()
		}
		if e.Properties == nil {
			e.Properties = []Property{}
		}
		s.data.Entities = append(s.data.Entities, e)
	}

	for _, r := range imported.Relationships {
		if r.ID == "" {
			r.ID = uuid.New().String()
		}
		if r.Properties == nil {
			r.Properties = []Property{}
		}
		s.data.Relationships = append(s.data.Relationships, r)
	}

	if len(imported.EntityTypes) > 0 {
		existing := make(map[string]bool)
		for _, et := range s.data.EntityTypes {
			existing[et.ID] = true
		}
		for _, et := range imported.EntityTypes {
			if !existing[et.ID] {
				s.data.EntityTypes = append(s.data.EntityTypes, et)
			}
		}
	}
	if len(imported.RelationTypes) > 0 {
		existing := make(map[string]bool)
		for _, rt := range s.data.RelationTypes {
			existing[rt.ID] = true
		}
		for _, rt := range imported.RelationTypes {
			if !existing[rt.ID] {
				s.data.RelationTypes = append(s.data.RelationTypes, rt)
			}
		}
	}

	return s.data, nil
}

func (s *GraphService) ExportJSON() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.MarshalIndent(s.data, "", "  ")
	return string(data)
}

func (s *GraphService) ImportCSVEntities(csvStr string) (GraphData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	if err != nil {
		return s.data, fmt.Errorf("CSV解析失败: %w", err)
	}
	if len(records) < 2 {
		return s.data, fmt.Errorf("CSV数据不足")
	}

	headers := records[0]
	for _, row := range records[1:] {
		e := Entity{
			ID:         uuid.New().String(),
			Properties: []Property{},
		}
		for i, header := range headers {
			if i >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[i])
			switch strings.ToLower(header) {
			case "id":
				if val != "" {
					e.ID = val
				}
			case "type", "typeid":
				e.TypeID = val
			case "label", "name":
				e.Label = val
			default:
				if val != "" {
					e.Properties = append(e.Properties, Property{Key: header, Value: val})
				}
			}
		}
		if e.Label != "" {
			if e.TypeID == "" {
				e.TypeID = "person"
			}
			s.data.Entities = append(s.data.Entities, e)
		}
	}
	return s.data, nil
}

func (s *GraphService) ImportCSVRelationships(csvStr string) (GraphData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	if err != nil {
		return s.data, fmt.Errorf("CSV解析失败: %w", err)
	}
	if len(records) < 2 {
		return s.data, fmt.Errorf("CSV数据不足")
	}

	headers := records[0]
	for _, row := range records[1:] {
		r := Relationship{
			ID:         uuid.New().String(),
			Properties: []Property{},
		}
		for i, header := range headers {
			if i >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[i])
			switch strings.ToLower(header) {
			case "id":
				if val != "" {
					r.ID = val
				}
			case "source", "from":
				r.Source = val
			case "target", "to":
				r.Target = val
			case "type", "typeid":
				r.TypeID = val
			case "label":
				r.Label = val
			default:
				if val != "" {
					r.Properties = append(r.Properties, Property{Key: header, Value: val})
				}
			}
		}
		if r.Source != "" && r.Target != "" {
			if r.TypeID == "" {
				r.TypeID = "associate"
			}
			s.data.Relationships = append(s.data.Relationships, r)
		}
	}
	return s.data, nil
}

func (s *GraphService) ClearData() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Entities = []Entity{}
	s.data.Relationships = []Relationship{}
}

// ==================== Analysis Algorithms ====================

func (s *GraphService) buildAdjacency() (map[string][]string, map[string][]Relationship) {
	adj := make(map[string][]string)
	edgeMap := make(map[string][]Relationship)
	for _, r := range s.data.Relationships {
		adj[r.Source] = append(adj[r.Source], r.Target)
		adj[r.Target] = append(adj[r.Target], r.Source)
		edgeMap[r.Source] = append(edgeMap[r.Source], r)
		edgeMap[r.Target] = append(edgeMap[r.Target], r)
	}
	return adj, edgeMap
}

// LinkAnalysis: BFS to find all entities within N hops
func (s *GraphService) LinkAnalysis(entityID string, depth int) LinkAnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	adj, _ := s.buildAdjacency()
	visited := make(map[string]bool)
	visited[entityID] = true

	queue := []struct {
		id    string
		depth int
	}{{entityID, 0}}

	resultEntityIDs := make(map[string]bool)
	resultEntityIDs[entityID] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.depth >= depth {
			continue
		}

		for _, neighbor := range adj[curr.id] {
			if !visited[neighbor] {
				visited[neighbor] = true
				resultEntityIDs[neighbor] = true
				queue = append(queue, struct {
					id    string
					depth int
				}{neighbor, curr.depth + 1})
			}
		}
	}

	result := LinkAnalysisResult{}
	for _, e := range s.data.Entities {
		if resultEntityIDs[e.ID] {
			result.Entities = append(result.Entities, e)
		}
	}
	for _, r := range s.data.Relationships {
		if resultEntityIDs[r.Source] && resultEntityIDs[r.Target] {
			result.Relationships = append(result.Relationships, r)
		}
	}
	return result
}

// PathAnalysis: BFS to find shortest paths between two entities
func (s *GraphService) PathAnalysis(sourceID, targetID string) PathResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	adj, _ := s.buildAdjacency()

	type pathNode struct {
		id   string
		path []string
	}

	visited := make(map[string]bool)
	queue := []pathNode{{sourceID, []string{sourceID}}}
	visited[sourceID] = true

	var paths [][]string
	shortestLen := -1

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if shortestLen > 0 && len(curr.path) > shortestLen {
			break
		}

		if curr.id == targetID {
			pathCopy := make([]string, len(curr.path))
			copy(pathCopy, curr.path)
			paths = append(paths, pathCopy)
			shortestLen = len(curr.path)
			continue
		}

		for _, neighbor := range adj[curr.id] {
			if !visited[neighbor] || neighbor == targetID {
				newPath := make([]string, len(curr.path)+1)
				copy(newPath, curr.path)
				newPath[len(curr.path)] = neighbor
				queue = append(queue, pathNode{neighbor, newPath})
				if neighbor != targetID {
					visited[neighbor] = true
				}
			}
		}
	}

	if paths == nil {
		paths = [][]string{}
	}
	return PathResult{Paths: paths}
}

// ClusterAnalysis: Label Propagation for community detection
func (s *GraphService) ClusterAnalysis() ClusterResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	labels := make(map[string]int)
	for i, e := range s.data.Entities {
		labels[e.ID] = i
	}

	adj, _ := s.buildAdjacency()

	for iter := 0; iter < 100; iter++ {
		changed := false
		for _, e := range s.data.Entities {
			neighbors := adj[e.ID]
			if len(neighbors) == 0 {
				continue
			}
			labelCount := make(map[int]int)
			for _, n := range neighbors {
				labelCount[labels[n]]++
			}
			maxCount := 0
			maxLabel := labels[e.ID]
			for label, count := range labelCount {
				if count > maxCount || (count == maxCount && label < maxLabel) {
					maxCount = count
					maxLabel = label
				}
			}
			if labels[e.ID] != maxLabel {
				labels[e.ID] = maxLabel
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return ClusterResult{Clusters: labels}
}

// SocialNetworkAnalysis: Degree, Betweenness, Closeness centrality
func (s *GraphService) SocialNetworkAnalysis() SNAResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := len(s.data.Entities)
	if n == 0 {
		return SNAResult{Metrics: []SNAMetrics{}}
	}

	adj, _ := s.buildAdjacency()

	entityIDIndex := make(map[string]int)
	for i, e := range s.data.Entities {
		entityIDIndex[e.ID] = i
	}

	metrics := make([]SNAMetrics, n)
	for i, e := range s.data.Entities {
		metrics[i].EntityID = e.ID
	}

	// Degree Centrality
	maxDegree := 0
	for i, e := range s.data.Entities {
		degree := len(adj[e.ID])
		if degree > maxDegree {
			maxDegree = degree
		}
		metrics[i].DegreeCentrality = float64(degree)
	}
	if maxDegree > 0 {
		for i := range metrics {
			metrics[i].DegreeCentrality = metrics[i].DegreeCentrality / float64(n-1)
		}
	}

	// BFS-based shortest paths for betweenness and closeness
	dist := make([][]int, n)
	for i := range dist {
		dist[i] = make([]int, n)
		for j := range dist[i] {
			dist[i][j] = -1
		}
		dist[i][i] = 0
	}

	for i, e := range s.data.Entities {
		queue := []int{i}
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			currID := s.data.Entities[curr].ID
			for _, neighborID := range adj[currID] {
				j := entityIDIndex[neighborID]
				if dist[i][j] == -1 {
					dist[i][j] = dist[i][curr] + 1
					queue = append(queue, j)
				}
			}
		}
		_ = e
	}

	// Closeness Centrality
	for i := range metrics {
		totalDist := 0
		reachable := 0
		for j := 0; j < n; j++ {
			if i != j && dist[i][j] > 0 {
				totalDist += dist[i][j]
				reachable++
			}
		}
		if reachable > 0 && totalDist > 0 {
			metrics[i].ClosenessCentrality = float64(reachable) / float64(totalDist)
		}
	}

	// Betweenness Centrality (approximate via BFS shortest path counting)
	betweenness := make([]float64, n)
	for src := 0; src < n; src++ {
		for tgt := src + 1; tgt < n; tgt++ {
			if dist[src][tgt] <= 0 {
				continue
			}
			shortDist := dist[src][tgt]
			for mid := 0; mid < n; mid++ {
				if mid == src || mid == tgt {
					continue
				}
				if dist[src][mid] > 0 && dist[mid][tgt] > 0 &&
					dist[src][mid]+dist[mid][tgt] == shortDist {
					betweenness[mid]++
				}
			}
		}
	}
	maxBetweenness := 0.0
	for _, b := range betweenness {
		if b > maxBetweenness {
			maxBetweenness = b
		}
	}
	normFactor := float64((n - 1) * (n - 2) / 2)
	if normFactor == 0 {
		normFactor = 1
	}
	for i := range metrics {
		metrics[i].BetweennessCentrality = math.Round(betweenness[i]/normFactor*10000) / 10000
	}

	return SNAResult{Metrics: metrics}
}
