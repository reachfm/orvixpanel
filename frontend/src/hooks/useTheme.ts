import { useEffect, useState } from 'react';

/**
 * Loads a JSON resource once and returns the parsed value.
 * Phase 7 will swap this for a Zustand-backed cached theme.
 */
export function useTheme<T = unknown>(url: string): T | null {
  const [data, setData] = useState<T | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch(url)
      .then((r) => (r.ok ? r.json() : Promise.reject(r.statusText)))
      .then((d) => {
        if (!cancelled) setData(d as T);
      })
      .catch(() => {
        if (!cancelled) setData(null);
      });
    return () => {
      cancelled = true;
    };
  }, [url]);

  return data;
}
