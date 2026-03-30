import { Metadata } from 'next';
import { BlogList } from '@/components/blog';

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }> | { locale: string };
}): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale || 'fr';

  const titles = {
    fr: 'Blog - Actualités Développement',
    en: 'Blog - Development News',
  };

  const descriptions = {
    fr: 'Articles générés automatiquement depuis mes commits. Suivez mon activité de développement en temps réel.',
    en: 'Articles automatically generated from my commits. Follow my development activity in real-time.',
  };

  return {
    title: titles[locale as keyof typeof titles] || titles.fr,
    description: descriptions[locale as keyof typeof descriptions] || descriptions.fr,
    openGraph: {
      title: titles[locale as keyof typeof titles] || titles.fr,
      description: descriptions[locale as keyof typeof descriptions] || descriptions.fr,
      type: 'website',
      images: ['https://maiprofiles.etheryale.com/images/img_dbb0624c'],
    },
  };
}

export default async function BlogPage({
  params,
}: {
  params: Promise<{ locale: string }> | { locale: string };
}) {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale || 'fr';

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-800">
      <div className="container mx-auto px-4 py-12">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
            {locale === 'fr' ? 'Blog' : 'Blog'}
          </h1>
          <p className="text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
            {locale === 'fr'
              ? 'Articles générés automatiquement depuis mes commits. Découvrez mon activité de développement.'
              : 'Articles automatically generated from my commits. Discover my development activity.'}
          </p>
        </div>

        {/* Blog List */}
        <BlogList locale={locale} />

        {/* RSS Feed Link */}
        <div className="mt-12 text-center">
          <a
            href="/api/v1/blog/feed.xml"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 text-orange-600 dark:text-orange-400 hover:text-orange-700 dark:hover:text-orange-300 transition-colors"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M6.503 20.752c0 1.794-1.456 3.248-3.251 3.248-1.796 0-3.252-1.454-3.252-3.248 0-1.794 1.456-3.248 3.252-3.248 1.795.001 3.251 1.454 3.251 3.248zm-6.503-12.572v4.811c6.05.062 10.96 4.966 11.022 11.009h4.817c-.062-8.71-7.118-15.758-15.839-15.82zm0-8.18v4.819c12.024.072 21.75 9.79 21.822 21.821h4.178c-.072-14.372-11.631-25.92-26-26z" />
            </svg>
            {locale === 'fr' ? 'Flux RSS' : 'RSS Feed'}
          </a>
        </div>
      </div>
    </div>
  );
}
