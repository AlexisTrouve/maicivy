'use client';

import { useEffect, useState } from 'react';
import { healthApi } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function ApiTestContent() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [data, setData] = useState<Record<string, unknown> | null>(null);

  useEffect(() => {
    healthApi
      .check()
      .then((res) => {
        setData(res);
        setStatus('success');
      })
      .catch((err) => {
        console.error(err);
        setStatus('error');
      });
  }, []);

  return (
    <div className="container py-12">
      <Card>
        <CardHeader>
          <CardTitle>API Health Check</CardTitle>
        </CardHeader>
        <CardContent>
          {status === 'loading' && <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full mx-auto" />}
          {status === 'success' && (
            <div className="text-green-600">
              OK API connectee: {JSON.stringify(data)}
            </div>
          )}
          {status === 'error' && (
            <div className="text-red-600">X Erreur de connexion API</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
