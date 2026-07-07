package service

import (
	"testing"

	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/stretchr/testify/require"
)

func TestPostConsumeQuotaRejectsUnexpectedNegativeQuota(t *testing.T) {
	t.Parallel()

	err := PostConsumeQuota(&relaycommon.RelayInfo{FinalPreConsumedQuota: 0}, -1, 0, false)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid negative quota adjustment")
}
