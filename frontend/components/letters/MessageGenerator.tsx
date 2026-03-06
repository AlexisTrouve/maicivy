'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Copy, Check, Loader2, Send } from 'lucide-react';
import { messagesApi } from '@/lib/api';
import type { PlatformMessageResponse } from '@/lib/types';

const PLATFORMS = [
  { value: 'malt', label: 'Malt' },
  { value: 'linkedin', label: 'LinkedIn' },
  { value: 'upwork', label: 'Upwork' },
];

export function MessageGenerator() {
  const [mission, setMission] = useState('');
  const [platform, setPlatform] = useState('malt');
  const [tjm, setTjm] = useState<string>('');
  const [result, setResult] = useState<PlatformMessageResponse | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleGenerate = async () => {
    if (mission.length < 20) {
      setError('La description de la mission est trop courte.');
      return;
    }

    setIsGenerating(true);
    setError(null);
    setResult(null);

    try {
      const data = await messagesApi.generate({
        mission,
        platform: platform as 'malt' | 'linkedin' | 'upwork',
        tjm: tjm ? parseInt(tjm, 10) : undefined,
      });
      setResult(data);
    } catch (err: any) {
      const status = err.statusCode;
      if (status === 429) {
        setError('Limite journalière atteinte. Réessaie demain.');
      } else {
        setError(err.message || 'Erreur lors de la génération.');
      }
    } finally {
      setIsGenerating(false);
    }
  };

  const handleCopy = async () => {
    if (!result) return;
    await navigator.clipboard.writeText(result.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleReset = () => {
    setResult(null);
    setError(null);
    setCopied(false);
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700">

        {!result ? (
          <div className="space-y-6">
            {/* Platform selector */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Plateforme
              </label>
              <div className="flex gap-2">
                {PLATFORMS.map((p) => (
                  <button
                    key={p.value}
                    type="button"
                    onClick={() => setPlatform(p.value)}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                      platform === p.value
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
                    }`}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Mission textarea */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Description de la mission
                <span className="text-slate-400 font-normal ml-1">(colle l&apos;annonce ici)</span>
              </label>
              <textarea
                value={mission}
                onChange={(e) => setMission(e.target.value)}
                rows={8}
                placeholder="Colle ici la description de la mission depuis Malt, LinkedIn ou Upwork..."
                disabled={isGenerating}
                className="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all resize-none disabled:opacity-50"
              />
              <p className="mt-1 text-xs text-slate-400">{mission.length} caractères</p>
            </div>

            {/* TJM */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                TJM
                <span className="text-slate-400 font-normal ml-1">(€, optionnel)</span>
              </label>
              <input
                type="number"
                value={tjm}
                onChange={(e) => setTjm(e.target.value)}
                placeholder="ex: 550"
                disabled={isGenerating}
                min={50}
                max={5000}
                className="w-32 px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all disabled:opacity-50"
              />
            </div>

            {/* Error */}
            {error && (
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4"
              >
                <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
              </motion.div>
            )}

            {/* Generate button */}
            <button
              type="button"
              onClick={handleGenerate}
              disabled={isGenerating || mission.length < 20}
              className="w-full bg-gradient-to-r from-blue-600 to-purple-600 text-white px-6 py-4 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {isGenerating ? (
                <>
                  <Loader2 className="w-5 h-5 animate-spin" />
                  Génération en cours...
                </>
              ) : (
                <>
                  <Send className="w-5 h-5" />
                  Générer le message
                </>
              )}
            </button>
          </div>
        ) : (
          /* Result */
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="space-y-4"
          >
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-500 dark:text-slate-400 capitalize">
                Message {result.platform} · {result.tokens_used} tokens
              </span>
              <button
                onClick={handleCopy}
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 text-sm transition-all"
              >
                {copied ? (
                  <><Check className="w-4 h-4 text-green-500" /> Copié</>
                ) : (
                  <><Copy className="w-4 h-4" /> Copier</>
                )}
              </button>
            </div>

            <div className="bg-slate-50 dark:bg-slate-900 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
              <pre className="whitespace-pre-wrap text-sm text-slate-800 dark:text-slate-200 font-sans leading-relaxed">
                {result.content}
              </pre>
            </div>

            <button
              onClick={handleReset}
              className="w-full py-3 rounded-lg border border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 text-sm transition-all"
            >
              Nouveau message
            </button>
          </motion.div>
        )}
      </div>
    </div>
  );
}
