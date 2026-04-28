package subscription

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/poouo/NebulaPanel/internal/db"
	"github.com/poouo/NebulaPanel/internal/logger"
)

type Node struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Transport   string `json:"transport"`
	TLS         int    `json:"tls"`
	TLSSNI      string `json:"tls_sni"`
	UUID        string `json:"uuid"`
	AlterID     int    `json:"alter_id"`
	ExtraConfig string `json:"extra_config"`
	Enabled     int    `json:"enabled"`
}

// GenerateForUser generates subscription content for a user
func GenerateForUser(userID int, format string) (string, error) {
	nodes, err := getUserNodes(userID)
	if err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes available")
	}

	// Check if there's a template
	tmpl := getDefaultTemplate(format)

	switch format {
	case "clash", "mihomo":
		return generateClash(nodes, tmpl)
	case "v2ray", "base64":
		return generateBase64(nodes)
	case "surge":
		return generateSurge(nodes, tmpl)
	default:
		return generateBase64(nodes)
	}
}

func getUserNodes(userID int) ([]Node, error) {
	query := `
		SELECT n.id, n.name, n.address, n.port, n.protocol, n.transport, 
			   n.tls, COALESCE(n.tls_sni,''), COALESCE(n.uuid,''), n.alter_id, 
			   COALESCE(n.extra_config,''), n.enabled
		FROM nodes n
		INNER JOIN user_nodes un ON n.id = un.node_id
		WHERE un.user_id = ? AND n.enabled = 1
		ORDER BY n.sort_order ASC, n.id ASC`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		err := rows.Scan(&n.ID, &n.Name, &n.Address, &n.Port, &n.Protocol,
			&n.Transport, &n.TLS, &n.TLSSNI, &n.UUID, &n.AlterID,
			&n.ExtraConfig, &n.Enabled)
		if err != nil {
			continue
		}
		nodes = append(nodes, n)
	}

	// If user has no specific nodes assigned, get all enabled nodes
	if len(nodes) == 0 {
		query = `SELECT id, name, address, port, protocol, transport, 
				 tls, COALESCE(tls_sni,''), COALESCE(uuid,''), alter_id, 
				 COALESCE(extra_config,''), enabled
				 FROM nodes WHERE enabled = 1 ORDER BY sort_order ASC, id ASC`
		rows2, err := db.DB.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows2.Close()
		for rows2.Next() {
			var n Node
			err := rows2.Scan(&n.ID, &n.Name, &n.Address, &n.Port, &n.Protocol,
				&n.Transport, &n.TLS, &n.TLSSNI, &n.UUID, &n.AlterID,
				&n.ExtraConfig, &n.Enabled)
			if err != nil {
				continue
			}
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func getDefaultTemplate(format string) string {
	var tmpl string
	db.DB.QueryRow("SELECT content FROM sub_templates WHERE is_default = 1 AND format = ?", format).Scan(&tmpl)
	return tmpl
}

func generateBase64(nodes []Node) (string, error) {
	var links []string
	for _, n := range nodes {
		link := nodeToLink(n)
		if link != "" {
			links = append(links, link)
		}
	}
	result := strings.Join(links, "\n")
	return base64.StdEncoding.EncodeToString([]byte(result)), nil
}

func nodeToLink(n Node) string {
	switch n.Protocol {
	case "vmess":
		return vmessLink(n)
	case "vless":
		return vlessLink(n)
	case "trojan":
		return trojanLink(n)
	case "ss", "shadowsocks":
		return ssLink(n)
	case "hysteria2", "hy2":
		return hy2Link(n)
	default:
		return ""
	}
}

func vmessLink(n Node) string {
	config := map[string]interface{}{
		"v":    "2",
		"ps":   n.Name,
		"add":  n.Address,
		"port": n.Port,
		"id":   n.UUID,
		"aid":  n.AlterID,
		"net":  n.Transport,
		"type": "none",
		"tls":  "",
	}
	if n.TLS == 1 {
		config["tls"] = "tls"
		if n.TLSSNI != "" {
			config["sni"] = n.TLSSNI
		}
	}
	// Parse extra config
	if n.ExtraConfig != "" {
		var extra map[string]interface{}
		if json.Unmarshal([]byte(n.ExtraConfig), &extra) == nil {
			for k, v := range extra {
				config[k] = v
			}
		}
	}
	data, _ := json.Marshal(config)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func vlessLink(n Node) string {
	params := url.Values{}
	params.Set("type", n.Transport)
	if n.TLS == 1 {
		params.Set("security", "tls")
		if n.TLSSNI != "" {
			params.Set("sni", n.TLSSNI)
		}
	}
	if n.ExtraConfig != "" {
		var extra map[string]string
		if json.Unmarshal([]byte(n.ExtraConfig), &extra) == nil {
			for k, v := range extra {
				params.Set(k, v)
			}
		}
	}
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		n.UUID, n.Address, n.Port, params.Encode(), url.PathEscape(n.Name))
}

func trojanLink(n Node) string {
	params := url.Values{}
	params.Set("type", n.Transport)
	if n.TLSSNI != "" {
		params.Set("sni", n.TLSSNI)
	}
	if n.ExtraConfig != "" {
		var extra map[string]string
		if json.Unmarshal([]byte(n.ExtraConfig), &extra) == nil {
			for k, v := range extra {
				params.Set(k, v)
			}
		}
	}
	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		n.UUID, n.Address, n.Port, params.Encode(), url.PathEscape(n.Name))
}

func ssLink(n Node) string {
	// extra_config should contain {"method":"aes-256-gcm","password":"xxx"}
	var extra map[string]string
	if n.ExtraConfig != "" {
		json.Unmarshal([]byte(n.ExtraConfig), &extra)
	}
	method := extra["method"]
	password := extra["password"]
	if method == "" {
		method = "aes-256-gcm"
	}
	if password == "" {
		password = n.UUID
	}
	userInfo := base64.StdEncoding.EncodeToString([]byte(method + ":" + password))
	return fmt.Sprintf("ss://%s@%s:%d#%s", userInfo, n.Address, n.Port, url.PathEscape(n.Name))
}

func hy2Link(n Node) string {
	params := url.Values{}
	if n.TLSSNI != "" {
		params.Set("sni", n.TLSSNI)
	}
	if n.ExtraConfig != "" {
		var extra map[string]string
		if json.Unmarshal([]byte(n.ExtraConfig), &extra) == nil {
			for k, v := range extra {
				params.Set(k, v)
			}
		}
	}
	return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s",
		n.UUID, n.Address, n.Port, params.Encode(), url.PathEscape(n.Name))
}

func generateClash(nodes []Node, tmpl string) (string, error) {
	var proxies []string
	var proxyNames []string

	for _, n := range nodes {
		proxy := nodeToClashProxy(n)
		if proxy != "" {
			proxies = append(proxies, proxy)
			proxyNames = append(proxyNames, fmt.Sprintf("      - \"%s\"", n.Name))
		}
	}

	if tmpl != "" {
		result := strings.ReplaceAll(tmpl, "{{PROXIES}}", strings.Join(proxies, "\n"))
		result = strings.ReplaceAll(result, "{{PROXY_NAMES}}", strings.Join(proxyNames, "\n"))
		return result, nil
	}

	// Default clash config
	var sb strings.Builder
	sb.WriteString("mixed-port: 7890\n")
	sb.WriteString("allow-lan: false\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n\n")
	sb.WriteString("proxies:\n")
	sb.WriteString(strings.Join(proxies, "\n"))
	sb.WriteString("\n\nproxy-groups:\n")
	sb.WriteString("  - name: \"Proxy\"\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString(strings.Join(proxyNames, "\n"))
	sb.WriteString("\n\nrules:\n")
	sb.WriteString("  - MATCH,Proxy\n")

	return sb.String(), nil
}

func nodeToClashProxy(n Node) string {
	switch n.Protocol {
	case "vmess":
		tls := "false"
		if n.TLS == 1 {
			tls = "true"
		}
		s := fmt.Sprintf("  - name: \"%s\"\n    type: vmess\n    server: %s\n    port: %d\n    uuid: %s\n    alterId: %d\n    cipher: auto\n    tls: %s\n    network: %s",
			n.Name, n.Address, n.Port, n.UUID, n.AlterID, tls, n.Transport)
		if n.TLSSNI != "" && n.TLS == 1 {
			s += fmt.Sprintf("\n    servername: %s", n.TLSSNI)
		}
		return s
	case "vless":
		s := fmt.Sprintf("  - name: \"%s\"\n    type: vless\n    server: %s\n    port: %d\n    uuid: %s\n    network: %s",
			n.Name, n.Address, n.Port, n.UUID, n.Transport)
		if n.TLS == 1 {
			s += "\n    tls: true"
			if n.TLSSNI != "" {
				s += fmt.Sprintf("\n    servername: %s", n.TLSSNI)
			}
		}
		return s
	case "trojan":
		s := fmt.Sprintf("  - name: \"%s\"\n    type: trojan\n    server: %s\n    port: %d\n    password: %s\n    network: %s",
			n.Name, n.Address, n.Port, n.UUID, n.Transport)
		if n.TLSSNI != "" {
			s += fmt.Sprintf("\n    sni: %s", n.TLSSNI)
		}
		return s
	case "ss", "shadowsocks":
		var extra map[string]string
		if n.ExtraConfig != "" {
			json.Unmarshal([]byte(n.ExtraConfig), &extra)
		}
		method := extra["method"]
		password := extra["password"]
		if method == "" {
			method = "aes-256-gcm"
		}
		if password == "" {
			password = n.UUID
		}
		return fmt.Sprintf("  - name: \"%s\"\n    type: ss\n    server: %s\n    port: %d\n    cipher: %s\n    password: %s",
			n.Name, n.Address, n.Port, method, password)
	case "hysteria2", "hy2":
		s := fmt.Sprintf("  - name: \"%s\"\n    type: hysteria2\n    server: %s\n    port: %d\n    password: %s",
			n.Name, n.Address, n.Port, n.UUID)
		if n.TLSSNI != "" {
			s += fmt.Sprintf("\n    sni: %s", n.TLSSNI)
		}
		return s
	}
	return ""
}

func generateSurge(nodes []Node, tmpl string) (string, error) {
	var proxyLines []string
	var proxyNames []string
	for _, n := range nodes {
		line := nodeToSurgeLine(n)
		if line != "" {
			proxyLines = append(proxyLines, line)
			proxyNames = append(proxyNames, n.Name)
		}
	}
	if tmpl != "" {
		result := strings.ReplaceAll(tmpl, "{{PROXIES}}", strings.Join(proxyLines, "\n"))
		result = strings.ReplaceAll(result, "{{PROXY_NAMES}}", strings.Join(proxyNames, ", "))
		return result, nil
	}
	var sb strings.Builder
	sb.WriteString("[Proxy]\n")
	sb.WriteString(strings.Join(proxyLines, "\n"))
	sb.WriteString("\n\n[Proxy Group]\nProxy = select, ")
	sb.WriteString(strings.Join(proxyNames, ", "))
	sb.WriteString("\n\n[Rule]\nFINAL,Proxy\n")
	return sb.String(), nil
}

func nodeToSurgeLine(n Node) string {
	switch n.Protocol {
	case "vmess":
		tls := ""
		if n.TLS == 1 {
			tls = ", tls=true"
			if n.TLSSNI != "" {
				tls += fmt.Sprintf(", sni=%s", n.TLSSNI)
			}
		}
		return fmt.Sprintf("%s = vmess, %s, %d, username=%s%s", n.Name, n.Address, n.Port, n.UUID, tls)
	case "trojan":
		sni := n.Address
		if n.TLSSNI != "" {
			sni = n.TLSSNI
		}
		return fmt.Sprintf("%s = trojan, %s, %d, password=%s, sni=%s", n.Name, n.Address, n.Port, n.UUID, sni)
	}
	return ""
}

// GetUserBySubToken returns user info by subscription token
func GetUserBySubToken(token string) (int, bool, error) {
	var userID int
	var enabled int
	var trafficLimit, trafficUsed int64
	var expireAt sql.NullString

	err := db.DB.QueryRow(
		"SELECT id, enabled, traffic_limit, traffic_used, expire_at FROM users WHERE sub_token = ?",
		token).Scan(&userID, &enabled, &trafficLimit, &trafficUsed, &expireAt)
	if err != nil {
		return 0, false, err
	}
	if enabled != 1 {
		return userID, false, nil
	}
	if trafficLimit > 0 && trafficUsed >= trafficLimit {
		return userID, false, nil
	}
	logger.Infof("Subscription", "User %d accessed subscription", userID)
	return userID, true, nil
}
