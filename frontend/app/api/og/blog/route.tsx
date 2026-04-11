import { ImageResponse } from 'next/og';
import { NextRequest } from 'next/server';

export const runtime = 'edge';

// Fetch blog post from backend
async function getBlogPost(slug: string) {
  try {
    const apiUrl = process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const response = await fetch(`${apiUrl}/api/v1/blog/posts/${slug}`, {
      next: { revalidate: 3600 }, // Cache 1h
    });
    if (!response.ok) return null;
    return await response.json();
  } catch {
    return null;
  }
}

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const slug = searchParams.get('slug') || '';
  const locale = searchParams.get('locale') || 'fr';

  // Fetch post data
  const post = await getBlogPost(slug);

  const title = post?.title || 'Article';
  const project = post?.project_name || 'maicivy';
  const readingTime = post?.reading_time_minutes || 3;
  const tags = post?.tags?.slice(0, 3) || [];

  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 50%, #1e293b 100%)',
          fontFamily: 'system-ui, sans-serif',
          padding: '60px',
        }}
      >
        {/* Decorative gradient */}
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'radial-gradient(circle at 80% 20%, rgba(59, 130, 246, 0.15) 0%, transparent 40%)',
          }}
        />

        {/* Top bar */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '40px',
          }}
        >
          {/* Logo */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
            }}
          >
            <div
              style={{
                width: '50px',
                height: '50px',
                borderRadius: '12px',
                background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <span style={{ fontSize: '24px', color: 'white', fontWeight: 700 }}>AT</span>
            </div>
            <span style={{ fontSize: '24px', color: '#94a3b8', fontWeight: 500 }}>
              maicivy blog
            </span>
          </div>

          {/* Reading time */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              padding: '10px 20px',
              background: 'rgba(255, 255, 255, 0.05)',
              borderRadius: '9999px',
              color: '#94a3b8',
              fontSize: '18px',
            }}
          >
            <span>{readingTime} min</span>
          </div>
        </div>

        {/* Main content */}
        <div style={{ display: 'flex', flexDirection: 'column', flex: 1, justifyContent: 'center' }}>
          {/* Project badge */}
          <div
            style={{
              display: 'flex',
              marginBottom: '20px',
            }}
          >
            <span
              style={{
                padding: '8px 16px',
                background: 'rgba(59, 130, 246, 0.2)',
                border: '1px solid rgba(59, 130, 246, 0.3)',
                borderRadius: '9999px',
                color: '#60a5fa',
                fontSize: '18px',
                fontWeight: 500,
              }}
            >
              {project}
            </span>
          </div>

          {/* Title */}
          <h1
            style={{
              fontSize: '52px',
              fontWeight: 700,
              color: 'white',
              lineHeight: 1.2,
              marginBottom: '30px',
              maxWidth: '900px',
            }}
          >
            {title.length > 80 ? title.slice(0, 80) + '...' : title}
          </h1>

          {/* Tags */}
          {tags.length > 0 && (
            <div
              style={{
                display: 'flex',
                gap: '12px',
              }}
            >
              {tags.map((tag: string) => (
                <span
                  key={tag}
                  style={{
                    padding: '8px 16px',
                    background: 'rgba(255, 255, 255, 0.05)',
                    border: '1px solid rgba(255, 255, 255, 0.1)',
                    borderRadius: '9999px',
                    color: '#94a3b8',
                    fontSize: '16px',
                  }}
                >
                  #{tag}
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            color: '#475569',
            fontSize: '16px',
          }}
        >
          <span>maicivy.etheryale.com/blog</span>
          <span>Alexis Trouvé</span>
        </div>
      </div>
    ),
    {
      width: 1200,
      height: 630,
    }
  );
}
