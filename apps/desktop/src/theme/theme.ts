export const themeIds = ["default-dark", "default-light"] as const;
export type ThemeId = (typeof themeIds)[number];

export const themeOptions = [
  { id: "default-dark", name: "默认暗色" },
  { id: "default-light", name: "默认亮色" },
] satisfies { id: ThemeId; name: string }[];

export function isThemeId(value: string | null): value is ThemeId {
  return value !== null && themeIds.some((id) => id === value);
}

export function applyTheme(themeId: ThemeId) {
  document.documentElement.dataset.theme = themeId;
}
