import { LitElement, html, svg, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { icons } from "../icons.ts";
import { PROVINCES, DISPUTED, RIVERS_TRANSFORM, RIVERS, BORDERS } from "./china-map-data.ts";

type CityNode = {
  id: string;
  name: string;
  label: string;
  x: number;
  y: number;
};

const ALL_CITY_COORDS: Record<string, Omit<CityNode, "id">> = {
  beijing: { name: "北京数据中心", label: "北京", x: 736.9, y: 335.3 },
  shanghai: { name: "上海公有云", label: "上海", x: 861.0, y: 515.1 },
  shenzhen: { name: "深圳灾备中心", label: "深圳", x: 745.0, y: 730.0 },
  chengdu: { name: "成都边缘节点", label: "成都", x: 542.5, y: 560.5 },
  wuhan: { name: "武汉研发中心", label: "武汉", x: 724.4, y: 547.3 },
  xian: { name: "西安控制节点", label: "西安", x: 620.3, y: 479.3 },
  harbin: { name: "哈尔滨数据中心", label: "哈尔滨", x: 888.7, y: 177.3 },
  hohhot: { name: "呼和浩特数据中心", label: "呼和浩特", x: 665.1, y: 319.2 },
  hangzhou: { name: "杭州数据中心", label: "杭州", x: 834.4, y: 538.4 },
  guangzhou: { name: "广州数据中心", label: "广州", x: 730.9, y: 717.5 },
  chongqing: { name: "重庆数据中心", label: "重庆", x: 582.4, y: 584.3 },
  lanzhou: { name: "兰州数据中心", label: "兰州", x: 529.5, y: 440.8 },
};

const MASTER_LINKS = [
  { from: "beijing", to: "xian" },
  { from: "beijing", to: "shanghai" },
  { from: "shanghai", to: "wuhan" },
  { from: "shanghai", to: "shenzhen" },
  { from: "shenzhen", to: "wuhan" },
  { from: "chengdu", to: "xian" },
  { from: "chengdu", to: "wuhan" },
  { from: "wuhan", to: "xian" },
  { from: "harbin", to: "beijing" },
  { from: "hohhot", to: "beijing" },
  { from: "hohhot", to: "xian" },
  { from: "hangzhou", to: "shanghai" },
  { from: "hangzhou", to: "shenzhen" },
  { from: "guangzhou", to: "shenzhen" },
  { from: "chongqing", to: "chengdu" },
  { from: "chongqing", to: "wuhan" },
  { from: "lanzhou", to: "xian" },
  { from: "lanzhou", to: "chengdu" },
];

const DOMAIN_NAMES: Record<string, string> = {
  hadoop: "BCH生态",
  fi: "FI商业生态",
  gbase: "GBase数据库",
  governance: "开发治理平台",
  dataapps: "数据App运维",
};

// No baseline data. Data is fetched exclusively from cluster assets.

function getCityIdByRegion(region: string): string | null {
  if (!region) return null;
  const r = region.toLowerCase();
  if (r.includes("哈尔滨") || r.includes("哈池") || r.includes("harbin")) return "harbin";
  if (r.includes("呼和浩特") || r.includes("呼池") || r.includes("hohhot")) return "hohhot";
  if (r.includes("北京") || r.includes("beijing") || r.includes("华北")) return "beijing";
  if (r.includes("上海") || r.includes("shanghai") || r.includes("华东")) return "shanghai";
  if (r.includes("深圳") || r.includes("shenzhen")) return "shenzhen";
  if (r.includes("广州") || r.includes("guangzhou")) return "guangzhou";
  if (r.includes("华南")) return "shenzhen";
  if (r.includes("杭州") || r.includes("hangzhou")) return "hangzhou";
  if (r.includes("成都") || r.includes("chengdu")) return "chengdu";
  if (r.includes("重庆") || r.includes("chongqing")) return "chongqing";
  if (r.includes("西南")) return "chengdu";
  if (r.includes("武汉") || r.includes("wuhan") || r.includes("华中")) return "wuhan";
  if (r.includes("西安") || r.includes("xian") || r.includes("西北")) return "xian";
  if (r.includes("兰州") || r.includes("lanzhou")) return "lanzhou";
  return null;
}

@customElement("ops-map")
export class OpsMap extends LitElement {
  @property({ type: Array }) clusters: OpsClusterRecord[] = [];
  @property({ type: Array }) snapshots: any[] = [];
  @property({ type: Object }) onNavigateDomain?: (domain: string) => void;

  @state() private hoveredCityId: string | null = null;

  createRenderRoot() {
    return this; // Render in light DOM to utilize global CSS in ops-dashboard.css
  }

  private getCitiesData() {
    const result: Record<string, Array<{ domain: string; score: number; status: string; alerts: number }>> = {};
    
    // Initialize empty lists for all known cities
    for (const cityId of Object.keys(ALL_CITY_COORDS)) {
      result[cityId] = [];
    }

    // Process real clusters if present
    if (this.clusters && this.clusters.length > 0) {
      const clustersByCity: Record<string, OpsClusterRecord[]> = {};
      for (const cluster of this.clusters) {
        const cityId = getCityIdByRegion(cluster.region || "");
        if (cityId) {
          if (!clustersByCity[cityId]) {
            clustersByCity[cityId] = [];
          }
          clustersByCity[cityId].push(cluster);
        }
      }

      // Override baseline with real data for cities that have clusters
      for (const cityId of Object.keys(clustersByCity)) {
        const cityClusters = clustersByCity[cityId];
        const clustersByDomain: Record<string, OpsClusterRecord[]> = {};
        for (const c of cityClusters) {
          if (!clustersByDomain[c.domain]) {
            clustersByDomain[c.domain] = [];
          }
          clustersByDomain[c.domain].push(c);
        }

        const domainList: Array<{ domain: string; score: number; status: string; alerts: number }> = [];
        for (const domainKey of Object.keys(clustersByDomain)) {
          const domainClusters = clustersByDomain[domainKey];
          let worstStatus = "healthy";
          let alertsCount = 0;
          let totalScore = 0;
          let countWithScore = 0;

          for (const c of domainClusters) {
            // Find L3 Health Snapshot for this cluster
            const snap = (this.snapshots || []).find(s => s.clusterId === c.id);
            
            let status = c.status;
            let score = c.status === "healthy" ? 95 : c.status === "warning" ? 80 : c.status === "critical" ? 50 : 75;
            
            if (snap) {
              // Map snapshot status
              if (snap.scoreStatus === "ok") {
                status = "healthy";
              } else if (snap.scoreStatus === "warning" || snap.scoreStatus === "partial") {
                status = "warning";
              } else if (snap.scoreStatus === "critical" || snap.scoreStatus === "degraded") {
                status = "critical";
              } else {
                status = "unknown";
              }
              
              if (snap.score != null) {
                score = snap.score;
              } else {
                score = status === "healthy" ? 95 : status === "warning" ? 80 : status === "critical" ? 50 : 75;
              }
            }

            if (status === "critical") {
              worstStatus = "critical";
              alertsCount++;
            } else if (status === "warning") {
              if (worstStatus !== "critical") worstStatus = "warning";
              alertsCount++;
            } else if (status === "unknown" || status === "inactive") {
              if (worstStatus !== "critical" && worstStatus !== "warning") worstStatus = "unknown";
            }
            
            totalScore += score;
            countWithScore++;
          }

          const avgScore = countWithScore > 0 ? Math.round(totalScore / countWithScore) : 90;

          domainList.push({
            domain: domainKey,
            score: avgScore,
            status: worstStatus,
            alerts: alertsCount
          });
        }

        result[cityId] = domainList;
      }
    }

    return result;
  }

  private handleNodeMouseEnter(cityId: string) {
    this.hoveredCityId = cityId;
  }

  private handleNodeMouseLeave() {
    this.hoveredCityId = null;
  }

  render() {
    const citiesData = this.getCitiesData();
    
    // Resolve active cities from clusters
    const activeCities = Object.keys(ALL_CITY_COORDS)
      .filter(id => citiesData[id] && citiesData[id].length > 0)
      .map(id => ({
        id,
        ...ALL_CITY_COORDS[id]
      }));

    const hoveredCity = activeCities.find(c => c.id === this.hoveredCityId);
    const hoveredCityData = hoveredCity ? citiesData[hoveredCity.id] : null;

    // Build active connections
    const activeLinks: Array<{ from: string; to: string }> = [];
    const connectedCities = new Set<string>();

    for (const link of MASTER_LINKS) {
      const fromActive = activeCities.some(c => c.id === link.from);
      const toActive = activeCities.some(c => c.id === link.to);
      if (fromActive && toActive) {
        activeLinks.push(link);
        connectedCities.add(link.from);
        connectedCities.add(link.to);
      }
    }

    // Connect isolated active cities to their nearest active neighbor
    if (activeCities.length > 1) {
      for (const city of activeCities) {
        if (!connectedCities.has(city.id)) {
          let minDistance = Infinity;
          let closestCityId: string | null = null;
          for (const other of activeCities) {
            if (other.id === city.id) continue;
            const dist = Math.sqrt(Math.pow(city.x - other.x, 2) + Math.pow(city.y - other.y, 2));
            if (dist < minDistance) {
              minDistance = dist;
              closestCityId = other.id;
            }
          }
          if (closestCityId) {
            const linkExists = activeLinks.some(l => 
              (l.from === city.id && l.to === closestCityId) || 
              (l.from === closestCityId && l.to === city.id)
            );
            if (!linkExists) {
              activeLinks.push({ from: city.id, to: closestCityId });
            }
            connectedCities.add(city.id);
            connectedCities.add(closestCityId);
          }
        }
      }
    }

    return html`
      <div class="ops-map-container">
        <svg class="ops-map-svg" viewBox="0 0 1000 850" xmlns="http://www.w3.org/2000/svg">
          <!-- Background tech grid pattern -->
          <defs>
            <pattern id="map-grid" width="20" height="20" patternUnits="userSpaceOnUse">
              <path d="M 20 0 L 0 0 0 20" fill="none" stroke="rgba(148, 163, 184, 0.12)" stroke-width="1"></path>
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#map-grid)" class="ops-map__grid-pattern"></rect>

          <!-- Stylized China boundaries (Detailed Province Mesh) -->
          <g class="province">
            ${PROVINCES.map(p => svg`<path id="${p.id}" d="${p.d}"></path>`)}
          </g>

          <g class="disputed">
            ${DISPUTED.map(p => svg`<path id="${p.id}" d="${p.d}" style="${p.style || ''}"></path>`)}
          </g>

          <g transform="${RIVERS_TRANSFORM}">
            ${RIVERS.map(p => svg`<path id="${p.id}" d="${p.d}"></path>`)}
          </g>

          <g class="borders">
            ${BORDERS.map(p => svg`<path id="${p.id}" d="${p.d}" style="${p.style || ''}"></path>`)}
          </g>

          <!-- Glowing Network Trunk Links -->
          <g class="ops-map__links">
            ${activeLinks.map(link => {
              const fromCity = activeCities.find(c => c.id === link.from);
              const toCity = activeCities.find(c => c.id === link.to);
              if (!fromCity || !toCity) return nothing;
              return svg`
                <line 
                  x1="${fromCity.x}" 
                  y1="${fromCity.y}" 
                  x2="${toCity.x}" 
                  y2="${toCity.y}" 
                  class="ops-map__link"
                ></line>
              `;
            })}
          </g>

          <!-- Cities Nodes & Orbiting Satellite Business Dots -->
          <g class="ops-map__nodes">
            ${activeCities.map(city => {
              const bList = citiesData[city.id] || [];
              const numSats = bList.length;
              const radius = 22; // Orbit radius
              
              // Calculate overall status color of the city center node (unknown/grey if no clusters)
              let cityStatus = "unknown";
              if (bList.length > 0) {
                cityStatus = "healthy";
                if (bList.some(b => b.status === "critical")) {
                  cityStatus = "critical";
                } else if (bList.some(b => b.status === "warning")) {
                  cityStatus = "warning";
                } else if (bList.some(b => b.status === "unknown")) {
                  cityStatus = "unknown";
                }
              }

              return svg`
                <!-- Group for mouse interactions -->
                <g 
                  class="ops-map__node-group" 
                  @mouseenter=${() => this.handleNodeMouseEnter(city.id)}
                  @mouseleave=${() => this.handleNodeMouseLeave()}
                >
                  <!-- Invisible large catch-circle to prevent hover flickering -->
                  <circle 
                    cx="${city.x}" 
                    cy="${city.y}" 
                    r="32" 
                    fill="transparent" 
                    style="cursor: pointer;"
                  ></circle>

                  <!-- Outer halo wave -->
                  <circle 
                    cx="${city.x}" 
                    cy="${city.y}" 
                    class="ops-map__node-halo ops-map__node-halo--${cityStatus}" 
                  ></circle>
                  <!-- Middle ring for structural definition -->
                  <circle 
                    cx="${city.x}" 
                    cy="${city.y}" 
                    r="7" 
                    class="ops-map__node-middle ops-map__node-middle--${cityStatus}" 
                  ></circle>
                  <!-- Inner core circle -->
                  <circle 
                    cx="${city.x}" 
                    cy="${city.y}" 
                    r="3.5" 
                    class="ops-map__node-center ops-map__node-center--${cityStatus}" 
                  ></circle>
                  
                  <!-- Satellite business indicator dots -->
                  ${bList.map((b, idx) => {
                    const angle = (2 * Math.PI * idx) / numSats - Math.PI / 2;
                    const satX = city.x + radius * Math.cos(angle);
                    const satY = city.y + radius * Math.sin(angle);
                    return svg`
                      <g class="ops-map__sat-dot ops-map__sat-dot--${b.status}">
                        <circle cx="${satX}" cy="${satY}" r="4" class="ops-map__sat-core"></circle>
                        <circle cx="${satX}" cy="${satY}" r="4" class="ops-map__sat-ring"></circle>
                      </g>
                    `;
                  })}

                  <!-- Location Label -->
                  <text 
                    x="${city.x}" 
                    y="${city.y + 36}" 
                    class="ops-map__node-label"
                  >${city.label}</text>
                </g>
              `;
            })}
          </g>
        </svg>

        <!-- Floating Glassmorphic Tooltip positioned absolute by percentage coords -->
        ${hoveredCity && hoveredCityData ? html`
          <div 
            class="ops-map__tooltip ops-map__tooltip--visible" 
            style="left: ${(hoveredCity.x / 1000) * 100}%; top: ${(hoveredCity.y / 850) * 100 - 4}%; transform: translate(-50%, -100%);"
          >
            <div class="ops-map__tooltip-header">
              <div class="ops-map__tooltip-title">
                <span class="ops-map__tooltip-title-icon">${icons.building}</span>
                <span>${hoveredCity.name}</span>
              </div>
              <span class="ops-map__tooltip-subtitle">
                ${hoveredCityData.length} 业务运行
              </span>
            </div>
            
            <div class="ops-map__tooltip-list">
              ${hoveredCityData.length === 0 ? html`
                <div class="ops-map__tooltip-no-clusters">暂无部署业务</div>
              ` : hoveredCityData.map(b => html`
                <div 
                  class="ops-map__tooltip-item" 
                  style="${this.onNavigateDomain ? 'cursor: pointer;' : ''}"
                  @click=${() => this.onNavigateDomain?.(b.domain)}
                >
                  <div class="ops-map__tooltip-item-left">
                    <span class="ops-map__tooltip-status-dot ops-map__tooltip-status-dot--${b.status}"></span>
                    <span class="ops-map__tooltip-item-name">${DOMAIN_NAMES[b.domain] || b.domain}</span>
                  </div>
                  <div class="ops-map__tooltip-item-right">
                    <span class="ops-map__tooltip-score ops-map__tooltip-score--${b.score >= 90 ? 'ok' : b.score >= 75 ? 'warning' : 'danger'}">
                      ${b.score}分
                    </span>
                    ${b.alerts > 0 ? html`
                      <span class="ops-map__tooltip-alerts" title="活动告警数">${b.alerts}告警</span>
                    ` : nothing}
                  </div>
                </div>
              `)}
            </div>
          </div>
        ` : nothing}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "ops-map": OpsMap;
  }
}
