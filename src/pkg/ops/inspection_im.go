package ops

// InspectionIMStatus describes whether low-score inspection alerts can reach IM.
type InspectionIMStatus struct {
	IMConfigured      bool     `json:"imConfigured"`
	Channels          []string `json:"channels"`
	LowScoreThreshold int      `json:"lowScoreThreshold"`
	Hint              string   `json:"hint,omitempty"`
}

// InspectionIMStatusFromChannels builds status from enabled channel IDs.
func InspectionIMStatusFromChannels(enabled []string) InspectionIMStatus {
	st := InspectionIMStatus{
		IMConfigured:      len(enabled) > 0,
		Channels:          enabled,
		LowScoreThreshold: 85,
	}
	if !st.IMConfigured {
		st.Hint = "巡检健康分低于 85 时将尝试推送 IM，请先在「通道配置」启用飞书或钉钉。"
	}
	return st
}
