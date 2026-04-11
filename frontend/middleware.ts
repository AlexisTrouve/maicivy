import createMiddleware from 'next-intl/middleware';
import { locales, defaultLocale } from './i18n/config';

export default createMiddleware({
  locales,
  defaultLocale,
  localePrefix: 'always',
  localeDetection: false // Disable auto-detection, respect URL locale
});

export const config = {
  // Exclude api, _next, _vercel, static files, and special routes (api-test)
  matcher: ['/((?!api|_next|_vercel|api-test|.*\\..*).*)']
};
