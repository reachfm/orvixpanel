import { useState } from 'react';
import { useTheme } from './hooks/useTheme';

interface LoginResponse {
  access_token?: string;
  refresh_token?: string;
  requires_totp?: boolean;
  user_id?: string;
  error?: string;
}

export default function Login() {
  const theme = useTheme<{ brand_name: string; primary_color: string; logo_url: string }>(
    '/api/v1/public/theme'
  );
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [totp, setTotp] = useState('');
  const [pendingUserId, setPendingUserId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const body: Record<string, string> = { email, password };
      if (totp) body.totp_code = totp;
      if (pendingUserId) body.user_id = pendingUserId;
      const r = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data: LoginResponse = await r.json();
      if (!r.ok) {
        setError(data.error || 'login_failed');
        return;
      }
      if (data.requires_totp) {
        setPendingUserId(data.user_id || null);
        return;
      }
      if (data.access_token) {
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token || '');
        window.location.href = '/app';
      }
    } catch (e) {
      setError('network_error');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main
      style={{
        minHeight: '100vh',
        background: '#0b1020',
        color: '#e8ecf3',
        display: 'grid',
        placeItems: 'center',
        fontFamily: 'system-ui, sans-serif',
        padding: 24,
      }}
    >
      <form
        onSubmit={submit}
        style={{
          width: 380,
          background: 'rgba(255,255,255,0.04)',
          border: '1px solid rgba(255,255,255,0.08)',
          borderRadius: 12,
          padding: 32,
        }}
      >
        <h1 style={{ color: theme?.primary_color ?? '#1a73e8', margin: 0 }}>
          {theme?.brand_name ?? 'OrvixPanel'}
        </h1>
        <p style={{ color: '#99a3b3', marginTop: 4 }}>Sign in to your account</p>

        {error && (
          <div
            role="alert"
            style={{
              background: 'rgba(255,80,80,0.12)',
              border: '1px solid rgba(255,80,80,0.3)',
              padding: 10,
              borderRadius: 6,
              marginTop: 12,
              color: '#ffb3b3',
              fontSize: 14,
            }}
          >
            {error}
          </div>
        )}

        {!pendingUserId && (
          <>
            <label style={labelStyle}>Email</label>
            <input
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              style={inputStyle}
            />
            <label style={labelStyle}>Password</label>
            <input
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              style={inputStyle}
            />
          </>
        )}
        {pendingUserId && (
          <>
            <label style={labelStyle}>2FA code</label>
            <input
              type="text"
              required
              inputMode="numeric"
              pattern="[0-9]{6}"
              maxLength={6}
              autoComplete="one-time-code"
              value={totp}
              onChange={(e) => setTotp(e.target.value)}
              style={inputStyle}
              placeholder="123 456"
            />
          </>
        )}
        <button
          type="submit"
          disabled={loading}
          style={{
            width: '100%',
            marginTop: 16,
            padding: 10,
            background: theme?.primary_color ?? '#1a73e8',
            color: '#fff',
            border: 'none',
            borderRadius: 6,
            cursor: loading ? 'wait' : 'pointer',
            fontSize: 15,
          }}
        >
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </main>
  );
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  marginTop: 12,
  marginBottom: 4,
  fontSize: 13,
  color: '#bcc6d4',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  background: 'rgba(255,255,255,0.06)',
  color: '#e8ecf3',
  border: '1px solid rgba(255,255,255,0.12)',
  borderRadius: 6,
  fontSize: 14,
  boxSizing: 'border-box',
};
