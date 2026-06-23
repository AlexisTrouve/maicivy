'use client';

import { useState } from 'react';
import { Calendar } from 'lucide-react';
import { useTranslations, useLocale } from 'next-intl';
import { useRouter, usePathname, useSearchParams } from 'next/navigation';

type PeriodPreset = 'today' | '7d' | '30d' | 'all';

interface DateRange {
  from: Date | undefined;
  to: Date | undefined;
}

export default function DateFilter() {
  const t = useTranslations('analytics.periods');
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // Sélection initiale lue depuis l'URL (?period=) → cohérent au reload / partage de lien ; défaut 7 jours.
  const initialPreset = (searchParams.get('period') as PeriodPreset) || '7d';
  const [selectedPreset, setSelectedPreset] = useState<PeriodPreset>(initialPreset);
  // dateRange reste calculé au clic (pas au montage) pour éviter un mismatch d'hydratation sur new Date().
  const [dateRange, setDateRange] = useState<DateRange>({ from: undefined, to: undefined });

  const presets: { value: PeriodPreset; label: string }[] = [
    { value: 'today', label: t('today') },
    { value: '7d', label: t('last7days') },
    { value: '30d', label: t('last30days') },
    { value: 'all', label: t('all') },
  ];

  const handlePresetChange = (preset: PeriodPreset) => {
    setSelectedPreset(preset);

    // Calculate date range based on preset (affichage)
    const now = new Date();
    const from = new Date();

    switch (preset) {
      case 'today':
        from.setHours(0, 0, 0, 0);
        setDateRange({ from, to: now });
        break;
      case '7d':
        from.setDate(now.getDate() - 7);
        setDateRange({ from, to: now });
        break;
      case '30d':
        from.setDate(now.getDate() - 30);
        setDateRange({ from, to: now });
        break;
      case 'all':
        setDateRange({ from: undefined, to: undefined });
        break;
    }

    // Écrit la période dans l'URL → c'est CE qui pilote les widgets (StatsOverview lit ?period=).
    // Avant, le filtre n'avait aucun consommateur → cliquer ne changeait rien. router.replace = pas
    // d'entrée d'historique en plus, scroll: false = on ne saute pas en haut de page.
    const params = new URLSearchParams(searchParams.toString());
    params.set('period', preset);
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  };

  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Calendar className="h-4 w-4" />
        <span>{t('label')}</span>
      </div>

      <div className="flex gap-1 bg-muted p-1 rounded-md">
        {presets.map((preset) => (
          <button
            key={preset.value}
            onClick={() => handlePresetChange(preset.value)}
            className={`px-3 py-1.5 text-sm rounded transition-colors ${
              selectedPreset === preset.value
                ? 'bg-primary text-primary-foreground'
                : 'hover:bg-background'
            }`}
          >
            {preset.label}
          </button>
        ))}
      </div>

      {dateRange.from && dateRange.to && (
        <div className="text-xs text-muted-foreground ml-2">
          {dateRange.from.toLocaleDateString(locale, {
            day: '2-digit',
            month: 'short',
          })}{' '}
          -{' '}
          {dateRange.to.toLocaleDateString(locale, {
            day: '2-digit',
            month: 'short',
            year: 'numeric',
          })}
        </div>
      )}
    </div>
  );
}
