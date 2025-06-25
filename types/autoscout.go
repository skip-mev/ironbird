package types

type AutoscoutInstanceCreateResponse struct {
	InstanceId string `json:"instance_id"`
}

type AutoscoutInstanceGetResponse struct {
	InstanceId    string          `json:"instance_id"`
	Config        AutoscoutConfig `json:"config"`
	BlockscoutUrl string          `json:"blockscout_url"`
}

// Config holds all configuration fields.
type AutoscoutConfig struct {
	RPCURL              string       `json:"rpc_url"`
	ServerSize          string       `json:"server_size"`
	ChainType           string       `json:"chain_type"`
	NodeType            string       `json:"node_type"`
	ChainID             string       `json:"chain_id"`
	TokenSymbol         string       `json:"token_symbol"`
	ChainName           string       `json:"chain_name"`
	HomeplateBackground string       `json:"homeplate_background"`
	HomeplateTextColor  string       `json:"homeplate_text_color"`
	IsTestnet           bool         `json:"is_testnet"`
	StatsEnabled        bool         `json:"stats_enabled"`
	RPCWSURL            string       `json:"rpc_ws_url"`
	NavigationLayout    string       `json:"navigation_layout"`
	ColorTheme          string       `json:"color_theme"`
	Identicon           string       `json:"identicon"`
	Ads                 AutoscoutAds `json:"ads"`
}

// AutoscoutAds providers configuration.
type AutoscoutAds struct {
	TextProvider   string `json:"text_provider"`
	BannerProvider string `json:"banner_provider"`
}
