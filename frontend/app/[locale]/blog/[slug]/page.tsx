import { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations } from 'next-intl/server';
import { BlogPostView } from '@/components/blog';
import { blogApi } from '@/lib/api';
import { blogPreviewImageUrl, socialMetadata } from '@/lib/seo/social';

interface BlogPostPageProps {
  params: Promise<{ locale: string; slug: string }>;
}

// Rend l'article, `null` s'il n'existe pas, et relance toute autre erreur.
//
// POURQUOI : auparavant toute erreur devenait `null` puis `notFound()` — une panne de maiProFiles
// faisait passer des articles existants pour supprimés, pour les visiteurs comme pour Google.
// COMMENT : seul un 404 du backend (ErrBlogPostAbsent côté Go) mène à notFound(). Une panne (502,
// timeout, réseau) remonte à la frontière d'erreur `[locale]/error.tsx` (message + Réessayer).
async function getPost(slug: string, lang?: string) {
  try {
    return await blogApi.getPost(slug, lang);
  } catch (error) {
    if ((error as { statusCode?: number })?.statusCode === 404) return null;
    throw error;
  }
}

export async function generateMetadata({ params }: BlogPostPageProps): Promise<Metadata> {
  const resolvedParams = await params;
  const post = await getPost(resolvedParams.slug, resolvedParams.locale);

  if (!post) {
    const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'blog.post' });
    return { title: t('notFound') };
  }

  return {
    title: post.title,
    description: post.summary,
    ...socialMetadata({
      locale: resolvedParams.locale,
      title: post.title,
      description: post.summary,
      type: 'article',
      extra: { publishedTime: post.published_at, authors: ['Alexis Trouvé'], tags: post.tags },
      // La couverture de l'article si elle existe (dimensions inconnues : non déclarées), sinon
      // l'aperçu généré de l'article — titre et sujets au style du site.
      image: post.cover_image_url
        ? { url: post.cover_image_url }
        : { url: blogPreviewImageUrl(resolvedParams.locale, post.slug), width: 1200, height: 630 },
    }),
  };
}

export default async function BlogPostPage({ params }: BlogPostPageProps) {
  const resolvedParams = await params;
  const { locale, slug } = resolvedParams;

  const post = await getPost(slug, locale);

  if (!post) {
    notFound();
  }

  // Pas de fond propre : l'article se lit sur la nuit du site (le ciel posé par le layout).
  return (
    <div className="px-5 pb-20 pt-8 md:px-8 md:pt-14">
      <BlogPostView post={post} locale={locale} />
    </div>
  );
}
