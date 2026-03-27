'use client';

// CenterZone — zone centrale réservée (20% de la largeur)
// Espace vide décoratif entre le chat et le panel droit, avec un séparateur visuel subtil.
export function CenterZone() {
  return (
    <div className="w-[20%] flex flex-col items-center justify-center border-x relative">
      {/* Ligne verticale décorative */}
      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
        <div className="w-px h-full bg-gradient-to-b from-transparent via-border to-transparent opacity-50" />
      </div>

      {/* Badge central */}
      <div className="relative z-10 flex flex-col items-center gap-2 text-muted-foreground/40 select-none">
        <div className="text-2xl">✦</div>
      </div>
    </div>
  );
}
