'use client';

import { useState } from 'react';
import { Calendar } from 'lucide-react';

type PeriodPreset = 'today' | '7d' | '30d' | 'all';

interface DateRange {
  from: Date | undefined;
  to: Date | undefined;
}

export default function DateFilter() {
  const [selectedPreset, setSelectedPreset] = useState<PeriodPreset>('7d');
  const [dateRange, setDateRange] = useState<DateRange>({
    from: undefined,
    to: undefined,
  });

  const presets: { value: PeriodPreset; label: string }[] = [
    { value: 'today', label: "Aujourd'hui" },
    { value: '7d', label: '7 derniers jours' },
    { value: '30d', label: '30 derniers jours' },
    { value: 'all', label: 'Tout' },
  ];

  const handlePresetChange = (preset: PeriodPreset) => {
    setSelectedPreset(preset);

    // Calculate date range based on preset
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
  };

  const formatDateRange = (): string => {
    const preset = presets.find((p) => p.value === selectedPreset);
    return preset?.label || 'Sélectionner une période';
  };

  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Calendar className="h-4 w-4" />
        <span>Période:</span>
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
          {dateRange.from.toLocaleDateString('fr-FR', {
            day: '2-digit',
            month: 'short',
          })}{' '}
          -{' '}
          {dateRange.to.toLocaleDateString('fr-FR', {
            day: '2-digit',
            month: 'short',
            year: 'numeric',
          })}
        </div>
      )}
    </div>
  );
}
