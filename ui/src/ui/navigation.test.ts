import { describe, expect, it } from "vitest";
import {
  PRODUCT_TOUR_VERSION,
  getProductTourSteps,
  getTabGroups,
  iconForTab,
  inferBasePathFromPathname,
  markProductTourCompleted,
  normalizeBasePath,
  normalizePath,
  pathForTab,
  shouldShowProductTour,
  subtitleForTab,
  tabFromPath,
  titleForTab,
  type Tab,
} from "./navigation.ts";

/** All valid tab identifiers derived from getTabGroups() */
const ALL_TABS: Tab[] = getTabGroups().flatMap((group) => group.tabs) as Tab[];

describe("iconForTab", () => {
  it("returns a non-empty string for every tab", () => {
    for (const tab of ALL_TABS) {
      const icon = iconForTab(tab);
      expect(icon).toBeTruthy();
      expect(typeof icon).toBe("string");
      expect(icon.length).toBeGreaterThan(0);
    }
  });

  it("returns stable icons for known tabs", () => {
    expect(iconForTab("message")).toBe("messageSquare");
    expect(iconForTab("scheduledTasks")).toBe("alarmClock");
    expect(iconForTab("employeeMarket")).toBe("users");
    expect(iconForTab("skillLibrary")).toBe("zap");
    expect(iconForTab("toolLibrary")).toBe("wrench");
    expect(iconForTab("modelLibrary")).toBe("modelCube");
    expect(iconForTab("tutorials")).toBe("book");
    expect(iconForTab("community")).toBe("globe");
    expect(iconForTab("chat")).toBe("messageSquare");
    expect(iconForTab("overview")).toBe("overviewGrid");
    expect(iconForTab("channels")).toBe("link");
    expect(iconForTab("instances")).toBe("radio");
    expect(iconForTab("sessions")).toBe("scrollText");
    expect(iconForTab("cron")).toBe("loader");
    expect(iconForTab("cronHistory")).toBe("historyClock");
    expect(iconForTab("skills")).toBe("zap");
    expect(iconForTab("nodes")).toBe("monitor");
    expect(iconForTab("config")).toBe("settings");
    expect(iconForTab("debug")).toBe("bug");
    expect(iconForTab("usage")).toBe("usageBars");
    expect(iconForTab("models")).toBe("modelCube");
    expect(iconForTab("sandbox")).toBe("sandbox");
    expect(iconForTab("envVars")).toBe("envVars");
    expect(iconForTab("llmTrace")).toBe("traceBars");
    expect(iconForTab("logs")).toBe("scrollText");
  });

  it("returns active icons only for top-tab items", () => {
    expect(iconForTab("message", true)).toBe("messageSquareActive");
    expect(iconForTab("scheduledTasks", true)).toBe("alarmClockActive");
    expect(iconForTab("employeeMarket", true)).toBe("usersActive");
    expect(iconForTab("skillLibrary", true)).toBe("zapActive");
    expect(iconForTab("toolLibrary", true)).toBe("wrenchActive");
    expect(iconForTab("modelLibrary", true)).toBe("modelCubeActive");
    expect(iconForTab("tutorials", true)).toBe("bookActive");
    expect(iconForTab("community", true)).toBe("globeActive");
    expect(iconForTab("config", true)).toBe("settingsActive");
  });

  it("does not switch unrelated tabs to active-only icons", () => {
    expect(iconForTab("chat", true)).toBe("messageSquare");
    expect(iconForTab("skills", true)).toBe("zap");
    expect(iconForTab("cronHistory", true)).toBe("historyClock");
    expect(iconForTab("overview", true)).toBe("overviewGrid");
  });

  it("returns a fallback icon for unknown tab", () => {
    // TypeScript won't allow this normally, but runtime could receive unexpected values
    const unknownTab = "unknown" as Tab;
    expect(iconForTab(unknownTab)).toBe("folder");
  });
});

describe("titleForTab", () => {
  it("returns a non-empty string for every tab", () => {
    for (const tab of ALL_TABS) {
      const title = titleForTab(tab);
      expect(title).toBeTruthy();
      expect(typeof title).toBe("string");
    }
  });

  it("returns expected titles", () => {
    expect(titleForTab("chat")).toBe("Chat");
    expect(titleForTab("overview")).toBe("运维驾驶舱");
    expect(titleForTab("cron")).toBe("Cron Jobs");
  });
});

describe("subtitleForTab", () => {
  it("returns a string for every tab", () => {
    for (const tab of ALL_TABS) {
      const subtitle = subtitleForTab(tab);
      expect(typeof subtitle).toBe("string");
    }
  });

  it("returns descriptive subtitles", () => {
    expect(subtitleForTab("chat")).toContain("chat session");
    expect(subtitleForTab("config")).toContain("openclaw.json");
  });
});

describe("normalizeBasePath", () => {
  it("returns empty string for falsy input", () => {
    expect(normalizeBasePath("")).toBe("");
  });

  it("adds leading slash if missing", () => {
    expect(normalizeBasePath("ui")).toBe("/ui");
  });

  it("removes trailing slash", () => {
    expect(normalizeBasePath("/ui/")).toBe("/ui");
  });

  it("returns empty string for root path", () => {
    expect(normalizeBasePath("/")).toBe("");
  });

  it("handles nested paths", () => {
    expect(normalizeBasePath("/apps/openclaw")).toBe("/apps/openclaw");
  });
});

describe("normalizePath", () => {
  it("returns / for falsy input", () => {
    expect(normalizePath("")).toBe("/");
  });

  it("adds leading slash if missing", () => {
    expect(normalizePath("chat")).toBe("/chat");
  });

  it("removes trailing slash except for root", () => {
    expect(normalizePath("/chat/")).toBe("/chat");
    expect(normalizePath("/")).toBe("/");
  });
});

describe("pathForTab", () => {
  it("returns correct path without base", () => {
    expect(pathForTab("chat")).toBe("/chat");
    expect(pathForTab("overview")).toBe("/overview");
    expect(pathForTab("modelLibrary")).toBe("/model-library");
  });

  it("prepends base path", () => {
    expect(pathForTab("chat", "/ui")).toBe("/ui/chat");
    expect(pathForTab("sessions", "/apps/openclaw")).toBe("/apps/openclaw/sessions");
  });
});

describe("tabFromPath", () => {
  it("returns tab for valid path", () => {
    expect(tabFromPath("/chat")).toBe("chat");
    expect(tabFromPath("/overview")).toBe("overview");
    expect(tabFromPath("/sessions")).toBe("sessions");
    expect(tabFromPath("/model-library")).toBe("modelLibrary");
  });

  it("returns message for root path", () => {
    expect(tabFromPath("/")).toBe("message");
  });

  it("handles base paths", () => {
    expect(tabFromPath("/ui/chat", "/ui")).toBe("chat");
    expect(tabFromPath("/apps/openclaw/sessions", "/apps/openclaw")).toBe("sessions");
  });

  it("returns null for unknown path", () => {
    expect(tabFromPath("/unknown")).toBeNull();
  });

  it("is case-insensitive", () => {
    expect(tabFromPath("/CHAT")).toBe("chat");
    expect(tabFromPath("/Overview")).toBe("overview");
  });
});

describe("inferBasePathFromPathname", () => {
  it("returns empty string for root", () => {
    expect(inferBasePathFromPathname("/")).toBe("");
  });

  it("returns empty string for direct tab path", () => {
    expect(inferBasePathFromPathname("/chat")).toBe("");
    expect(inferBasePathFromPathname("/overview")).toBe("");
  });

  it("infers base path from nested paths", () => {
    expect(inferBasePathFromPathname("/ui/chat")).toBe("/ui");
    expect(inferBasePathFromPathname("/apps/openclaw/sessions")).toBe("/apps/openclaw");
  });

  it("handles index.html suffix", () => {
    expect(inferBasePathFromPathname("/index.html")).toBe("");
    expect(inferBasePathFromPathname("/ui/index.html")).toBe("/ui");
  });
});

describe("product tour", () => {
  it("defines six steps in onboarding order", () => {
    const tabs = getProductTourSteps().map((s) => s.tab);
    expect(tabs).toEqual([
      "modelLibrary",
      "message",
      "skillLibrary",
      "toolLibrary",
      "employeeMarket",
      "scheduledTasks",
    ]);
  });

  it("shows tour until completed for current version", () => {
    localStorage.clear();
    expect(shouldShowProductTour()).toBe(true);
    markProductTourCompleted();
    expect(shouldShowProductTour()).toBe(false);
    expect(shouldShowProductTour("other-version")).toBe(true);
    localStorage.clear();
  });

  it("uses a stable product tour version constant", () => {
    expect(PRODUCT_TOUR_VERSION).toMatch(/^v\d+\.\d+\.\d+/);
  });
});

describe("getTabGroups", () => {
  it("contains all expected groups", () => {
    const labels = getTabGroups().map((g) => g.label);
    expect(labels).toContain("Chat");
    expect(labels).toContain("Control");
    expect(labels).toContain("Agent");
    expect(labels).toContain("Settings");
  });

  it("all tabs are unique", () => {
    const allTabs = getTabGroups().flatMap((g) => g.tabs);
    const uniqueTabs = new Set(allTabs);
    expect(uniqueTabs.size).toBe(allTabs.length);
  });
});
