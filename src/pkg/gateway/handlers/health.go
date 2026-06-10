package handlers

import (
	"time"

	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/rbac"
)

// HealthSnapshot is a minimal health payload (compatible with protocol).
type HealthSnapshot struct {
	OK           bool                   `json:"ok"`
	Ts           int64                  `json:"ts"`
	DurationMs   int64                  `json:"durationMs"`
	Channels     map[string]interface{} `json:"channels"`
	ChannelOrder []string               `json:"channelOrder"`
	Agents       []interface{}          `json:"agents"`
	Sessions     interface{}            `json:"sessions"`
}

// HealthHandler handles the "health" method.
func HealthHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "health context not configured",
		}, nil)
		return nil
	}
	wantsProbe := false
	if p, ok := opts.Params["probe"].(bool); ok {
		wantsProbe = p
	}
	var snap interface{}
	if !wantsProbe && ctx.GetHealthCache != nil {
		snap = ctx.GetHealthCache()
	}
	if snap == nil && ctx.RefreshHealth != nil {
		s, err := ctx.RefreshHealth(wantsProbe)
		if err != nil {
			opts.Respond(false, nil, &protocol.ErrorShape{
				Code:    protocol.ErrCodeServiceUnavailable,
				Message: err.Error(),
			}, nil)
			return nil
		}
		snap = s
	}
	if snap == nil {
		snap = &HealthSnapshot{
			OK:           true,
			Ts:           time.Now().UnixMilli(),
			DurationMs:   0,
			Channels:     map[string]interface{}{},
			ChannelOrder: []string{},
			Agents:       []interface{}{},
			Sessions:     map[string]interface{}{},
		}
	}
	meta := map[string]interface{}{}
	if ctx.GetHealthCache != nil && ctx.GetHealthCache() != nil {
		meta["cached"] = true
	}
	opts.Respond(true, snap, nil, meta)
	return nil
}

// StatusHandler handles the "status" method.
func StatusHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx != nil && ctx.GetStatusSummary != nil {
		summary, err := ctx.GetStatusSummary()
		if err != nil {
			opts.Respond(false, nil, &protocol.ErrorShape{
				Code:    protocol.ErrCodeInternal,
				Message: err.Error(),
			}, nil)
			return nil
		}
		opts.Respond(true, summary, nil, nil)
		return nil
	}
	// Phase 2c: return minimal status when no context
	opts.Respond(true, DefaultStatusSummary(), nil, nil)
	return nil
}

// NewRegistry returns a registry with all handlers.
// Methods aligned with src/gateway/server-methods-list.ts BASE_METHODS.
func NewRegistry(ctx *Context) Registry {
	r := Registry{
		"health":                     {Handler: HealthHandler},
		"logs.tail":                  {Handler: LogsTailHandler},
		"channels.status":            {Handler: ChannelsStatusHandler},
		"channels.logout":            {Handler: ChannelsLogoutHandler},
		"channels.wework.qr.start":   {Handler: WeWorkQRStartHandler},
		"channels.wework.qr.poll":    {Handler: WeWorkQRPollHandler},
		"channels.weixin.qr.start":   {Handler: WeixinQRStartHandler},
		"channels.weixin.qr.poll":    {Handler: WeixinQRPollHandler},
		"status":                     {Handler: StatusHandler},
		"usage.status":               {Handler: UsageStatusHandler},
		"usage.cost":                 {Handler: UsageCostHandler},
		"tts.status":                 {Handler: TtsStatusHandler},
		"tts.providers":              {Handler: TtsProvidersHandler},
		"tts.enable":                 {Handler: TtsEnableHandler},
		"tts.disable":                {Handler: TtsDisableHandler},
		"tts.convert":                {Handler: TtsConvertHandler},
		"tts.setProvider":            {Handler: TtsSetProviderHandler},
		"config.get":                 {Handler: ConfigGetHandler, RequiredPermission: "menu:config"},
		"config.env":                 {Handler: ConfigEnvHandler, RequiredPermission: "menu:config"},
		"config.set":                 {Handler: ConfigSetHandler, RequiredPermission: "menu:config"},
		"config.apply":               {Handler: ConfigApplyHandler, RequiredPermission: "menu:config"},
		"config.patch":               {Handler: ConfigPatchHandler, RequiredPermission: "menu:config"},
		"mcp.servers.delete":         {Handler: McpServersDeleteHandler, RequiredPermission: "menu:config"},
		"config.schema":              {Handler: ConfigSchemaHandler, RequiredPermission: "menu:config"},
		"exec.approvals.get":         {Handler: ExecApprovalsGetHandler},
		"exec.approvals.set":         {Handler: ExecApprovalsSetHandler, RequiredPermission: "ops:ack"},
		"exec.approvals.node.get":    {Handler: ExecApprovalsNodeGetHandler},
		"exec.approvals.node.set":    {Handler: ExecApprovalsNodeSetHandler, RequiredPermission: "ops:ack"},
		"exec.approval.request":      {Handler: ExecApprovalRequestHandler},
		"exec.approval.resolve":      {Handler: ExecApprovalResolveHandler, RequiredPermission: "ops:ack"},
		"wizard.start":               {Handler: WizardStubHandler},
		"wizard.next":                {Handler: WizardStubHandler},
		"wizard.cancel":              {Handler: WizardStubHandler},
		"wizard.status":              {Handler: WizardStubHandler},
		"talk.mode":                  {Handler: TalkModeHandler},
		"models.list":                {Handler: ModelsListHandler},
		"agents.list":                {Handler: AgentsListHandler},
		"agents.create":              {Handler: AgentsCreateHandler, RequiredPermission: "menu:config"},
		"agents.update":              {Handler: AgentsUpdateHandler, RequiredPermission: "menu:config"},
		"agents.delete":              {Handler: AgentsDeleteHandler, RequiredPermission: "menu:config"},
		"agents.files.list":          {Handler: AgentsFilesListHandler},
		"agents.files.get":           {Handler: AgentsFilesGetHandler},
		"agents.files.set":           {Handler: AgentsFilesSetHandler, RequiredPermission: "menu:config"},
		"employees.list":             {Handler: EmployeesListHandler},
		"employees.get":              {Handler: EmployeesGetHandler},
		"employees.create":           {Handler: EmployeesCreateHandler, RequiredPermission: "menu:config"},
		"employees.delete":           {Handler: EmployeesDeleteHandler, RequiredPermission: "menu:config"},
		"employee.tasks.list":        {Handler: EmployeeTasksListHandler},
		"employee.tasks.get":         {Handler: EmployeeTasksGetHandler},
		"employee.tasks.create":      {Handler: EmployeeTasksCreateHandler, RequiredPermission: "menu:config"},
		"employee.tasks.update":      {Handler: EmployeeTasksUpdateHandler, RequiredPermission: "menu:config"},
		"employee.tasks.delete":      {Handler: EmployeeTasksDeleteHandler, RequiredPermission: "menu:config"},
		"employee.effectiveness.get": {Handler: EmployeeEffectivenessGetHandler},
		"skills.status":              {Handler: SkillsStatusHandler},
		"skills.getDoc":              {Handler: SkillsGetDocHandler},
		"skills.bins":                {Handler: SkillsBinsHandler},
		"skills.install":             {Handler: SkillsInstallHandler, RequiredPermission: "menu:config"},
		"skills.update":              {Handler: SkillsUpdateHandler, RequiredPermission: "menu:config"},
		"skills.delete":              {Handler: SkillsDeleteHandler, RequiredPermission: "menu:config"},
		"skills.listFiles":           {Handler: SkillsListFilesHandler},
		"skills.getFile":             {Handler: SkillsGetFileHandler},
		"skills.saveFile":            {Handler: SkillsSaveFileHandler, RequiredPermission: "menu:config"},
		"files.read":                 {Handler: FilesReadHandler},
		"update.run":                 {Handler: UpdateRunHandler, RequiredPermission: "menu:config"},
		"voicewake.get":              {Handler: VoicewakeGetHandler},
		"voicewake.set":              {Handler: VoicewakeSetHandler, RequiredPermission: "menu:config"},
		"sessions.list":              {Handler: SessionsListHandler},
		"sessions.create":            {Handler: SessionsCreateHandler},
		"sessions.ensure":            {Handler: SessionsEnsureHandler},
		"sessions.preview":           {Handler: SessionsPreviewHandler},
		"sessions.patch":             {Handler: SessionsPatchHandler},
		"sessions.reset":             {Handler: SessionsResetHandler},
		"sessions.delete":            {Handler: SessionsDeleteHandler},
		"sessions.compact":           {Handler: SessionsCompactHandler},
		"sessions.usage":             {Handler: SessionsUsageHandler},
		"sessions.usage.timeseries":  {Handler: SessionsUsageTimeseriesHandler},
		"sessions.usage.logs":        {Handler: SessionsUsageLogsHandler},
		"trace.list":                 {Handler: TraceListHandler},
		"trace.content":              {Handler: TraceContentHandler},
		"approvals.list":             {Handler: ApprovalsListHandler},
		"approvals.approve":          {Handler: ApprovalsApproveHandler, RequiredPermission: "ops:ack"},
		"approvals.deny":             {Handler: ApprovalsDenyHandler, RequiredPermission: "ops:ack"},
		"approvals.whitelistSession": {Handler: ApprovalsWhitelistSessionHandler, RequiredPermission: "ops:ack"},
		"last-heartbeat":             {Handler: LastHeartbeatHandler},
		"set-heartbeats":             {Handler: SetHeartbeatsHandler},
		"wake":                       {Handler: WakeHandler},
		"node.pair.request":          {Handler: NodePairRequestHandler},
		"node.pair.list":             {Handler: NodePairListHandler},
		"node.pair.approve":          {Handler: NodePairApproveHandler, RequiredPermission: "menu:config"},
		"node.pair.reject":           {Handler: NodePairRejectHandler, RequiredPermission: "menu:config"},
		"node.pair.verify":           {Handler: NodePairVerifyHandler},
		"device.pair.list":           {Handler: DevicePairListHandler},
		"device.pair.approve":        {Handler: DevicePairApproveHandler, RequiredPermission: "menu:config"},
		"device.pair.reject":         {Handler: DevicePairRejectHandler, RequiredPermission: "menu:config"},
		"device.token.rotate":        {Handler: DeviceTokenRotateHandler, RequiredPermission: "menu:config"},
		"device.token.revoke":        {Handler: DeviceTokenRevokeHandler, RequiredPermission: "menu:config"},
		"node.rename":                {Handler: NodeRenameHandler},
		"node.list":                  {Handler: NodeListHandler},
		"node.describe":              {Handler: NodeDescribeHandler},
		"node.invoke":                {Handler: NodeInvokeHandler},
		"node.invoke.result":         {Handler: NodeInvokeResultHandler},
		"node.event":                 {Handler: NodeEventHandler},
		"system-presence":            {Handler: SystemPresenceHandler},
		"system-event":               {Handler: SystemEventHandler},
		"send":                       {Handler: SendHandler},
		"agent":                      {Handler: AgentHandler},
		"agent.identity.get":         {Handler: AgentIdentityGetHandler},
		"agent.wait":                 {Handler: AgentWaitHandler},
		"browser.request":            {Handler: BrowserRequestHandler},
		"chat.history":               {Handler: ChatHistoryHandler},
		"chat.abort":                 {Handler: ChatAbortHandler},
		"chat.send":                  {Handler: ChatSendHandler},
		"chat.inject":                {Handler: ChatInjectHandler, RequiredPermission: "menu:config"},
		"swarm.workspaces.list":      {Handler: SwarmWorkspacesListHandler},
		"swarm.workspaces.create":    {Handler: SwarmWorkspacesCreateHandler, RequiredPermission: "menu:config"},
		"swarm.workspaces.delete":    {Handler: SwarmWorkspacesDeleteHandler, RequiredPermission: "menu:config"},
		"swarm.workspaces.abortAll":  {Handler: SwarmWorkspacesAbortAllHandler, RequiredPermission: "menu:config"},
		"swarm.members.list":         {Handler: SwarmMembersListHandler},
		"swarm.members.add":          {Handler: SwarmMembersAddHandler, RequiredPermission: "menu:config"},
		"swarm.members.remove":       {Handler: SwarmMembersRemoveHandler, RequiredPermission: "menu:config"},
		"swarm.message.send":         {Handler: SwarmMessageSendHandler, RequiredPermission: "menu:config"},
		"swarm.graph.get":            {Handler: SwarmGraphGetHandler},
		"swarm.history.get":          {Handler: SwarmHistoryGetHandler},
		"web.login.start":            {Handler: WebLoginStubHandler},
		"web.login.wait":             {Handler: WebLoginStubHandler},
	}
	if ctx != nil && ctx.CronService != nil {
		r["cron.list"] = MethodDescriptor{Handler: CronListHandler}
		r["cron.status"] = MethodDescriptor{Handler: CronStatusHandler}
		r["cron.add"] = MethodDescriptor{Handler: CronAddHandler}
		r["cron.remove"] = MethodDescriptor{Handler: CronRemoveHandler}
		r["cron.update"] = MethodDescriptor{Handler: CronUpdateHandler}
		r["cron.run"] = MethodDescriptor{Handler: CronRunHandler}
		r["cron.runs"] = MethodDescriptor{Handler: CronRunsHandler}
	}
	for method, perm := range rbac.MethodPermissions {
		if desc, ok := r[method]; ok {
			desc.RequiredPermission = perm
			r[method] = desc
		}
	}
	return r
}
