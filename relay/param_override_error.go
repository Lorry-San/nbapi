package relay

import (
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/Lorry-San/nbapi/types"
)

func nbapiErrorFromParamOverride(err error) *types.NBAPIError {
	if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
		return relaycommon.NBAPIErrorFromParamOverride(fixedErr)
	}
	return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
}
