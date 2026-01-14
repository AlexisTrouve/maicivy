import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'API Test - maicivy',
  description: 'Test de connexion API backend',
};

export default function ApiTestLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
