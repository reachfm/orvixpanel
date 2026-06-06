import { useEffect, useState } from 'react';
import { useTheme } from './hooks/useTheme';

interface Account {
  id: string;
  username: string;
  domain: string;
  plan: string;
  status: string;
  disk_used_mb: number;
  disk_quota_mb: number;
  bandwidth_used_gb: number;
  bandwidth_gb: number;
}

interface MeResponse {
  user_id: string;
  email: string;
  role: string;
  tenant_id: string;
}

export default function Dashboard() {
  const theme = useTheme<{ brand_name: string; primary_color: string }>('/api/v1/public/theme');
  const [me, setMe] = useState<MeResponse | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [wsSnap, setWsSnap] = useState<{ cpu: { usage_percent: number }; memory: { usage_percent: number }; load: number[] } | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (!token) {
      window.location.href = '/login';
      return;
    }
    fetch('/api/v1/me', { headers: { Authorization: `Bearer ${token}` } })
      .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
      .then(setMe)
      .catch(() => {
        localStorage.removeItem('access_token');
        window.location.href = '/login';
      });
    fetch('/api/v1/accounts', { headers: { Authorization: `Bearer ${token}` } })
      .then((r) => r.json())
      .then((d) => setAccounts(d.accounts || []));
  }, []);

  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (!token) return;
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${proto}//${window.location.host}/api/v1/ws/metrics`);
    ws.onmessage = (e) => {
      const snap = JSON.parse(e.data);
      setWsSnap({
        cpu: snap.cpu,
        memory: snap.memory,
        load: snap.load,
      });
    };
    return () => ws.close();
  }, []);

  return (
    <main style={{ minHeight: '100vh', background: '#0b1020', color: '#e8ecf3', fontFamily: 'system-ui' }}>
      <header
        style={{
          background: 'rgba(255,255,255,0.04)',
          borderBottom: '1px solid rgba(255,255,255,0.08)',
          padding: '14px 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <h1 style={{ margin: 0, fontSize: 18, color: theme?.primary_color ?? '#1a73e8' }}>
          {theme?.brand_name ?? 'OrvixPanel'} — Dashboard
        </h1>
        <div style={{ fontSize: 13, color: '#99a3b3' }}>
          {me?.email} · <span style={{ color: theme?.primary_color }}>{me?.role}</span>
        </div>
      </header>

      <section style={{ padding: 24, display: 'grid', gap: 16, gridTemplateColumns: '1fr 1fr' }}>
        <div style={cardStyle}>
          <h3 style={cardTitle}>System</h3>
          {wsSnap ? (
            <div style={{ display: 'grid', gap: 8, marginTop: 8 }}>
              <Bar label="CPU" pct={wsSnap.cpu.usage_percent} />
              <Bar label="Memory" pct={wsSnap.memory.usage_percent} />
              <div style={{ fontSize: 12, color: '#99a3b3', marginTop: 8 }}>
                Load: {wsSnap.load.map((l) => l.toFixed(2)).join(' · ')}
              </div>
            </div>
          ) : (
            <div style={{ color: '#99a3b3', fontSize: 13 }}>Connecting to live feed…</div>
          )}
        </div>

        <div style={cardStyle}>
          <h3 style={cardTitle}>Accounts ({accounts.length})</h3>
          {accounts.length === 0 ? (
            <div style={{ color: '#99a3b3', fontSize: 13 }}>No accounts yet.</div>
          ) : (
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {accounts.slice(0, 5).map((a) => (
                <li
                  key={a.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    padding: '8px 0',
                    borderBottom: '1px solid rgba(255,255,255,0.06)',
                    fontSize: 13,
                  }}
                >
                  <span>{a.username} <span style={{ color: '#99a3b3' }}>· {a.domain}</span></span>
                  <span style={{ color: a.status === 'active' ? '#7be38e' : '#ffb3b3' }}>{a.status}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>
    </main>
  );
}

function Bar({ label, pct }: { label: string; pct: number }) {
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: '#bcc6d4' }}>
        <span>{label}</span>
        <span>{pct.toFixed(1)}%</span>
      </div>
      <div
        style={{
          height: 6,
          background: 'rgba(255,255,255,0.06)',
          borderRadius: 3,
          marginTop: 4,
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            width: `${Math.min(100, pct)}%`,
            height: '100%',
            background: pct > 85 ? '#ff7373' : pct > 60 ? '#ffc14d' : '#7be38e',
            transition: 'width 300ms',
          }}
        />
      </div>
    </div>
  );
}

const cardStyle: React.CSSProperties = {
  background: 'rgba(255,255,255,0.04)',
  border: '1px solid rgba(255,255,255,0.08)',
  borderRadius: 12,
  padding: 20,
};

const cardTitle: React.CSSProperties = { margin: 0, fontSize: 14, color: '#bcc6d4' };
