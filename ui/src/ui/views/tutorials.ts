import { html, nothing } from "lit";
import type { EduCategory, EduCourse, EduLesson } from "../controllers/remote-market.ts";
import { icons } from "../icons.js";
import { getTutorialIcon } from "../tutorial-icons.ts";

export type TutorialsProps = {
  loading: boolean;
  error: string | null;
  categories: EduCategory[];
  query: string;
  selectedCategoryId: number | null;
  playingLink: string | null;
  onQueryChange: (next: string) => void;
  onSelectCategory: (id: number) => void;
  onLessonClick: (link: string) => void;
  onPlayingClose: () => void;
  onRefresh: () => void;
};

type SelectedTutorialLesson = {
  category: EduCategory;
  course: EduCourse;
  lesson: EduLesson;
};

function normalizeQuery(raw: string) {
  return (raw ?? "").trim().toLowerCase();
}

function includesQ(text: string, q: string) {
  if (!q) return true;
  return (text ?? "").toLowerCase().includes(q);
}

function tutorialAccentClass(accent?: string): string {
  const key = (accent ?? "").trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "-");
  return key ? `tutorials-category-${key}` : "";
}

/**
 * 从 B 站链接解析出内嵌播放器 URL。
 * 支持 bilibili.com/video/BVxxx、bilibili.com/video/avxxx 等格式。
 */
export function toBilibiliEmbedUrl(link: string): string | null {
  const url = (link ?? "").trim();
  if (!url) return null;
  try {
    const u = new URL(url);
    const host = u.hostname.toLowerCase();
    if (!host.includes("bilibili.com") && !host.includes("b23.tv")) return null;
    const path = u.pathname;
    const bvidMatch = path.match(/\/video\/(BV[0-9A-Za-z]+)/i);
    if (bvidMatch) {
      return `https://player.bilibili.com/player.html?bvid=${bvidMatch[1]}&high_quality=1`;
    }
    const avMatch = path.match(/\/video\/av(\d+)/i);
    if (avMatch) {
      return `https://player.bilibili.com/player.html?aid=${avMatch[1]}&high_quality=1`;
    }
    return null;
  } catch {
    return null;
  }
}

function findSelectedTutorialLesson(
  categories: EduCategory[],
  playingLink: string | null,
): SelectedTutorialLesson | null {
  const target = (playingLink ?? "").trim();
  if (!target) return null;

  for (const category of categories ?? []) {
    for (const course of category.courses ?? []) {
      const courseLink = (course.link ?? "").trim();
      const courseType = (course.course_type ?? "").trim().toLowerCase();
      if (courseType === "standalone" && courseLink === target) {
        return {
          category,
          course,
          lesson: {
            id: course.id,
            title: course.title,
            duration: course.duration,
            link: course.link,
            sort_order: course.sort_order,
            created_at: course.created_at,
            updated_at: course.updated_at,
          },
        };
      }
      for (const lesson of course.lessons ?? []) {
        if ((lesson.link ?? "").trim() === target) {
          return { category, course, lesson };
        }
      }
    }
  }

  return null;
}

const outcomes = [
  {
    role: "SRE工程师",
    name: "邓雪",
    avatar: "/img/avatar/dengxue.png",
    desc: "用上ApexOps后，服务器巡检、告警排查全交给智能体，深夜再也不用爬起来处理故障，值班轻松了不止一倍",
  },
  {
    role: "DBA工程师",
    name: "许知意",
    avatar: "/img/avatar/xuzhiyi.png",
    desc: "数据库巡检、慢SQL分析、备份校验都能自动执行，高危操作还有审批拦截，工作更稳更省心，效率直接拉满",
  },
  {
    role: "测试工程师",
    name: "杨雅琪",
    avatar: "/img/avatar/yangyaqi.png",
    desc: "自动化测试脚本批量生成、环境一键巡检、测试数据自动准备，迭代更快，加班次数少了一大半",
  },
  {
    role: "安全工程师",
    name: "孙梦佳",
    avatar: "/img/avatar/sunmengjia.png",
    desc: "安全基线检查、漏洞扫描、日志审计智能体自动跑，异常行为实时告警，安全合规省心又可控",
  },
] as const;

export function renderTutorials(props: TutorialsProps) {
  const orderedCategories = [...(props.categories ?? [])].sort(
    (a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.name.localeCompare(b.name, "zh-Hans-CN"),
  );

  const activeCategoryId =
    props.selectedCategoryId && orderedCategories.some((c) => c.id === props.selectedCategoryId)
      ? props.selectedCategoryId
      : orderedCategories[0]?.id ?? null;
  const activeCategory = activeCategoryId ? orderedCategories.find((c) => c.id === activeCategoryId) ?? null : null;

  const q = normalizeQuery(props.query);
  const courses = (activeCategory?.courses ?? [])
    .slice()
    .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.title.localeCompare(b.title, "zh-Hans-CN"))
    .filter((course) => {
      if (includesQ(course.title ?? "", q)) return true;
      return (course.lessons ?? []).some((l) => includesQ(l.title ?? "", q));
    });

  const embedUrl = props.playingLink ? toBilibiliEmbedUrl(props.playingLink) : null;
  const selectedLesson = findSelectedTutorialLesson(orderedCategories, props.playingLink);
  const toolbarActions = html`
    <div class="emp-toolbar__actions">
      <div class="emp-search">
        <span class="input"><input
          class="emp-search__input"
          type="text"
          placeholder="搜索课程/课时"
          .value=${props.query}
          ?disabled=${props.loading}
          @input=${(e: Event) => props.onQueryChange((e.target as HTMLInputElement).value)}
        /></span>
        <span class="emp-search__icon" aria-hidden="true">${icons.search}</span>
      </div>
      <button class="btn" @click=${props.onRefresh} ?disabled=${props.loading}>刷新</button>
    </div>
  `;

  return html`
    <main class="tutorials-page">
      <div class="tutorials-board">
        <div class="tutorials-board__header">
          <div class="tutorials-board__title-wrap">
            <h2 class="tutorials-board__title">ApexOps 教程</h2>
          </div>
          ${toolbarActions}
        </div>
        <div class="tutorials-board__body">
          <aside class="tutorials-categories">
            ${orderedCategories.length === 0
              ? html`<button class="tutorials-category" disabled>暂无分类</button>`
              : orderedCategories.map((category) => {
                  const active = activeCategoryId === category.id;
                  const iconSvg = getTutorialIcon(category.icon_class);
                  const iconText = (category.name ?? "").trim().slice(0, 1) || "教";
                  const accentClass = tutorialAccentClass(category.accent);
                  return html`
                    <button
                      class="tutorials-category ${accentClass} ${active ? "active" : ""}"
                      type="button"
                      ?disabled=${props.loading}
                      @click=${() => props.onSelectCategory(category.id)}
                    >
                      <span class="tutorials-category__icon" aria-hidden="true">
                        ${iconSvg ? html`<span class="tutorials-category__icon-svg">${iconSvg}</span>` : iconText}
                      </span>
                      <span class="tutorials-category__label">${category.name}</span>
                    </button>
                  `;
                })}
          </aside>

          <div class="tutorials-content">
            ${props.error ? html`<div class="callout danger" style="margin-bottom: 16px;">${props.error}</div>` : nothing}

            ${
              !activeCategory
                ? html`<div class="emp-empty">暂无分类数据，请点击“刷新”。</div>`
                : courses.length
                  ? html`
                      <div class="tutorials-card">
                        ${courses.map((course) => {
                  const isStandalone = (course.course_type ?? "").trim().toLowerCase() === "standalone";
                  const sortedLessons = (course.lessons ?? [])
                    .slice()
                    .sort(
                      (a, b) =>
                        (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.title.localeCompare(b.title, "zh-Hans-CN"),
                    );

                  const lessons = sortedLessons
                    .map((lesson, index) => ({
                      ...lesson,
                      sequence: index + 1,
                    }))
                    .filter((l) => includesQ(l.title ?? "", q) || includesQ(course.title ?? "", q));

                  const effectiveLessons: Array<{
                    id: number;
                    title?: string;
                    duration?: string;
                    link?: string;
                    sequence: number;
                  }> =
                    isStandalone && lessons.length === 0
                      ? [
                          {
                            id: course.id,
                            title: course.title,
                            duration: course.duration,
                            link: course.link,
                            sequence: 1,
                          },
                        ]
                      : lessons;

                  const openByDefault = true;

                  return html`
                      <details ?open=${openByDefault} class="tutorials-course">
                        <summary class="tutorials-course__summary">
                          <span class="tutorials-course__title-row">
                            <span class="tutorials-course__caret" aria-hidden="true"></span>
                            <span class="tutorials-course__title">${course.title}</span>
                          </span>
                        </summary>

                      <div class="tutorials-lessons">
                        ${effectiveLessons.map((lesson) => {
                          const indexText = String(lesson.sequence).padStart(2, "0");
                          const hasLink = !!(lesson.link ?? "").trim();
                          const handleClick = () => {
                            if (hasLink) props.onLessonClick(lesson.link!);
                          };
                          return html`
                            <div
                              class="tutorials-lesson ${hasLink ? "tutorials-lesson--clickable" : "tutorials-lesson--disabled"}"
                              @click=${handleClick}
                              role=${hasLink ? "button" : "none"}
                              tabindex=${hasLink ? 0 : nothing}
                              @keydown=${(e: KeyboardEvent) => {
                                if (hasLink && (e.key === "Enter" || e.key === " ")) {
                                  e.preventDefault();
                                  props.onLessonClick(lesson.link!);
                                }
                              }}
                            >
                              <span class="tutorials-lesson__index">${indexText}</span>
                              <span class="tutorials-lesson__title">${lesson.title}</span>
                              ${(lesson.duration ?? "").trim()
                                ? html`<span class="tutorials-lesson__duration">${lesson.duration}</span>`
                                : nothing}
                            </div>
                          `;
                        })}
                      </div>
                    </details>
                  `;
                })}
                      </div>
                    `
                  : html`<div class="emp-empty">没有匹配的课程/课时</div>`
            }
          </div>
        </div>
      </div>

      ${props.playingLink
        ? html`
            <div class="modal-overlay" @click=${props.onPlayingClose} role="dialog" aria-modal="true" aria-labelledby="tutorial-detail-title">
              <div class="modal card emp-detail-modal emp-detail-modal--large tutorials-detail-modal" @click=${(e: Event) => e.stopPropagation()}>
                <div class="emp-detail-modal__header tutorials-detail-modal__header">
                  <div class="emp-detail-header tutorials-detail-modal__header-main">
                    <h1 id="tutorial-detail-title" class="emp-detail-title tutorials-detail-modal__title">
                      ${selectedLesson?.lesson.title ?? "教程详情"}
                    </h1>
                  </div>
                  <div class="tutorials-detail-modal__actions">
                    <a
                      class="tutorials-detail-modal__link"
                      href=${props.playingLink}
                      target="_blank"
                      rel="noopener noreferrer"
                      @click=${(e: Event) => e.stopPropagation()}
                    >
                      在哔哩哔哩打开
                    </a>
                    <button
                      class="emp-detail-modal__close"
                      type="button"
                      aria-label="关闭"
                      @click=${props.onPlayingClose}
                    >
                      ${icons.x}
                    </button>
                  </div>
                </div>
                <div class="emp-detail-modal__body tutorials-detail-modal__body">
                  ${embedUrl
                    ? html`
                        <div class="tutorials-video-context tutorials-detail-video-context">
                          <div class="tutorials-video-wrap tutorials-detail-video-wrap">
                            <iframe
                              src=${embedUrl}
                              scrolling="no"
                              border="0"
                              frameborder="no"
                              framespacing="0"
                              allowfullscreen="true"
                              title=${selectedLesson?.lesson.title ?? "B站视频播放"}
                            ></iframe>
                          </div>
                        </div>
                      `
                    : html`
                        <div class="callout info">该教程暂不支持内嵌播放，请点击“打开原链接”查看详情。</div>
                      `}
                </div>
              </div>
            </div>
          `
        : nothing}
    </main>
  `;
}
