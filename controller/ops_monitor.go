package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

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
