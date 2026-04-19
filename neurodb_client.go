package main

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NeuroNode struct {
	ID         int               `json:"id"`
	Labels     []string          `json:"labels"`
	Properties map[string]string `json:"properties"`
}

type NeuroLink struct {
	ID         int               `json:"id"`
	HeadID     int               `json:"hid"`
	TailID     int               `json:"tid"`
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
}

type NeuroResult struct {
	Status      string
	Cursor      int
	ResultCount int
	AddNodes    int
	AddLinks    int
	ModifyNodes int
	ModifyLinks int
	DeleteNodes int
	DeleteLinks int
	Nodes       []NeuroNode
	Links       []NeuroLink
	RawEntries  []string
}

type NeuroDBClient struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
	host   string
	port   int
	alive  bool
}

func NewNeuroDBClient(host string, port int) *NeuroDBClient {
	return &NeuroDBClient{host: host, port: port}
}

func (c *NeuroDBClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		c.alive = false
		return fmt.Errorf("NeuroDB 连接失败 (%s): %w", addr, err)
	}
	c.conn = conn
	c.reader = bufio.NewReaderSize(conn, 1024*64)
	c.alive = true
	return nil
}

func (c *NeuroDBClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.alive && c.conn != nil
}

func (c *NeuroDBClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.alive = false
}

func (c *NeuroDBClient) Execute(cmd string) (*NeuroResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || !c.alive {
		return nil, fmt.Errorf("NeuroDB 未连接")
	}

	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := c.conn.Write([]byte(cmd + "\r\n"))
	if err != nil {
		c.alive = false
		return nil, fmt.Errorf("发送命令失败: %w", err)
	}

	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	return c.readResponse()
}

var statusRegex = regexp.MustCompile(
	`status:(\w+),cursor:(\d+),result:(\d+),add nodes:(\d+),add links:(\d+),modify nodes:(\d+),modify links:(\d+),delete nodes:(\d+),delete links:(\d+)`,
)

func (c *NeuroDBClient) readResponse() (*NeuroResult, error) {
	var lines []string
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.alive = false
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)

		if strings.HasPrefix(line, "status:") {
			break
		}
		if strings.HasPrefix(line, "INFO:") || strings.HasPrefix(line, "ERROR:") {
			break
		}
	}

	result := &NeuroResult{}
	statusLine := lines[len(lines)-1]

	if strings.HasPrefix(statusLine, "ERROR:") || strings.HasPrefix(statusLine, "INFO:") {
		result.Status = statusLine
		return result, nil
	}

	matches := statusRegex.FindStringSubmatch(statusLine)
	if len(matches) >= 10 {
		result.Status = matches[1]
		result.Cursor, _ = strconv.Atoi(matches[2])
		result.ResultCount, _ = strconv.Atoi(matches[3])
		result.AddNodes, _ = strconv.Atoi(matches[4])
		result.AddLinks, _ = strconv.Atoi(matches[5])
		result.ModifyNodes, _ = strconv.Atoi(matches[6])
		result.ModifyLinks, _ = strconv.Atoi(matches[7])
		result.DeleteNodes, _ = strconv.Atoi(matches[8])
		result.DeleteLinks, _ = strconv.Atoi(matches[9])
	}

	if result.ResultCount > 0 {
		for i := 0; i < result.ResultCount; i++ {
			entry := ""
			for {
				line, err := c.reader.ReadString('\n')
				if err != nil {
					break
				}
				line = strings.TrimRight(line, "\r\n")
				if strings.HasPrefix(line, "(") && strings.Contains(line, "---") {
					continue
				}
				if strings.TrimSpace(line) == "" {
					if entry != "" {
						break
					}
					continue
				}
				entry += line
			}
			if entry != "" {
				result.RawEntries = append(result.RawEntries, strings.TrimSpace(entry))
				node, link := parseEntry(entry)
				if node != nil {
					result.Nodes = append(result.Nodes, *node)
				}
				if link != nil {
					result.Links = append(result.Links, *link)
				}
			}
		}
	}
	return result, nil
}

var (
	nodeRegex = regexp.MustCompile(`ID:(\d+)\s+LABELS:(\S+)\s+PROPS:\{([^}]*)\}`)
	linkRegex = regexp.MustCompile(`ID:(\d+)\s+HEAD:(\d+)\s+TAIL:(\d+)\s+TYPE:(\S+)\s+PROPS:\{([^}]*)\}`)
)

func parseEntry(entry string) (*NeuroNode, *NeuroLink) {
	entry = strings.TrimSpace(entry)

	if linkMatch := linkRegex.FindStringSubmatch(entry); len(linkMatch) >= 6 {
		id, _ := strconv.Atoi(linkMatch[1])
		hid, _ := strconv.Atoi(linkMatch[2])
		tid, _ := strconv.Atoi(linkMatch[3])
		return nil, &NeuroLink{
			ID:         id,
			HeadID:     hid,
			TailID:     tid,
			Type:       linkMatch[4],
			Properties: parseProps(linkMatch[5]),
		}
	}

	if nodeMatch := nodeRegex.FindStringSubmatch(entry); len(nodeMatch) >= 4 {
		id, _ := strconv.Atoi(nodeMatch[1])
		return &NeuroNode{
			ID:         id,
			Labels:     []string{nodeMatch[2]},
			Properties: parseProps(nodeMatch[3]),
		}, nil
	}

	return nil, nil
}

func parseProps(raw string) map[string]string {
	props := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return props
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			val = strings.Trim(val, "\"")
			props[key] = val
		}
	}
	return props
}

func escapeNeuro(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func (c *NeuroDBClient) CreateNode(label string, props map[string]string) (*NeuroResult, error) {
	propParts := make([]string, 0, len(props))
	for k, v := range props {
		propParts = append(propParts, fmt.Sprintf(`%s:"%s"`, k, escapeNeuro(v)))
	}
	cmd := fmt.Sprintf(`CREATE (n:%s {%s}) RETURN n`, label, strings.Join(propParts, ","))
	return c.Execute(cmd)
}

func (c *NeuroDBClient) CreateRelation(headUID, tailUID, relType string, props map[string]string) (*NeuroResult, error) {
	propParts := make([]string, 0, len(props))
	for k, v := range props {
		propParts = append(propParts, fmt.Sprintf(`%s:"%s"`, k, escapeNeuro(v)))
	}
	propsStr := ""
	if len(propParts) > 0 {
		propsStr = " {" + strings.Join(propParts, ",") + "}"
	}
	cmd := fmt.Sprintf(`MATCH (a {_uid:"%s"}),(b {_uid:"%s"}) CREATE (a)-[:%s%s]->(b)`,
		escapeNeuro(headUID), escapeNeuro(tailUID), relType, propsStr)
	return c.Execute(cmd)
}

func (c *NeuroDBClient) DeleteNodeByUID(uid string) (*NeuroResult, error) {
	cmd := fmt.Sprintf(`MATCH (n {_uid:"%s"}) DETACH DELETE n`, escapeNeuro(uid))
	return c.Execute(cmd)
}

func (c *NeuroDBClient) UpdateNodeProps(uid string, props map[string]string) (*NeuroResult, error) {
	setParts := make([]string, 0, len(props))
	for k, v := range props {
		setParts = append(setParts, fmt.Sprintf(`n.%s="%s"`, k, escapeNeuro(v)))
	}
	cmd := fmt.Sprintf(`MATCH (n {_uid:"%s"}) SET %s RETURN n`, escapeNeuro(uid), strings.Join(setParts, ","))
	return c.Execute(cmd)
}

func (c *NeuroDBClient) QueryAll() (*NeuroResult, error) {
	return c.Execute("MATCH (n) RETURN n")
}

func (c *NeuroDBClient) QueryAllRelations() (*NeuroResult, error) {
	return c.Execute("MATCH (n)-[r]->(m) RETURN n,r,m")
}

func (c *NeuroDBClient) SaveDB() (*NeuroResult, error) {
	return c.Execute("savedb")
}
