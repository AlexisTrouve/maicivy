import createMiddleware from 'next-intl/middleware';
import { locales, defaultLocale } from './i18n/config';

export default createMiddleware({
  locales,
  defaultLocale,
  localePrefix: 'as-needed'
});

export const config = {
  // Exclude api, _next, _vercel, static files, and special routes (3d-demo, api-test)
  matcher: ['/((?!api|_next|_vercel|3d-demo|api-test|.*\\..*).*)']
};
