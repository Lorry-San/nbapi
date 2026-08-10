package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/logger"
	"github.com/Lorry-San/nbapi/model"

	"github.com/gin-gonic/gin"
)

const maxLogCsvExportRows = 10000

func GetAllLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	upstreamRequestId := c.Query("upstream_request_id")
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId, upstreamRequestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	upstreamRequestId := c.Query("upstream_request_id")
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId, upstreamRequestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func parseLogQuery(c *gin.Context) (logType int, startTimestamp int64, endTimestamp int64, username string, tokenName string, modelName string, channel int, group string, requestId string, upstreamRequestId string) {
	logType, _ = strconv.Atoi(c.Query("type"))
	startTimestamp, _ = strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ = strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username = c.Query("username")
	tokenName = c.Query("token_name")
	modelName = c.Query("model_name")
	channel, _ = strconv.Atoi(c.Query("channel"))
	group = c.Query("group")
	requestId = c.Query("request_id")
	upstreamRequestId = c.Query("upstream_request_id")
	return
}

func writeLogsCsv(c *gin.Context, filename string, logs []*model.Log) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	_ = writer.Write([]string{
		"id",
		"user_id",
		"username",
		"created_at",
		"type",
		"content",
		"token_name",
		"model_name",
		"quota",
		"quota_display",
		"prompt_tokens",
		"completion_tokens",
		"use_time",
		"is_stream",
		"channel",
		"channel_name",
		"token_id",
		"group",
		"ip",
		"request_id",
		"upstream_request_id",
		"other",
	})

	for _, log := range logs {
		if err := writer.Write([]string{
			strconv.Itoa(log.Id),
			strconv.Itoa(log.UserId),
			log.Username,
			time.Unix(log.CreatedAt, 0).Format(time.RFC3339),
			strconv.Itoa(log.Type),
			log.Content,
			log.TokenName,
			log.ModelName,
			strconv.Itoa(log.Quota),
			logger.LogQuota(log.Quota),
			strconv.Itoa(log.PromptTokens),
			strconv.Itoa(log.CompletionTokens),
			strconv.Itoa(log.UseTime),
			strconv.FormatBool(log.IsStream),
			strconv.Itoa(log.ChannelId),
			log.ChannelName,
			strconv.Itoa(log.TokenId),
			log.Group,
			log.Ip,
			log.RequestId,
			log.UpstreamRequestId,
			log.Other,
		}); err != nil {
			common.SysLog("failed to write log csv: " + err.Error())
			return
		}
	}
}

func ExportAllLogs(c *gin.Context) {
	logType, startTimestamp, endTimestamp, username, tokenName, modelName, channel, group, requestId, upstreamRequestId := parseLogQuery(c)
	logs, err := model.GetAllLogsForExport(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group, requestId, upstreamRequestId, maxLogCsvExportRows)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeLogsCsv(c, fmt.Sprintf("usage-logs-%s.csv", time.Now().Format("20060102150405")), logs)
}

func ExportUserLogs(c *gin.Context) {
	logType, startTimestamp, endTimestamp, _, tokenName, modelName, _, group, requestId, upstreamRequestId := parseLogQuery(c)
	userId := c.GetInt("id")
	logs, err := model.GetUserLogsForExport(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, group, requestId, upstreamRequestId, maxLogCsvExportRows)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeLogsCsv(c, fmt.Sprintf("usage-logs-self-%s.csv", time.Now().Format("20060102150405")), logs)
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
	return
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString("username")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	quotaNum, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum.Quota,
			"rpm":   quotaNum.Rpm,
			"tpm":   quotaNum.Tpm,
			//"token": tokenNum,
		},
	})
	return
}
