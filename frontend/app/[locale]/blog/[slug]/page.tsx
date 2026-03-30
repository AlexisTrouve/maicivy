import { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { BlogPostView } from '@/components/blog';
import { blogApi } from '@/lib/api';

interface BlogPostPageProps {
  params: Promise<{ locale: string; slug: string }> | { locale: string; slug: string };
}

async function getPost(slug: string) {
  try {
    return await blogApi.getPost(slug);
  } catch {
    return null;
  }
}

export async function generateMetadata({ params }: BlogPostPageProps): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const post = await getPost(resolvedParams.slug);

  if (!post) {
    return {
      title: 'Article non trouvé',
    };
  }

  const baseUrl = process.env.NEXT_PUBLIC_BASE_URL || 'https://maicivy.etheryale.com';

  return {
    title: post.title,
    description: post.summary,
    openGraph: {
      title: post.title,
      description: post.summary,
      type: 'article',
      publishedTime: post.published_at,
      authors: ['Alexis Trouvé'],
      tags: post.tags,
      images: [
        {
          // Utilise la cover du post si dispo, sinon image statique maiprofiles
          url: post.cover_image_url || 'https://maiprofiles.etheryale.com/images/img_dbb0624c',
          width: 1200,
          height: 630,
          alt: post.title,
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: post.title,
      description: post.summary,
      images: [post.cover_image_url || 'https://maiprofiles.etheryale.com/images/img_dbb0624c'],
    },
  };
}

export default async function BlogPostPage({ params }: BlogPostPageProps) {
  const resolvedParams = params instanceof Promise ? await params : params;
  const { locale, slug } = resolvedParams;

  const post = await getPost(slug);

  if (!post) {
    notFound();
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-800">
      <div className="container mx-auto px-4 py-12">
        <BlogPostView post={post} locale={locale} />
      </div>
    </div>
  );
}
