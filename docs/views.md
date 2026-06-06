# Implementation Plan - Workbench Scenario Refactoring

This plan addresses the remaining/in-progress tasks in [workbench-scenario-refactor-task-list.md](file:///Users/isadmin/MagicSpace/openocta/docs/workbench-scenario-refactor-task-list.md) across technical domains, object/job-level contexts, entry/navigation rules, and permission checks.

## User Review Required

> [!IMPORTANT]
> - **Job-level & Directory Context mapping**: Flink/Spark jobs and HDFS directories will be integrated into the global "Object Scope" (对象范围) dropdown context in the workbench tab. We will load these objects dynamically from the app state as needed.
> - **Global Assistant naming**: The AI assistant panel will display "全域值班数字员工" (All-Domain Duty Employee) when the active domain is "all", preventing visual confusion of showing the "BCH" assistant by default.
> - **Permission Check in Scenario Directory**: In "All Domains" view, scenarios will be filtered by the user's technical domain permissions, so they only see scenarios for domains they have access to.

## Open Questions

None at this stage. The requirements are clear and align perfectly with the refactoring roadmap.

## Proposed Changes

### Component 1: Context and Global State

#### [MODIFY] [app.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app.ts)
- Add state variables `opsFlinkJobs` and `opsSparkJobs` (along with their loading indicators).
- Add methods `loadOpsFlinkJobs` and `loadOpsSparkJobs` to fetch Flink/Spark jobs list from the backend BCH client.

#### [MODIFY] [app-render.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app-render.ts)
- Dynamically trigger `loadOpsFlinkJobs` or `loadOpsSparkJobs` when the user opens Flink/Spark scenarios in the workbench if they are not already loaded.
- Pass `opsFlinkJobs` and `opsSparkJobs` down to the `renderWorkbench` container.
- Document workbench entry navigation rules and deep link handling with clear code comments.

### Component 2: Object Scope Context

#### [MODIFY] [workbench-context.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/ops/workbench-context.ts)
- Expand `objectOptionsForScenario` to return:
  - Flink jobs list for `bch-flink-health` scenario.
  - Spark jobs list for `bch-spark-tuning` scenario.
  - Common HDFS directory paths (e.g. `/tmp`, `/user`, `/app`) prefixed with namespace for `bch-hdfs-capacity` scenario.

### Component 3: Component Integration

#### [MODIFY] [scenario-components.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/ops/scenario-components.ts)
- Map `props.objectScope` to the `bch-flink-diagnosis`, `bch-spark-governance`, and `bch-fsimage-dashboard` components correctly.
- For HDFS directory scope (e.g. `directory:NS1:/tmp`), extract the namespace `NS1` and set it as `.activeNamespace` on the dashboard, passing down the full `.objectScope`.

#### [MODIFY] [bch-flink-diagnosis.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/ops/bch-flink-diagnosis.ts)
- Define `objectScope` property and import `parseWorkbenchObjectScope` from `workbench-context.ts`.
- Filter Flink jobs list by `objectScope` (supporting cluster filter and individual job name filter).

#### [MODIFY] [bch-spark-governance.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/ops/bch-spark-governance.ts)
- Define `objectScope` property and import `parseWorkbenchObjectScope` from `workbench-context.ts`.
- Filter Spark jobs list by `objectScope` (supporting cluster filter and individual job name filter).

#### [MODIFY] [bch-fsimage-dashboard.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/ops/bch-fsimage-dashboard.ts)
- Add `objectScope` property.
- Display a visually appealing banner displaying the active HDFS directory context if `objectScope` points to a directory path.

### Component 4: Layout and Directory Optimization

#### [MODIFY] [domain-filter.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/components/domain-filter.ts)
- Update `opsAssistantForDomain` to return a "全域值班数字员工" profile when the domain is `"all"`, using the BCH duty employee as the backend fallback.

#### [MODIFY] [workbench.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/workbench.ts)
- Import `canAccessOpsDomain` from `domain-filter.ts`.
- In `renderScenarioDirectory`, filter the list of scenarios to display in the All Domains view using `canAccessOpsDomain(props.user, scenario.domain)`.

### Component 5: Documentation

#### [NEW] [workbench-entry-context-rules.md](file:///Users/isadmin/MagicSpace/openocta/docs/workbench-entry-context-rules.md)
- Product rules document detailing context inheritance, entrance paths, parameters carried by each entry point, and technical design notes.

#### [NEW] [workbench-hierarchy-model.md](file:///Users/isadmin/MagicSpace/openocta/docs/workbench-hierarchy-model.md)
- Architecture document defining the workbench's four-layer structural hierarchy: Technical Domain Context -> Operation Object -> Scenario/Specialized View -> Workflow Task.

---

## Verification Plan

### Automated Tests
- Run `npm test` inside `ui` directory to ensure that all 356 unit/browser tests remain green.

### Manual Verification
- Verify the workbench layout under narrow and desktop viewports (avoid overlaps).
- Verify navigation from domain card / detail page sets correct domain context.
- Verify context dropdowns for object scopes correctly show Flink/Spark jobs and HDFS directory lists when selected.
