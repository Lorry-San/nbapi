package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/Lorry-San/nbapi/service"

	"github.com/gin-gonic/gin"
)

func GetOpsMonitor(c *gin.Context) {
	endTs, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	startTs, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	if endTs <= 0 {
		endTs = time.Now().Unix()
	}
	if startTs <= 0 {
		rangeSeconds, _ := strconv.ParseInt(c.DefaultQuery("range_seconds", "3600"), 10, 64)
		if rangeSeconds <= 0 {
			rangeSeconds = 3600
		}
		startTs = endTs - rangeSeconds
	}
	channelId, _ := strconv.Atoi(c.Query("channel"))
	result, err := model.GetOpsMonitor(
		startTs,
		endTs,
		channelId,
		c.Query("model_name"),
		c.Query("group"),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

func ListOpsAlertRules(c *gin.Context) {
	if err := model.SeedDefaultOpsAlertRules(); err != nil {
		common.ApiError(c, err)
		return
	}
	rules, err := model.ListOpsAlertRules()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"rules":   rules,
		"metrics": model.GetOpsAlertMetricsMetadata(),
	})
}

func CreateOpsAlertRule(c *gin.Context) {
	var req model.OpsAlertRuleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	rule, err := model.CreateOpsAlertRule(req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rule)
}

func UpdateOpsAlertRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req model.OpsAlertRuleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	rule, err := model.UpdateOpsAlertRule(id, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rule)
}

func DeleteOpsAlertRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteOpsAlertRule(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ListOpsAlertEvents(c *gin.Context) {
	now := time.Now().Unix()
	rangeSeconds, _ := strconv.ParseInt(c.DefaultQuery("range_seconds", "604800"), 10, 64)
	if rangeSeconds <= 0 || rangeSeconds > 30*24*3600 {
		rangeSeconds = 7 * 24 * 3600
	}
	endTs, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	startTs, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	if endTs <= 0 {
		endTs = now
	}
	if startTs <= 0 {
		startTs = endTs - rangeSeconds
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	ruleId, _ := strconv.Atoi(c.Query("rule_id"))
	events, err := model.ListOpsAlertEvents(model.OpsAlertEventQuery{
		StartTs: startTs,
		EndTs:   endTs,
		Level:   c.Query("level"),
		Status:  c.Query("status"),
		RuleId:  ruleId,
		Limit:   limit,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, events)
}

func TriggerOpsAlertEvaluation(c *gin.Context) {
	if err := service.RunOpsAlertEvaluationOnce(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"message": "ops alert evaluation completed"})
}
