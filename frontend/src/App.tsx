import { useEffect, useState } from 'react';
import { useTheme } from './hooks/useTheme';

interface Theme {
  brand_name: string;
  primary_color: string;
  secondary_color: string;
  logo_url: string;
  support_email: string;
}

export default function App() {
  const theme = useTheme<Theme>('/api/v1/public/theme');
  const [health, setHealth] = useState<string>('checking…');

  useEffect(() => {
    fetch('/healthz')
      .then((r) => r.json())
      .then((d) => setHealth(d.status))
      .catch(() => setHealth('unreachable'));
  }, []);

  return (
    <main
      style={{
        minHeight: '100vh',
        background: theme?.secondary_color ?? '#0b1020',
        color: '#e8ecf3',
        display: 'grid',
        placeItems: 'center',
        fontFamily: 'system-ui, sans-serif',
        padding: 24,
      }}
    >
      <section
        style={{
          maxWidth: 520,
          textAlign: 'center',
          background: 'rgba(255,255,255,0.04)',
          padding: 32,
          borderRadius: 12,
          border: '1px solid rgba(255,255,255,0.08)',
        }}
      >
        <h1 style={{ color: theme?.primary_color ?? '#1a73e8', margin: 0 }}>
          {theme?.brand_name ?? 'OrvixPanel'}
        </h1>
        <p style={{ color: '#99a3b3', lineHeight: 1.5, marginTop: 12 }}>
          Phase 1 frontend scaffold is live. API health: <strong>{health}</strong>.
        </p>
        <p style={{ color: '#99a3b3', fontSize: 14 }}>
          Login UI, dashboard, and per-feature pages arrive in later phases.
        </p>
      </section>
    </main>
  );
}
