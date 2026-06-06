import React from 'react';
import ReactDOM from 'react-dom/client';
import Login from './Login';
import Dashboard from './Dashboard';
import FileManager from './FileManager';
import './styles/index.css';

function Router() {
  const path = window.location.pathname;
  if (path === '/login' || path === '/') {
    return <Login />;
  }
  if (path === '/app' || path === '/dashboard') {
    return <Dashboard />;
  }
  if (path === '/files') {
    const params = new URLSearchParams(window.location.search);
    return <FileManager accountId={params.get('account') || ''} />;
  }
  return (
    <div style={{ padding: 40, color: '#e8ecf3', background: '#0b1020', minHeight: '100vh' }}>
      <h1>404</h1>
      <p>Unknown path: <code>{path}</code></p>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <Router />
  </React.StrictMode>,
);
