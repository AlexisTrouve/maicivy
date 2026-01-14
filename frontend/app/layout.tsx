// Root layout - minimal wrapper for routes outside [locale] folder
// Most routes are in [locale] which has the full layout with HTML/body tags
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
