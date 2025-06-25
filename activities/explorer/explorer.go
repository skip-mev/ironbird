package explorer

import (
	"bytes"
	"cosmossdk.io/errors"
	"encoding/json"
	"fmt"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"go.temporal.io/sdk/workflow"
	"net/http"
	"time"
)

const AUTOSCOUT_API_URL = "https://autoscout.services.blockscout.com/api/v1/"
const AUTOSCOUT_INSTANCES_URL = AUTOSCOUT_API_URL + "instances"

type Activity struct {
	AutoscoutApiKey string
	Client          *http.Client
}

func (a *Activity) LaunchAutoscout(ctx workflow.Context, req messages.LaunchAutoScoutRequest) (messages.LaunchAutoScoutResponse, error) {
	config := types.AutoscoutConfig{
		RPCURL:              req.EvmRpcUrl,
		RPCWSURL:            req.EvmWsUrl,
		ChainID:             req.ChainId,
		TokenSymbol:         req.Token,
		ChainName:           req.ChainName,
		ServerSize:          "xsmall",
		ChainType:           "UNSPECIFIED_CHAIN_TYPE",
		NodeType:            "GETH",
		HomeplateBackground: "radial-gradient(farthest-corner at 0% 0%, rgba(183, 148, 244, 0.80) 0%, rgba(0, 163, 196, 0.80) 100%)",
		HomeplateTextColor:  "rgb(255,255,255)rgb(255,255,255)",
		IsTestnet:           true,
		StatsEnabled:        false,
		NavigationLayout:    "VERTICAL",
		ColorTheme:          "LIGHT",
		Identicon:           "JAZZICON",
		Ads: types.AutoscoutAds{
			BannerProvider: "SLISE",
			TextProvider:   "COINZILLA",
		},
	}

	configJson, err := json.Marshal(config)
	if err != nil {
		return messages.LaunchAutoScoutResponse{}, err
	}

	a.Client.Timeout = time.Minute * 5
	autoscoutReq, err := http.NewRequest(http.MethodPost, AUTOSCOUT_INSTANCES_URL, bytes.NewReader(configJson))
	if err != nil {
		return messages.LaunchAutoScoutResponse{}, errors.Wrap(err, "failed to create request")
	}

	autoscoutReq.Header.Set("Authorization", "Bearer "+a.AutoscoutApiKey)
	autoscoutReq.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(autoscoutReq)
	if err != nil {
		return messages.LaunchAutoScoutResponse{}, err
	}
	defer resp.Body.Close()

	var autoscoutResponse types.AutoscoutInstanceCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&autoscoutResponse); err != nil {
		return messages.LaunchAutoScoutResponse{}, errors.Wrap(err, "failed to decode instance create response")
	}

	return messages.LaunchAutoScoutResponse{InstanceId: autoscoutResponse.InstanceId}, nil
}

func (a *Activity) GetAutoscoutInstance(ctx workflow.Context, req messages.GetAutoScoutInstanceRequest) (messages.GetAutoScoutInstanceResponse, error) {
	autoscoutReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", AUTOSCOUT_INSTANCES_URL, req.InstanceId), nil)
	if err != nil {
		return messages.GetAutoScoutInstanceResponse{}, errors.Wrap(err, "failed to create request")
	}

	autoscoutReq.Header.Set("Authorization", "Bearer "+a.AutoscoutApiKey)
	autoscoutReq.Header.Set("Content-Type", "application/json")
	resp, err := a.Client.Do(autoscoutReq)
	if err != nil {
		return messages.GetAutoScoutInstanceResponse{}, errors.Wrap(err, "failed to get instance from autoscout")
	}
	defer resp.Body.Close()

	var autoscoutResponse types.AutoscoutInstanceGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&autoscoutResponse); err != nil {
		return messages.GetAutoScoutInstanceResponse{}, errors.Wrap(err, "failed to decode instance get response")
	}

	return messages.GetAutoScoutInstanceResponse{InstanceId: autoscoutResponse.InstanceId, Config: autoscoutResponse.Config, URL: autoscoutResponse.BlockscoutUrl}, nil
}

func (a *Activity) StopAutoScout() {

}
