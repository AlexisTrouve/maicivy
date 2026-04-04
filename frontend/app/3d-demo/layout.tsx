import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: '3D Demo - maicivy',
  description: 'Demonstration des effets 3D avec Three.js',
};

// Layout avec <html>/<body> obligatoire — cette route est hors [locale]
// donc le root layout ne fournit pas de structure document
export default function Demo3DLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
