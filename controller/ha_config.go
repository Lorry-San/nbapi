package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/Lorry-San/nbapi/setting/ha_setting"

	"github.com/gin-gonic/gin"
)

const haProbeTimeout = 3 * time.Second

type HAOverview struct {
	Config      ha_setting.HASetting           `json:"config"`
	CurrentNode HACurrentNode                  `json:"current_node"`
	Instances   []model.SystemInstanceResponse `json:"instances"`
	Probes      []HAHealthProbe                `json:"probes"`
	Checks      []HACheck                      `json:"checks"`
	Summary     string                         `json:"summary"`
	Snippets    HASnippets                     `json:"snippets"`
}

type HACurrentNode struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	IsMaster bool   `json:"is_master"`
}

type HACheck struct {
	Level   string `json:"level"`
	Key     string `json:"key"`
	Message string `json:"message"`
}

type HAHealthProbe struct {
	Target     string         `json:"target"`
	URL        string         `json:"url"`
	Reachable  bool           `json:"reachable"`
	StatusCode int            `json:"status_code,omitempty"`
	Success    *bool          `json:"success,omitempty"`
	Message    string         `json:"message,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

type HASnippets struct {
	PrimaryEnv       string `json:"primary_env"`
	StandbyEnv       string `json:"standby_env"`
	ComposeEnv       string `json:"compose_env"`
	CutoverChecklist string `json:"cutover_checklist"`
}

func GetHAOverview(c *gin.Context) {
	overview, err := buildHAOverview(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    overview,
	})
}

func UpdateHAConfig(c *gin.Context) {
	var payload ha_setting.HASetting
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request body",
		})
		return
	}

	payload = ha_setting.Normalize(payload)
	if err := ha_setting.Validate(payload); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	values := map[string]string{
		"ha_setting.enabled":                       strconv.FormatBool(payload.Enabled),
		"ha_setting.primary_node_name":             payload.PrimaryNodeName,
		"ha_setting.standby_node_name":             payload.StandbyNodeName,
		"ha_setting.primary_health_url":            payload.PrimaryHealthURL,
		"ha_setting.standby_health_url":            payload.StandbyHealthURL,
		"ha_setting.public_entry":                  payload.PublicEntry,
		"ha_setting.origin_entry":                  payload.OriginEntry,
		"ha_setting.primary_origin":                payload.PrimaryOrigin,
		"ha_setting.standby_origin":                payload.StandbyOrigin,
		"ha_setting.dns_provider":                  payload.DNSProvider,
		"ha_setting.dns_record_name":               payload.DNSRecordName,
		"ha_setting.database_engine":               payload.DatabaseEngine,
		"ha_setting.replication_mode":              payload.ReplicationMode,
		"ha_setting.redis_mode":                    payload.RedisMode,
		"ha_setting.failover_strategy":             payload.FailoverStrategy,
		"ha_setting.health_check_interval_seconds": strconv.Itoa(payload.HealthCheckIntervalSeconds),
		"ha_setting.failover_threshold":            strconv.Itoa(payload.FailoverThreshold),
		"ha_setting.cutover_runbook":               payload.CutoverRunbook,
		"ha_setting.rollback_runbook":              payload.RollbackRunbook,
		"ha_setting.notes":                         payload.Notes,
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiError(c, err)
		return
	}

	overview, err := buildHAOverview(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "option.update", map[string]interface{}{"key": "ha_setting"})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    overview,
	})
}

func buildHAOverview(ctx context.Context) (HAOverview, error) {
	cfg := ha_setting.GetHASetting()
	cfg = ha_setting.Normalize(cfg)
	now := common.GetTimestamp()
	identity := common.GetNodeIdentity()
	instances, err := model.ListSystemInstances()
	if err != nil {
		return HAOverview{}, err
	}

	responses := make([]model.SystemInstanceResponse, 0, len(instances))
	for _, instance := range instances {
		responses = append(responses, instance.ToResponse(now))
	}

	checks := buildHAChecks(cfg, responses)
	probes := []HAHealthProbe{}
	if cfg.PrimaryHealthURL != "" {
		probe := probeHAHealth(ctx, "primary", cfg.PrimaryHealthURL)
		probes = append(probes, probe)
		checks = append(checks, checkHAProbe("primary", probe, true))
	}
	if cfg.StandbyHealthURL != "" {
		probe := probeHAHealth(ctx, "standby", cfg.StandbyHealthURL)
		probes = append(probes, probe)
		checks = append(checks, checkHAProbe("standby", probe, false))
	}

	return HAOverview{
		Config: cfg,
		CurrentNode: HACurrentNode{
			Name:     identity.Name,
			Source:   identity.Source,
			IsMaster: common.IsMasterNode,
		},
		Instances: responses,
		Probes:    probes,
		Checks:    checks,
		Summary:   summarizeHAChecks(cfg.Enabled, checks),
		Snippets:  buildHASnippets(cfg),
	}, nil
}

func buildHAChecks(cfg ha_setting.HASetting, instances []model.SystemInstanceResponse) []HACheck {
	checks := []HACheck{}
	if !cfg.Enabled {
		return append(checks, HACheck{
			Level:   "warn",
			Key:     "ha_disabled",
			Message: "Primary/standby configuration is disabled.",
		})
	}
	if cfg.PrimaryNodeName == "" {
		checks = append(checks, HACheck{Level: "error", Key: "primary_missing", Message: "Primary node name is empty."})
	}
	if cfg.StandbyNodeName == "" {
		checks = append(checks, HACheck{Level: "error", Key: "standby_missing", Message: "Standby node name is empty."})
	}
	if cfg.PrimaryNodeName != "" && strings.EqualFold(cfg.PrimaryNodeName, cfg.StandbyNodeName) {
		checks = append(checks, HACheck{Level: "error", Key: "duplicate_node_names", Message: "Primary and standby node names must be different."})
	}
	if cfg.PrimaryHealthURL == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "primary_health_url_missing", Message: "Primary health URL is empty."})
	}
	if cfg.StandbyHealthURL == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "standby_health_url_missing", Message: "Standby health URL is empty."})
	}
	if cfg.PublicEntry == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "public_entry_missing", Message: "Public entry is empty."})
	}
	if cfg.OriginEntry == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "origin_entry_missing", Message: "Origin entry is empty."})
	}
	if cfg.DNSProvider == "cloudflare" && cfg.DNSRecordName == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "dns_record_name_missing", Message: "Cloudflare DNS record name is empty."})
	}
	if cfg.PrimaryOrigin == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "primary_origin_missing", Message: "Primary origin target is empty."})
	}
	if cfg.StandbyOrigin == "" {
		checks = append(checks, HACheck{Level: "warn", Key: "standby_origin_missing", Message: "Standby origin target is empty."})
	}
	if cfg.PrimaryOrigin != "" && cfg.StandbyOrigin != "" && strings.EqualFold(cfg.PrimaryOrigin, cfg.StandbyOrigin) {
		checks = append(checks, HACheck{Level: "warn", Key: "same_origin_target", Message: "Primary and standby origins point to the same target."})
	}
	if cfg.FailoverStrategy == "assisted" && (cfg.PrimaryHealthURL == "" || cfg.StandbyHealthURL == "") {
		checks = append(checks, HACheck{Level: "warn", Key: "assisted_failover_without_health", Message: "Assisted failover needs both health URLs to make reliable decisions."})
	}

	masterCount := 0
	var primary *model.SystemInstanceResponse
	var standby *model.SystemInstanceResponse
	for i := range instances {
		instance := &instances[i]
		if instanceIsMaster(*instance) {
			masterCount++
		}
		if cfg.PrimaryNodeName != "" && sameNodeName(*instance, cfg.PrimaryNodeName) {
			primary = instance
		}
		if cfg.StandbyNodeName != "" && sameNodeName(*instance, cfg.StandbyNodeName) {
			standby = instance
		}
	}

	switch {
	case masterCount == 0:
		checks = append(checks, HACheck{Level: "error", Key: "no_master", Message: "No reporting instance is marked as master."})
	case masterCount > 1:
		checks = append(checks, HACheck{Level: "error", Key: "multiple_masters", Message: fmt.Sprintf("%d instances are marked as master.", masterCount)})
	default:
		checks = append(checks, HACheck{Level: "ok", Key: "single_master", Message: "Exactly one reporting instance is marked as master."})
	}

	if primary == nil {
		checks = append(checks, HACheck{Level: "warn", Key: "primary_not_seen", Message: "Configured primary node has not reported a heartbeat."})
	} else {
		checks = append(checks, checkInstanceRole("primary", *primary, true))
		checks = append(checks, checkInstanceFreshness("primary", *primary))
	}
	if standby == nil {
		checks = append(checks, HACheck{Level: "warn", Key: "standby_not_seen", Message: "Configured standby node has not reported a heartbeat. A read-only standby may need a dedicated status reporter."})
	} else {
		checks = append(checks, checkInstanceRole("standby", *standby, false))
		checks = append(checks, checkInstanceFreshness("standby", *standby))
	}

	return checks
}

func checkInstanceRole(label string, instance model.SystemInstanceResponse, wantMaster bool) HACheck {
	isMaster := instanceIsMaster(instance)
	if isMaster == wantMaster {
		return HACheck{Level: "ok", Key: label + "_role", Message: fmt.Sprintf("%s node role matches the configured topology.", titleHALabel(label))}
	}
	if wantMaster {
		return HACheck{Level: "error", Key: label + "_role", Message: "Primary node is reporting as worker/slave."}
	}
	return HACheck{Level: "error", Key: label + "_role", Message: "Standby node is reporting as master."}
}

func checkInstanceFreshness(label string, instance model.SystemInstanceResponse) HACheck {
	if instance.Status == model.SystemInstanceStatusOnline {
		return HACheck{Level: "ok", Key: label + "_heartbeat", Message: fmt.Sprintf("%s node heartbeat is fresh.", titleHALabel(label))}
	}
	return HACheck{Level: "warn", Key: label + "_heartbeat", Message: fmt.Sprintf("%s node heartbeat is stale.", titleHALabel(label))}
}

func probeHAHealth(ctx context.Context, target string, rawURL string) HAHealthProbe {
	ctx, cancel := context.WithTimeout(ctx, haProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return HAHealthProbe{Target: target, URL: rawURL, Message: err.Error()}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return HAHealthProbe{Target: target, URL: rawURL, Message: err.Error()}
	}
	defer res.Body.Close()

	var body struct {
		Success bool           `json:"success"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := common.DecodeJson(res.Body, &body); err != nil {
		return HAHealthProbe{
			Target:     target,
			URL:        rawURL,
			Reachable:  true,
			StatusCode: res.StatusCode,
			Message:    "health endpoint returned non-JSON response",
		}
	}
	return HAHealthProbe{
		Target:     target,
		URL:        rawURL,
		Reachable:  true,
		StatusCode: res.StatusCode,
		Success:    &body.Success,
		Message:    body.Message,
		Data:       body.Data,
	}
}

func checkHAProbe(label string, probe HAHealthProbe, wantPrimary bool) HACheck {
	if !probe.Reachable {
		return HACheck{Level: "warn", Key: label + "_health", Message: fmt.Sprintf("%s health endpoint is unreachable: %s", titleHALabel(label), probe.Message)}
	}
	role := strings.ToLower(fmt.Sprint(probe.Data["databaseRole"]))
	database := strings.ToLower(fmt.Sprint(probe.Data["database"]))
	if wantPrimary {
		if probe.Success != nil && *probe.Success && role == "primary" && database == "ok" {
			return HACheck{Level: "ok", Key: label + "_health", Message: "Primary health endpoint reports a writable primary database."}
		}
		return HACheck{Level: "error", Key: label + "_health", Message: "Primary health endpoint does not report a writable primary database."}
	}
	if role == "standby" || database == "read_only" {
		return HACheck{Level: "ok", Key: label + "_health", Message: "Standby health endpoint reports a read-only standby database."}
	}
	if probe.Success != nil && *probe.Success && role == "primary" {
		return HACheck{Level: "error", Key: label + "_health", Message: "Standby health endpoint reports a writable primary database."}
	}
	return HACheck{Level: "warn", Key: label + "_health", Message: "Standby health endpoint did not return a recognizable standby status."}
}

func summarizeHAChecks(enabled bool, checks []HACheck) string {
	if !enabled {
		return "disabled"
	}
	summary := "ok"
	for _, check := range checks {
		if check.Level == "error" {
			return "error"
		}
		if check.Level == "warn" {
			summary = "warn"
		}
	}
	return summary
}

func buildHASnippets(cfg ha_setting.HASetting) HASnippets {
	primaryName := cfg.PrimaryNodeName
	if primaryName == "" {
		primaryName = "nbapi-main"
	}
	standbyName := cfg.StandbyNodeName
	if standbyName == "" {
		standbyName = "nbapi-backup"
	}
	primaryEnv := fmt.Sprintf("NODE_NAME=%s\nNODE_TYPE=master", primaryName)
	if cfg.PublicEntry != "" {
		primaryEnv += fmt.Sprintf("\n# PUBLIC_ENTRY=%s", cfg.PublicEntry)
	}
	if cfg.PrimaryOrigin != "" {
		primaryEnv += fmt.Sprintf("\n# PRIMARY_ORIGIN=%s", cfg.PrimaryOrigin)
	}
	standbyEnv := fmt.Sprintf("NODE_NAME=%s\nNODE_TYPE=slave", standbyName)
	if cfg.OriginEntry != "" {
		standbyEnv += fmt.Sprintf("\n# ORIGIN_ENTRY=%s", cfg.OriginEntry)
	}
	if cfg.StandbyOrigin != "" {
		standbyEnv += fmt.Sprintf("\n# STANDBY_ORIGIN=%s", cfg.StandbyOrigin)
	}
	return HASnippets{
		PrimaryEnv: primaryEnv,
		StandbyEnv: standbyEnv,
		ComposeEnv: fmt.Sprintf("environment:\n  NODE_NAME: ${NODE_NAME:-%s}\n  NODE_TYPE: ${NODE_TYPE:-master}", primaryName),
		CutoverChecklist: buildHACutoverChecklist(cfg),
	}
}

func buildHACutoverChecklist(cfg ha_setting.HASetting) string {
	items := []string{
		"1. Confirm primary health endpoint is unhealthy and standby database is ready.",
		"2. Promote the standby database or complete the external database failover.",
		"3. Start the standby NBAPI node with NODE_TYPE=master.",
		"4. Point the public entry to the standby origin.",
		"5. Verify /api/ha/health and a real chat request before opening traffic.",
	}
	if cfg.PublicEntry != "" {
		items[3] = fmt.Sprintf("4. Point %s to the standby origin.", cfg.PublicEntry)
	}
	if cfg.StandbyOrigin != "" {
		items[4] = fmt.Sprintf("5. Verify %s /api/ha/health and a real chat request before opening traffic.", cfg.StandbyOrigin)
	}
	if cfg.CutoverRunbook != "" {
		items = append(items, "", "Custom cutover runbook:", cfg.CutoverRunbook)
	}
	if cfg.RollbackRunbook != "" {
		items = append(items, "", "Rollback runbook:", cfg.RollbackRunbook)
	}
	return strings.Join(items, "\n")
}

func sameNodeName(instance model.SystemInstanceResponse, name string) bool {
	return strings.EqualFold(instance.NodeName, name) || strings.EqualFold(fmt.Sprint(nestedInfoValue(instance, "node", "name")), name)
}

func instanceIsMaster(instance model.SystemInstanceResponse) bool {
	value := nestedInfoValue(instance, "role", "is_master")
	isMaster, _ := value.(bool)
	return isMaster
}

func nestedInfoValue(instance model.SystemInstanceResponse, keys ...string) any {
	var value any = instance.Info
	for _, key := range keys {
		next, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = next[key]
	}
	return value
}

func titleHALabel(label string) string {
	if label == "" {
		return ""
	}
	return strings.ToUpper(label[:1]) + label[1:]
}
