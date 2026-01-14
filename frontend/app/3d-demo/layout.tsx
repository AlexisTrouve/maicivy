import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: '3D Demo - maicivy',
  description: 'Demonstration des effets 3D avec Three.js',
};

export default function Demo3DLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
