// Root layout - minimal wrapper, pas de <html>/<body> ici car [locale]/layout les fournit
// Les routes hors [locale] (3d-demo, api-test) doivent fournir leurs propres <html>/<body>
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
