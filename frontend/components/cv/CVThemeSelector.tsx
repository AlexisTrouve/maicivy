'use client';

import { useState, useEffect, useTransition } from 'react';
import { useSearchParams } from 'next/navigation';
import { useRouter } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Sparkles } from 'lucide-react';
import { Theme } from '@/lib/types';

// Fetch available themes from API
async function fetchThemes(): Promise<Theme[]> {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/cv/themes`);
  if (!res.ok) throw new Error('Failed to fetch themes');
  const data = await res.json();
  return data.themes || [];
}

interface CVThemeSelectorProps {
  currentTheme: string;
}

export default function CVThemeSelector({ currentTheme }: CVThemeSelectorProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [themes, setThemes] = useState<Theme[]>([]);
  const [loading, setLoading] = useState(true);
  const [isPending, startTransition] = useTransition();
  const t = useTranslations('cv');

  useEffect(() => {
    fetchThemes()
      .then(setThemes)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const handleThemeChange = (newTheme: string) => {
    if (newTheme === currentTheme) return;

    const params = new URLSearchParams(searchParams.toString());
    params.set('theme', newTheme);

    // Use startTransition for smoother navigation without blocking UI
    startTransition(() => {
      router.push(`/cv?${params.toString()}`);
    });
  };

  if (loading) {
    return (
      <div className="w-64 h-10 bg-gray-200 dark:bg-gray-700 rounded-md animate-pulse" />
    );
  }

  return (
    <div className="relative">
      <Select value={currentTheme} onValueChange={handleThemeChange} disabled={isPending}>
        <SelectTrigger
          className={`w-64 bg-white dark:bg-gray-800 border-2 border-gray-300 dark:border-gray-600 hover:border-blue-500 transition-colors ${
            isPending ? 'opacity-70 cursor-wait' : ''
          }`}
        >
          {isPending ? (
            <div className="w-4 h-4 mr-2 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
          ) : (
            <Sparkles className="w-4 h-4 mr-2 text-blue-600" />
          )}
          <SelectValue placeholder={t('selectTheme')} />
        </SelectTrigger>
        <SelectContent>
          {themes.map((theme) => (
            <SelectItem key={theme.id} value={theme.id}>
              <div className="flex items-center gap-2">
                <span>{theme.icon}</span>
                <div>
                  <div className="font-medium">{theme.name}</div>
                  <div className="text-xs text-gray-500">{theme.description}</div>
                </div>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
