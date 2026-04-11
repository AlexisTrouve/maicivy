export const locales = ['fr', 'en', 'de', 'it', 'zh'] as const;
export const defaultLocale = 'fr' as const;
export type Locale = (typeof locales)[number];
