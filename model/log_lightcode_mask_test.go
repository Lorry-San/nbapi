package model

import (
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/require"
)

func TestMaskLightcodeLogs(t *testing.T) {
	lc := &Log{Quota: 100, Other: `{"lightcode":true,"group_ratio":2,"user_group_ratio":1.2,"subscription_consumed":100,"model_ratio":1}`}
	legacy := &Log{Quota: 50, TokenName: "Lightcode", Other: `{"billing_source":"subscription","group_ratio":0.5,"subscription_remain":10}`}
	normal := &Log{Quota: 70, TokenName: "x", Other: `{"billing_source":"subscription","group_ratio":2}`}
	MaskLightcodeLogs([]*Log{lc, legacy, normal, nil})

	for _, l := range []*Log{lc, legacy} {
		require.Equal(t, 0, l.Quota)
		m, _ := common.StrToMap(l.Other)
		for _, k := range lightcodeHiddenOtherKeys {
			_, ok := m[k]
			require.False(t, ok, k)
		}
		require.Equal(t, true, m["lightcode"])
	}
	m, _ := common.StrToMap(lc.Other)
	require.EqualValues(t, 1, m["model_ratio"])
	require.Equal(t, 70, normal.Quota)
	m, _ = common.StrToMap(normal.Other)
	require.EqualValues(t, 2, m["group_ratio"])
}
