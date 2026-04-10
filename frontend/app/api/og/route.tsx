import { ImageResponse } from 'next/og';
import { NextRequest } from 'next/server';

export const runtime = 'edge';

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const locale = searchParams.get('locale') || 'fr';

  // Flag de locale pour les textes bilingues
  const isFr = locale === 'fr';

  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          background: '#0f172a',
          fontFamily: 'system-ui, -apple-system, sans-serif',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        {/* Gradient blobs en arrière-plan — bleu haut-gauche, violet bas-centre, cyan droite */}
        <div style={{
          position: 'absolute', top: '-100px', left: '-100px',
          width: '500px', height: '500px', borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(59,130,246,0.25) 0%, transparent 70%)',
          display: 'flex',
        }} />
        <div style={{
          position: 'absolute', bottom: '-80px', right: '200px',
          width: '400px', height: '400px', borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(139,92,246,0.2) 0%, transparent 70%)',
          display: 'flex',
        }} />
        <div style={{
          position: 'absolute', top: '100px', right: '-50px',
          width: '300px', height: '300px', borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(6,182,212,0.15) 0%, transparent 70%)',
          display: 'flex',
        }} />

        {/* Layout principal — deux colonnes : identité gauche, déco visuelle droite */}
        <div style={{
          display: 'flex', flexDirection: 'row',
          flex: 1, padding: '60px 70px',
          alignItems: 'center', justifyContent: 'space-between',
          position: 'relative', zIndex: 1,
        }}>

          {/* ── COLONNE GAUCHE : identité + features + tech ── */}
          <div style={{
            display: 'flex', flexDirection: 'column',
            alignItems: 'flex-start', justifyContent: 'center',
            flex: 1, paddingRight: '60px',
          }}>

            {/* Badge de disponibilité — fond vert translucide + point lumineux */}
            <div style={{
              display: 'flex', alignItems: 'center', gap: '8px',
              background: 'rgba(34,197,94,0.12)',
              border: '1px solid rgba(34,197,94,0.3)',
              borderRadius: '9999px',
              padding: '6px 16px',
              marginBottom: '28px',
            }}>
              <div style={{
                width: '8px', height: '8px', borderRadius: '50%',
                background: '#22c55e',
                /* Glow vert pour simuler un indicateur "online" */
                boxShadow: '0 0 8px rgba(34,197,94,0.8)',
                display: 'flex',
              }} />
              <span style={{ color: '#86efac', fontSize: '15px', fontWeight: 500 }}>
                {isFr ? 'Disponible · Missions freelance' : 'Available · Freelance contracts'}
              </span>
            </div>

            {/* Nom — grand, blanc, weight 800 pour l'impact visuel */}
            <div style={{
              fontSize: '62px', fontWeight: 800, color: '#f8fafc',
              lineHeight: 1.1, marginBottom: '10px',
              letterSpacing: '-1px',
              display: 'flex',
            }}>
              Alexis Trouvé
            </div>

            {/* Titre / poste — bleu clair pour contraste avec le nom blanc */}
            <div style={{
              fontSize: '28px', fontWeight: 500,
              color: '#60a5fa',
              marginBottom: '32px',
              display: 'flex',
            }}>
              {isFr ? 'Développeur Fullstack' : 'Fullstack Developer'}
            </div>

            {/* Séparateur gradient bleu→violet — accent visuel entre titre et features */}
            <div style={{
              width: '48px', height: '3px',
              background: 'linear-gradient(90deg, #3b82f6, #8b5cf6)',
              borderRadius: '2px',
              marginBottom: '28px',
              display: 'flex',
            }} />

            {/* Liste des features — 3 lignes, couleur muted pour ne pas écraser le nom */}
            <div style={{
              display: 'flex', flexDirection: 'column', gap: '8px',
              marginBottom: '36px',
            }}>
              {[
                isFr ? '⚡ CV interactif adaptatif par IA' : '⚡ AI-powered adaptive interactive CV',
                isFr ? '✉️ Génération de lettres de motivation' : '✉️ AI cover letter generation',
                isFr ? '📊 Analytics visiteurs temps réel' : '📊 Real-time visitor analytics',
              ].map((feature) => (
                <div key={feature} style={{
                  fontSize: '17px', color: '#94a3b8', display: 'flex',
                }}>
                  {feature}
                </div>
              ))}
            </div>

            {/* Badges tech — fond bleu translucide, bordure subtile */}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
              {['Go', 'Next.js', 'TypeScript', 'PostgreSQL', 'Redis', 'Claude AI'].map((tech) => (
                <div key={tech} style={{
                  padding: '6px 14px', borderRadius: '6px',
                  background: 'rgba(59,130,246,0.12)',
                  border: '1px solid rgba(59,130,246,0.25)',
                  color: '#93c5fd', fontSize: '14px', fontWeight: 500,
                  display: 'flex',
                }}>
                  {tech}
                </div>
              ))}
            </div>
          </div>

          {/* ── COLONNE DROITE : cercle AT + constellation décorative ── */}
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            width: '320px', position: 'relative',
          }}>

            {/* Points constellation — petits cercles bleus positionnés en absolu autour du cercle */}
            {[
              { top: '20px', left: '40px', size: '6px', opacity: 0.6 },
              { top: '60px', right: '30px', size: '4px', opacity: 0.4 },
              { top: '150px', left: '10px', size: '5px', opacity: 0.5 },
              { bottom: '80px', right: '20px', size: '7px', opacity: 0.5 },
              { bottom: '30px', left: '60px', size: '4px', opacity: 0.35 },
              { top: '220px', right: '60px', size: '5px', opacity: 0.45 },
            ].map((dot, i) => (
              <div key={i} style={{
                position: 'absolute',
                /* Spread les propriétés de position (top/left/right/bottom) */
                ...dot,
                width: dot.size, height: dot.size,
                borderRadius: '50%',
                background: '#60a5fa',
                /* Glow proportionnel à la taille du point */
                boxShadow: `0 0 ${parseInt(dot.size) * 3}px rgba(96,165,250,0.8)`,
                opacity: dot.opacity,
                display: 'flex',
              }} />
            ))}

            {/* Lignes constellation en SVG — relient les points, opacité 0.2 pour effet subtil */}
            <svg
              width="320" height="420"
              style={{ position: 'absolute', top: 0, left: 0, opacity: 0.2 }}
            >
              <line x1="46" y1="23" x2="290" y2="63" stroke="#3b82f6" strokeWidth="1" />
              <line x1="16" y1="153" x2="46" y2="23" stroke="#3b82f6" strokeWidth="1" />
              <line x1="290" y1="63" x2="280" y2="343" stroke="#3b82f6" strokeWidth="1" />
              <line x1="16" y1="153" x2="280" y2="343" stroke="#3b82f6" strokeWidth="1" />
              <line x1="120" y1="393" x2="280" y2="343" stroke="#3b82f6" strokeWidth="1" />
            </svg>

            {/* Cercle principal "AT" — gradient bleu→violet + glow multicouche pour effet premium */}
            <div style={{
              width: '180px', height: '180px', borderRadius: '50%',
              background: 'linear-gradient(135deg, #1d4ed8 0%, #7c3aed 100%)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              /* Triple glow : près/loin/inset pour profondeur */
              boxShadow: '0 0 60px rgba(59,130,246,0.5), 0 0 120px rgba(59,130,246,0.2), inset 0 1px 0 rgba(255,255,255,0.15)',
              border: '1px solid rgba(255,255,255,0.1)',
              position: 'relative', zIndex: 2,
            }}>
              <span style={{
                fontSize: '64px', fontWeight: 800, color: 'white',
                letterSpacing: '-2px', display: 'flex',
              }}>
                AT
              </span>
            </div>

            {/* Anneaux concentriques — deux cercles en border-only autour du cercle principal */}
            <div style={{
              position: 'absolute',
              width: '220px', height: '220px', borderRadius: '50%',
              border: '1px solid rgba(59,130,246,0.2)',
              display: 'flex',
            }} />
            <div style={{
              position: 'absolute',
              width: '260px', height: '260px', borderRadius: '50%',
              border: '1px solid rgba(59,130,246,0.08)',
              display: 'flex',
            }} />
          </div>
        </div>

        {/* Footer — URL à gauche, indicateur dispo à droite */}
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '0 70px 30px',
          position: 'relative', zIndex: 1,
        }}>
          <span style={{ color: '#334155', fontSize: '15px' }}>
            maicivy.etheryale.com
          </span>
          <div style={{
            display: 'flex', alignItems: 'center', gap: '6px',
          }}>
            {/* Petit point vert statique — indicateur de disponibilité discret */}
            <div style={{
              width: '6px', height: '6px', borderRadius: '50%',
              background: '#22c55e', display: 'flex',
            }} />
            <span style={{ color: '#475569', fontSize: '14px' }}>
              {isFr ? 'Ouvert aux opportunités' : 'Open to opportunities'}
            </span>
          </div>
        </div>
      </div>
    ),
    { width: 1200, height: 630 }
  );
}
