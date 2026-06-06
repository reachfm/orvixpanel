import { useEffect, useState } from 'react';

interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  modified_at: string;
}

function auth() {
  return { Authorization: `Bearer ${localStorage.getItem('access_token')}` };
}

export default function FileManager({ accountId }: { accountId: string }) {
  const [path, setPath] = useState('/');
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const params = new URLSearchParams({ path, account: accountId });
    fetch(`/api/v1/files?${params}`, { headers: auth() })
      .then(async (r) => {
        if (!r.ok) throw new Error((await r.json()).error || 'list_failed');
        return r.json();
      })
      .then((d) => setEntries(d.entries || []))
      .catch((e) => setError(String(e)));
  }, [path, accountId]);

  return (
    <div style={{ padding: 24, color: '#e8ecf3', background: '#0b1020', minHeight: '100vh' }}>
      <h2>File Manager</h2>
      <div style={{ fontSize: 13, color: '#99a3b3', marginBottom: 12 }}>
        <code>{path}</code>
        {path !== '/' && (
          <button
            onClick={() => setPath(path.split('/').slice(0, -1).join('/') || '/')}
            style={{ marginLeft: 12, padding: '2px 8px' }}
          >
            Up
          </button>
        )}
      </div>
      {error && <div style={{ color: '#ff7373' }}>{error}</div>}
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.1)', textAlign: 'left' }}>
            <th style={th}>Name</th>
            <th style={th}>Size</th>
            <th style={th}>Modified</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <tr
              key={e.path}
              onClick={() => e.is_dir && setPath(e.path)}
              style={{ borderBottom: '1px solid rgba(255,255,255,0.04)', cursor: e.is_dir ? 'pointer' : 'default' }}
            >
              <td style={td}>
                {e.is_dir ? '📁' : '📄'} {e.name}
              </td>
              <td style={td}>{e.is_dir ? '—' : formatSize(e.size)}</td>
              <td style={td}>{new Date(e.modified_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

const th: React.CSSProperties = { padding: 8, fontSize: 13, color: '#bcc6d4' };
const td: React.CSSProperties = { padding: 8, fontSize: 13 };
