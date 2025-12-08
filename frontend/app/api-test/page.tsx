'use client';

import { useEffect, useState } from 'react';
import { healthApi } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSpinner } from '@/components/shared/LoadingSpinner';

export default function ApiTestPage() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [data, setData] = useState<any>(null);

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
          {status === 'loading' && <LoadingSpinner />}
          {status === 'success' && (
            <div className="text-green-600">
              ✓ API connectée: {JSON.stringify(data)}
            </div>
          )}
          {status === 'error' && (
            <div className="text-red-600">✗ Erreur de connexion API</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
