import { ImageResponse } from 'next/og';
import { NextRequest } from 'next/server';

export const runtime = 'edge';

// Fetch activity stats from backend
async function getActivityStats() {
  try {
    const apiUrl = process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const response = await fetch(`${apiUrl}/api/v1/activity/stats`, {
      next: { revalidate: 900 }, // Cache 15 min
    });
    if (!response.ok) return null;
    return await response.json();
  } catch {
    return null;
  }
}

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const locale = searchParams.get('locale') || 'fr';

  // Fetch real stats
  const stats = await getActivityStats();

  // Textes selon la locale
  const texts = {
    fr: {
      title: 'Alexis Trouvé',
      subtitle: 'Développeur Full-Stack',
      commits: 'commits (30j)',
      projects: 'projets actifs',
    },
    en: {
      title: 'Alexis Trouvé',
      subtitle: 'Full-Stack Developer',
      commits: 'commits (30d)',
      projects: 'active projects',
    },
  };

  const t = texts[locale as keyof typeof texts] || texts.fr;

  const commitCount = stats?.total_commits_30d || 69;
  const projectCount = stats?.active_projects || 8;
  const topLangs = stats?.top_languages?.slice(0, 4) || ['Go', 'TypeScript', 'Python', 'Next.js'];

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
          background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%)',
          fontFamily: 'system-ui, sans-serif',
        }}
      >
        {/* Animated gradient background */}
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'radial-gradient(ellipse at 30% 20%, rgba(59, 130, 246, 0.2) 0%, transparent 50%), radial-gradient(ellipse at 70% 80%, rgba(16, 185, 129, 0.15) 0%, transparent 50%)',
          }}
        />

        {/* Main content */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '50px',
          }}
        >
          {/* Header with avatar */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '24px',
              marginBottom: '40px',
            }}
          >
            <div
              style={{
                width: '100px',
                height: '100px',
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #3b82f6 0%, #10b981 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: '0 15px 30px rgba(0, 0, 0, 0.4)',
              }}
            >
              <span style={{ fontSize: '40px', color: 'white', fontWeight: 700 }}>AT</span>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column' }}>
              <h1
                style={{
                  fontSize: '48px',
                  fontWeight: 700,
                  color: 'white',
                  margin: 0,
                }}
              >
                {t.title}
              </h1>
              <p
                style={{
                  fontSize: '24px',
                  color: '#94a3b8',
                  margin: 0,
                }}
              >
                {t.subtitle}
              </p>
            </div>
          </div>

          {/* Stats cards */}
          <div
            style={{
              display: 'flex',
              gap: '30px',
              marginBottom: '40px',
            }}
          >
            {/* Commits stat */}
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                padding: '30px 50px',
                borderRadius: '20px',
                background: 'rgba(59, 130, 246, 0.1)',
                border: '1px solid rgba(59, 130, 246, 0.2)',
              }}
            >
              <span
                style={{
                  fontSize: '56px',
                  fontWeight: 700,
                  color: '#3b82f6',
                }}
              >
                {commitCount}
              </span>
              <span style={{ fontSize: '18px', color: '#60a5fa' }}>{t.commits}</span>
            </div>

            {/* Projects stat */}
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                padding: '30px 50px',
                borderRadius: '20px',
                background: 'rgba(16, 185, 129, 0.1)',
                border: '1px solid rgba(16, 185, 129, 0.2)',
              }}
            >
              <span
                style={{
                  fontSize: '56px',
                  fontWeight: 700,
                  color: '#10b981',
                }}
              >
                {projectCount}
              </span>
              <span style={{ fontSize: '18px', color: '#34d399' }}>{t.projects}</span>
            </div>
          </div>

          {/* Tech stack */}
          <div
            style={{
              display: 'flex',
              gap: '12px',
              flexWrap: 'wrap',
              justifyContent: 'center',
            }}
          >
            {topLangs.map((tech: string) => (
              <div
                key={tech}
                style={{
                  padding: '10px 20px',
                  borderRadius: '9999px',
                  background: 'rgba(255, 255, 255, 0.05)',
                  border: '1px solid rgba(255, 255, 255, 0.1)',
                  color: '#e2e8f0',
                  fontSize: '18px',
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
            bottom: '25px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            color: '#475569',
            fontSize: '16px',
          }}
        >
          <span>maicivy.etheryale.com</span>
          <span style={{ color: '#334155' }}>|</span>
          <span>CV dynamique</span>
        </div>
      </div>
    ),
    {
      width: 1200,
      height: 630,
    }
  );
}
