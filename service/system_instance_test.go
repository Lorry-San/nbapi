package service

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
)

func TestReportCurrentSystemInstanceSkipsStandby(t *testing.T) {
	originalIsMasterNode := common.IsMasterNode
	t.Cleanup(func() {
		common.IsMasterNode = originalIsMasterNode
	})

	common.IsMasterNode = false
	assert.NoError(t, ReportCurrentSystemInstance())
}
