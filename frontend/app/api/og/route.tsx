import { ImageResponse } from 'next/og';
import { NextRequest } from 'next/server';

export const runtime = 'edge';

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const locale = searchParams.get('locale') || 'fr';

  // Textes selon la locale
  const texts = {
    fr: {
      title: 'Alexis Trouvé',
      subtitle: 'Développeur Full-Stack',
      description: 'CV intelligent avec IA',
      stats: ['Go', 'TypeScript', 'Next.js', 'PostgreSQL'],
    },
    en: {
      title: 'Alexis Trouvé',
      subtitle: 'Full-Stack Developer',
      description: 'AI-powered intelligent CV',
      stats: ['Go', 'TypeScript', 'Next.js', 'PostgreSQL'],
    },
  };

  const t = texts[locale as keyof typeof texts] || texts.fr;

  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)',
          fontFamily: 'system-ui, sans-serif',
        }}
      >
        {/* Decorative elements */}
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'radial-gradient(circle at 20% 80%, rgba(59, 130, 246, 0.15) 0%, transparent 50%), radial-gradient(circle at 80% 20%, rgba(139, 92, 246, 0.15) 0%, transparent 50%)',
          }}
        />

        {/* Main content */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '60px',
            textAlign: 'center',
          }}
        >
          {/* Logo/Avatar placeholder */}
          <div
            style={{
              width: '120px',
              height: '120px',
              borderRadius: '50%',
              background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginBottom: '30px',
              boxShadow: '0 20px 40px rgba(0, 0, 0, 0.3)',
            }}
          >
            <span style={{ fontSize: '48px', color: 'white', fontWeight: 700 }}>AT</span>
          </div>

          {/* Title */}
          <h1
            style={{
              fontSize: '64px',
              fontWeight: 700,
              color: 'white',
              margin: 0,
              marginBottom: '10px',
              textShadow: '0 4px 20px rgba(0, 0, 0, 0.3)',
            }}
          >
            {t.title}
          </h1>

          {/* Subtitle */}
          <p
            style={{
              fontSize: '32px',
              color: '#94a3b8',
              margin: 0,
              marginBottom: '20px',
            }}
          >
            {t.subtitle}
          </p>

          {/* Description */}
          <p
            style={{
              fontSize: '24px',
              color: '#64748b',
              margin: 0,
              marginBottom: '40px',
            }}
          >
            {t.description}
          </p>

          {/* Tech stack badges */}
          <div
            style={{
              display: 'flex',
              gap: '16px',
              flexWrap: 'wrap',
              justifyContent: 'center',
            }}
          >
            {t.stats.map((tech) => (
              <div
                key={tech}
                style={{
                  padding: '12px 24px',
                  borderRadius: '9999px',
                  background: 'rgba(59, 130, 246, 0.2)',
                  border: '1px solid rgba(59, 130, 246, 0.3)',
                  color: '#60a5fa',
                  fontSize: '20px',
                  fontWeight: 500,
                }}
              >
                {tech}
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            position: 'absolute',
            bottom: '30px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            color: '#475569',
            fontSize: '18px',
          }}
        >
          <span>maicivy.etheryale.com</span>
        </div>
      </div>
    ),
    {
      width: 1200,
      height: 630,
    }
  );
}
