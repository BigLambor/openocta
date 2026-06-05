# OpenOcta Operations Platform Upgrade Walkthrough

This document outlines the changes, tests, and verification results for the navigation, sidebar, and business scenario upgrades implemented on the OpenOcta platform.

## Summary of Changes

All requirements from the task list in [docs/navigation-sidebar-scenario-task-list.md](file:///Users/isadmin/MagicSpace/openocta/docs/navigation-sidebar-scenario-task-list.md) have been successfully implemented and validated.

### 1. Navigation & Routing (P0)
- Fixed Cockpit domain card navigation: "打开运维域" now navigates directly to the new domain detail view instead of the assets page.
- Added dual buttons to Cockpit cards: "进入域详情" (major action) and "查看资产" (minor action, linking to `/assets?domain=<domainId>`).
- Implemented routing support for `/overview/domain/:domainId` and correctly synchronized opsDomain contexts.

### 2. Domain Detail / Domain Insight View (P1)
- Developed the new view component [domain-insight.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/domain-insight.ts).
- Integrated core stats (health score, cluster count, alert group count, latest inspection score).
- Designed business health scenario cards for the BCH domain (Flink health, Spark tuning, HDFS storage & metadata).
- Integrated latest alarms and recent inspection summaries with deep linking to relevant workbench modules.

### 3. Workbench Sidebar (P2) & Assets Sidebar (P3)
- Restructured [workbench.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/workbench.ts) and [assets-view.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/assets-view.ts) layouts.
- Built and integrated the professional context sidebar [ops-context-sidebar.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/components/ops-context-sidebar.ts) to clean up layout headers, move filters/navigation items to the left side, and preserve the global `opsDomain` filter context.

### 4. Scenario Relocation & Deprecation (P4)
- Moved old tech-domain pages into the unified Workbench and Domain Detail areas.
- Added a migration banner in [tech-ops-domain.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/tech-ops-domain.ts) informing users that capabilities are relocated, providing action buttons to transition into the new modules.

### 5. Unified AI Context Integration (P5)
- Standardized the AIOps context structure by implementing `buildUnifiedAiContext` in [chat.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/controllers/chat.ts). It maps domain, scenario (workflowType), capability, objectRef (objectId, objectType), and assetRef (cluster, service) into the standard gateway `chat.send` context.
- Added interactive "AI 分析" buttons onto the BCH scenario cards in the domain details view.
- Added "AI 分析" buttons onto clusters, services, components, and jobs lists inside the assets catalog to trigger context-aware conversational runs with digital employees.
- Removed unused/skeleton buttons from diagnosing/tuning views, displaying clear, realistic input/output contexts instead.

---

## Verification & Testing

### Automated Tests
Ran the vitest test suite under the `ui` directory:
```bash
npm run test
```
All 326 tests passed successfully (including new unit tests specifically covering `buildUnifiedAiContext` mapping logic).

### Manual Verification Path
1. **Domain Detail:** Navigate to a domain (e.g. BCH) detail page. Verify that the score cards, quick links, scenario cards (Flink, Spark, HDFS), alarm list, and recent inspection reports render.
2. **AI Action in Scenario Cards:** Click "AI 分析" on Flink/Spark/HDFS cards. Verify that you are redirected to the AI assistant with pre-populated questions and standard contextual properties.
3. **AI Action in Assets Catalog:** Navigate to "服务与资产". Change views (clusters, services, components, jobs) and click "AI 分析" on any item. Verify that it starts a dialog session with the correct asset context.
