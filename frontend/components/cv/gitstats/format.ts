// ============================================================================
// Formats de la page /gitstats — tous dans la locale du visiteur, jamais 'fr-FR' en dur.
// ----------------------------------------------------------------------------
// Les dates arrivent en "YYYY-MM-DD" (jours nus) : on les lit à midi UTC et on les formate en UTC,
// pour qu'aucun fuseau ne les fasse glisser d'un jour.
// ============================================================================

export function createFormats(locale: string) {
  const whole = new Intl.NumberFormat(locale);
  const thousands = new Intl.NumberFormat(locale, { notation: 'compact', maximumFractionDigits: 0 });
  const millions = new Intl.NumberFormat(locale, { notation: 'compact', maximumFractionDigits: 1 });
  const percent = new Intl.NumberFormat(locale, { style: 'percent', minimumFractionDigits: 1, maximumFractionDigits: 1 });
  const date = (iso: string, opts: Intl.DateTimeFormatOptions) =>
    new Date(`${iso}T12:00:00Z`).toLocaleDateString(locale, { ...opts, timeZone: 'UTC' });

  return {
    number: (n: number) => whole.format(n),
    /** « 12,3 M », « 475 k » ; en dessous de 10 000, le nombre entier (« 2 033 ») reste lisible. */
    short: (n: number) => (Math.abs(n) >= 1e6 ? millions.format(n) : Math.abs(n) >= 1e4 ? thousands.format(n) : whole.format(n)),
    percent: (share: number) => percent.format(share),
    day: (iso: string) => date(iso, { day: 'numeric', month: 'short' }),
    dayLong: (iso: string) => date(iso, { day: 'numeric', month: 'long', year: 'numeric' }),
    month: (iso: string) => date(iso, { month: 'short' }),
  };
}

export type Formats = ReturnType<typeof createFormats>;
