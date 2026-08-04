package ha_setting

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Lorry-San/nbapi/setting/config"
)

type HASetting struct {
	Enabled                    bool   `json:"enabled"`
	PrimaryNodeName            string `json:"primary_node_name"`
	StandbyNodeName            string `json:"standby_node_name"`
	PrimaryHealthURL           string `json:"primary_health_url"`
	StandbyHealthURL           string `json:"standby_health_url"`
	PublicEntry                string `json:"public_entry"`
	OriginEntry                string `json:"origin_entry"`
	PrimaryOrigin              string `json:"primary_origin"`
	StandbyOrigin              string `json:"standby_origin"`
	DNSProvider                string `json:"dns_provider"`
	DNSRecordName              string `json:"dns_record_name"`
	DatabaseEngine             string `json:"database_engine"`
	ReplicationMode            string `json:"replication_mode"`
	RedisMode                  string `json:"redis_mode"`
	FailoverStrategy           string `json:"failover_strategy"`
	HealthCheckIntervalSeconds int    `json:"health_check_interval_seconds"`
	FailoverThreshold          int    `json:"failover_threshold"`
	CutoverRunbook             string `json:"cutover_runbook"`
	RollbackRunbook            string `json:"rollback_runbook"`
	Notes                      string `json:"notes"`
}

var haSetting = HASetting{
	Enabled:                    false,
	PrimaryNodeName:            "nbapi-main",
	StandbyNodeName:            "nbapi-backup",
	PrimaryHealthURL:           "",
	StandbyHealthURL:           "",
	PublicEntry:                "api.lcapi.online",
	OriginEntry:                "o-api.lcapi.online",
	PrimaryOrigin:              "",
	StandbyOrigin:              "",
	DNSProvider:                "cloudflare",
	DNSRecordName:              "",
	DatabaseEngine:             "postgresql",
	ReplicationMode:            "external",
	RedisMode:                  "shared",
	FailoverStrategy:           "manual",
	HealthCheckIntervalSeconds: 30,
	FailoverThreshold:          3,
	CutoverRunbook:             "",
	RollbackRunbook:            "",
	Notes:                      "",
}

func init() {
	config.GlobalConfig.Register("ha_setting", &haSetting)
}

func GetHASetting() HASetting {
	return haSetting
}

func Normalize(setting HASetting) HASetting {
	setting.PrimaryNodeName = strings.TrimSpace(setting.PrimaryNodeName)
	setting.StandbyNodeName = strings.TrimSpace(setting.StandbyNodeName)
	setting.PrimaryHealthURL = strings.TrimSpace(setting.PrimaryHealthURL)
	setting.StandbyHealthURL = strings.TrimSpace(setting.StandbyHealthURL)
	setting.PublicEntry = strings.TrimSpace(setting.PublicEntry)
	setting.OriginEntry = strings.TrimSpace(setting.OriginEntry)
	setting.PrimaryOrigin = strings.TrimSpace(setting.PrimaryOrigin)
	setting.StandbyOrigin = strings.TrimSpace(setting.StandbyOrigin)
	setting.DNSProvider = normalizeEnum(setting.DNSProvider, "cloudflare")
	setting.DNSRecordName = strings.TrimSpace(setting.DNSRecordName)
	setting.DatabaseEngine = normalizeEnum(setting.DatabaseEngine, "postgresql")
	setting.ReplicationMode = normalizeEnum(setting.ReplicationMode, "external")
	setting.RedisMode = normalizeEnum(setting.RedisMode, "shared")
	setting.FailoverStrategy = normalizeEnum(setting.FailoverStrategy, "manual")
	if setting.HealthCheckIntervalSeconds <= 0 {
		setting.HealthCheckIntervalSeconds = 30
	}
	if setting.FailoverThreshold <= 0 {
		setting.FailoverThreshold = 3
	}
	setting.CutoverRunbook = strings.TrimSpace(setting.CutoverRunbook)
	setting.RollbackRunbook = strings.TrimSpace(setting.RollbackRunbook)
	setting.Notes = strings.TrimSpace(setting.Notes)
	return setting
}

func Validate(setting HASetting) error {
	if setting.Enabled {
		if setting.PrimaryNodeName == "" {
			return fmt.Errorf("primary node name is required")
		}
		if setting.StandbyNodeName == "" {
			return fmt.Errorf("standby node name is required")
		}
		if strings.EqualFold(setting.PrimaryNodeName, setting.StandbyNodeName) {
			return fmt.Errorf("primary and standby node names must be different")
		}
	}
	if !isAllowed(setting.DatabaseEngine, "postgresql", "mysql", "sqlite", "external") {
		return fmt.Errorf("invalid database engine: %s", setting.DatabaseEngine)
	}
	if !isAllowed(setting.ReplicationMode, "external", "postgres_streaming", "mysql_replica", "managed", "manual") {
		return fmt.Errorf("invalid replication mode: %s", setting.ReplicationMode)
	}
	if !isAllowed(setting.RedisMode, "shared", "sentinel", "primary_standby", "disabled") {
		return fmt.Errorf("invalid redis mode: %s", setting.RedisMode)
	}
	if !isAllowed(setting.FailoverStrategy, "manual", "assisted") {
		return fmt.Errorf("invalid failover strategy: %s", setting.FailoverStrategy)
	}
	if !isAllowed(setting.DNSProvider, "cloudflare", "manual", "other") {
		return fmt.Errorf("invalid DNS provider: %s", setting.DNSProvider)
	}
	if setting.HealthCheckIntervalSeconds < 5 || setting.HealthCheckIntervalSeconds > 3600 {
		return fmt.Errorf("health check interval must be between 5 and 3600 seconds")
	}
	if setting.FailoverThreshold < 1 || setting.FailoverThreshold > 100 {
		return fmt.Errorf("failover threshold must be between 1 and 100")
	}
	for _, rawURL := range []string{setting.PrimaryHealthURL, setting.StandbyHealthURL} {
		if rawURL == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(rawURL)
		if err != nil {
			return fmt.Errorf("invalid health URL: %s", rawURL)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("health URL must use http or https: %s", rawURL)
		}
		if parsed.Host == "" {
			return fmt.Errorf("health URL must include a host: %s", rawURL)
		}
	}
	return nil
}

func normalizeEnum(value string, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func isAllowed(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
