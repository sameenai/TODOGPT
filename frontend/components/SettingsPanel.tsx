'use client';

import { useState, useEffect } from 'react';
import type { ConfigResponse } from '@/lib/types';
import { fetchConfig, saveConfig, fetchAuthStatus, googleAuthURL, disconnectGoogle } from '@/lib/api';
import type { AuthStatus } from '@/lib/api';

interface Props {
  open: boolean;
  onClose: () => void;
}

type Section = 'google' | 'weather' | 'news' | 'calendar' | 'slack' | 'email' | 'github' | 'jira' | 'notion' | 'ai' | 'server' | 'pomodoro';

const SECTIONS: { key: Section; label: string; icon: string }[] = [
  { key: 'google', label: 'Google', icon: '🔗' },
  { key: 'weather', label: 'Weather', icon: '🌤' },
  { key: 'news', label: 'News', icon: '📰' },
  { key: 'calendar', label: 'Calendar', icon: '📅' },
  { key: 'slack', label: 'Slack', icon: '💬' },
  { key: 'email', label: 'Email', icon: '📧' },
  { key: 'github', label: 'GitHub', icon: '🐙' },
  { key: 'jira', label: 'Jira', icon: '🎯' },
  { key: 'notion', label: 'Notion', icon: '📝' },
  { key: 'ai', label: 'AI Assistant', icon: '🤖' },
  { key: 'server', label: 'Server', icon: '⚙️' },
  { key: 'pomodoro', label: 'Pomodoro', icon: '🍅' },
];

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 ${
        checked ? 'bg-cyan-600' : 'bg-gray-600'
      }`}
    >
      <span
        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ${
          checked ? 'translate-x-4' : 'translate-x-0'
        }`}
      />
    </button>
  );
}

function Field({
  label, value, onChange, type = 'text', placeholder,
}: {
  label: string;
  value: string | number;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <label className="text-xs text-gray-400 w-28 flex-shrink-0">{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        spellCheck={false}
        autoComplete="off"
        className="flex-1 bg-gray-800 text-gray-100 placeholder-gray-600 text-xs px-2.5 py-1.5 rounded border border-gray-700 focus:outline-none focus:border-cyan-500"
      />
    </div>
  );
}

export function SettingsPanel({ open, onClose }: Props) {
  const [cfg, setCfg] = useState<ConfigResponse | null>(null);
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState('');
  const [activeSection, setActiveSection] = useState<Section>('google');
  const [disconnecting, setDisconnecting] = useState(false);

  useEffect(() => {
    if (open && !cfg) {
      setLoading(true);
      Promise.all([fetchConfig(), fetchAuthStatus()])
        .then(([c, a]) => { setCfg(c); setAuthStatus(a); })
        .catch(() => setSaveMsg('Failed to load config'))
        .finally(() => setLoading(false));
    }
  }, [open, cfg]);

  async function handleDisconnectGoogle() {
    setDisconnecting(true);
    try {
      await disconnectGoogle();
      setAuthStatus(prev => prev ? { ...prev, google: { ...prev.google, connected: false } } : prev);
    } finally {
      setDisconnecting(false);
    }
  }

  async function handleSave() {
    if (!cfg) return;
    setSaving(true);
    setSaveMsg('');
    try {
      const result = await saveConfig(cfg);
      setSaveMsg(result.message);
    } catch {
      setSaveMsg('Save failed — check server logs');
    } finally {
      setSaving(false);
    }
  }

  function set<K extends keyof ConfigResponse>(
    section: K,
    field: keyof ConfigResponse[K],
    value: unknown,
  ) {
    setCfg(prev => {
      if (!prev) return prev;
      return {
        ...prev,
        [section]: { ...prev[section], [field]: value },
      };
    });
  }

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div className="relative ml-auto w-full max-w-lg h-full bg-gray-950 border-l border-gray-800 flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
          <h2 className="text-sm font-semibold text-white">Settings</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white text-lg leading-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 rounded"
            aria-label="Close settings"
          >
            ✕
          </button>
        </div>

        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar nav */}
          <nav className="w-36 flex-shrink-0 border-r border-gray-800 overflow-y-auto py-2">
            {SECTIONS.map(s => (
              <button
                key={s.key}
                onClick={() => setActiveSection(s.key)}
                className={`w-full flex items-center gap-2 px-3 py-2 text-xs text-left transition-colors ${
                  activeSection === s.key
                    ? 'bg-gray-800 text-white'
                    : 'text-gray-400 hover:text-gray-200 hover:bg-gray-900'
                }`}
              >
                <span>{s.icon}</span>
                <span>{s.label}</span>
              </button>
            ))}
          </nav>

          {/* Content */}
          <div className="flex-1 overflow-y-auto p-5">
            {loading && <p className="text-sm text-gray-500">Loading…</p>}
            {!loading && cfg && (
              <div className="space-y-4">
                {activeSection === 'google' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Google Account</span>
                      {authStatus?.google.connected && (
                        <span className="text-xs bg-emerald-900/50 text-emerald-400 border border-emerald-700 px-2 py-0.5 rounded-full">Connected</span>
                      )}
                    </div>

                    {authStatus?.google.connected ? (
                      <div className="space-y-3">
                        <p className="text-xs text-gray-400">
                          Your Google account is connected. Calendar and Gmail will use the Google APIs.
                        </p>
                        <button
                          onClick={handleDisconnectGoogle}
                          disabled={disconnecting}
                          className="w-full py-2 px-4 bg-red-900/40 hover:bg-red-900/60 text-red-400 hover:text-red-300 text-sm rounded border border-red-800 transition-colors disabled:opacity-50"
                        >
                          {disconnecting ? 'Disconnecting…' : 'Disconnect Google Account'}
                        </button>
                      </div>
                    ) : authStatus?.google.configured ? (
                      <div className="space-y-3">
                        <p className="text-xs text-gray-400">
                          OAuth credentials are configured. Click below to sign in with your Google account.
                        </p>
                        <a
                          href={googleAuthURL()}
                          className="flex items-center justify-center gap-2 w-full py-2 px-4 bg-white hover:bg-gray-100 text-gray-900 text-sm font-medium rounded transition-colors"
                        >
                          <svg className="w-4 h-4" viewBox="0 0 24 24">
                            <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                            <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                            <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                            <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                          </svg>
                          Sign in with Google
                        </a>
                      </div>
                    ) : (
                      <div className="space-y-3">
                        <p className="text-xs text-gray-400">
                          Set your Google OAuth2 credentials to enable single sign-on for Calendar and Gmail.
                        </p>
                        <div className="rounded-lg border border-amber-800/50 bg-amber-950/20 p-3 space-y-2">
                          <p className="text-xs text-amber-400 font-medium">Setup steps:</p>
                          <ol className="text-xs text-gray-400 space-y-1 list-decimal list-inside">
                            <li>Go to Google Cloud Console → APIs &amp; Services → Credentials</li>
                            <li>Create an OAuth 2.0 Client ID (Web application)</li>
                            <li>Add <code className="bg-gray-800 px-1 rounded text-gray-300">http://localhost:8080/api/auth/google/callback</code> as an authorized redirect URI</li>
                            <li>Enable the Calendar API and Gmail API in your project</li>
                            <li>Paste your Client ID and Secret below, then save</li>
                          </ol>
                        </div>
                        <Field label="Client ID" value={cfg?.google.client_id ?? ''} onChange={v => set('google', 'client_id', v)} placeholder="....apps.googleusercontent.com" />
                        <Field label="Client Secret" value={cfg?.google.client_secret ?? ''} onChange={v => set('google', 'client_secret', v)} placeholder="GOCSPX-..." />
                        <p className="text-xs text-gray-500">After saving, restart the server then click "Sign in with Google".</p>
                      </div>
                    )}
                  </>
                )}

                {activeSection === 'weather' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Weather</span>
                      <Toggle checked={cfg.weather.enabled} onChange={v => set('weather', 'enabled', v)} />
                    </div>
                    <Field label="City" value={cfg.weather.city} onChange={v => set('weather', 'city', v)} />
                    <Field label="Country" value={cfg.weather.country} onChange={v => set('weather', 'country', v)} placeholder="US" />
                    <Field label="Units" value={cfg.weather.units} onChange={v => set('weather', 'units', v)} placeholder="imperial / metric" />
                    <Field label="API Key" value={cfg.weather.api_key} onChange={v => set('weather', 'api_key', v)} placeholder="Open-Meteo is free — no key needed" />
                  </>
                )}

                {activeSection === 'news' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">News</span>
                      <Toggle checked={cfg.news.enabled} onChange={v => set('news', 'enabled', v)} />
                    </div>
                    <Field label="Max Items" value={cfg.news.max_items} type="number" onChange={v => set('news', 'max_items', parseInt(v, 10) || 10)} />
                    <Field label="API Key" value={cfg.news.api_key} onChange={v => set('news', 'api_key', v)} placeholder="NewsAPI key (optional)" />
                  </>
                )}

                {activeSection === 'calendar' && (
                  <>
                    <span className="text-sm font-medium text-gray-300">Calendar</span>
                    {authStatus?.google.connected ? (
                      <p className="text-xs text-emerald-400">Using Google Calendar API via OAuth ✓</p>
                    ) : (
                      <>
                        <p className="text-xs text-gray-500">
                          Connect Google in the <button className="text-cyan-400 hover:text-cyan-300 underline underline-offset-2" onClick={() => setActiveSection('google')}>Google</button> section for live calendar sync, or paste a private iCal URL below.
                        </p>
                        <Field label="iCal URL" value={cfg.google.ical_url} onChange={v => set('google', 'ical_url', v)} placeholder="https://calendar.google.com/calendar/ical/..." />
                      </>
                    )}
                  </>
                )}

                {activeSection === 'slack' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Slack</span>
                      <Toggle checked={cfg.slack.enabled} onChange={v => set('slack', 'enabled', v)} />
                    </div>
                    <Field label="Bot Token" value={cfg.slack.bot_token} onChange={v => set('slack', 'bot_token', v)} placeholder="xoxb-..." />
                  </>
                )}

                {activeSection === 'email' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Email</span>
                      <Toggle checked={cfg.email.enabled} onChange={v => set('email', 'enabled', v)} />
                    </div>
                    {authStatus?.google.connected ? (
                      <p className="text-xs text-emerald-400">Using Gmail API via OAuth ✓</p>
                    ) : (
                      <>
                        <p className="text-xs text-gray-500">
                          Connect Google in the <button className="text-cyan-400 hover:text-cyan-300 underline underline-offset-2" onClick={() => setActiveSection('google')}>Google</button> section for Gmail, or configure IMAP below.
                        </p>
                        <p className="text-xs text-gray-600 font-medium mt-2">IMAP fallback</p>
                        <Field label="IMAP Server" value={cfg.email.imap_server} onChange={v => set('email', 'imap_server', v)} placeholder="imap.gmail.com" />
                        <Field label="Port" value={cfg.email.imap_port} type="number" onChange={v => set('email', 'imap_port', parseInt(v, 10) || 993)} />
                        <Field label="Username" value={cfg.email.username} onChange={v => set('email', 'username', v)} placeholder="you@example.com" />
                        <Field label="Password" value={cfg.email.password} onChange={v => set('email', 'password', v)} placeholder="App password" />
                      </>
                    )}
                  </>
                )}

                {activeSection === 'github' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">GitHub</span>
                      <Toggle checked={cfg.github.enabled} onChange={v => set('github', 'enabled', v)} />
                    </div>
                    <Field label="Token" value={cfg.github.token} onChange={v => set('github', 'token', v)} placeholder="ghp_..." />
                    <p className="text-xs text-gray-500">Needs <code className="bg-gray-800 px-1 rounded">notifications</code> scope.</p>
                  </>
                )}

                {activeSection === 'jira' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Jira</span>
                      <Toggle checked={cfg.jira.enabled} onChange={v => set('jira', 'enabled', v)} />
                    </div>
                    <Field label="Base URL" value={cfg.jira.base_url} onChange={v => set('jira', 'base_url', v)} placeholder="https://yourorg.atlassian.net" />
                    <Field label="Email" value={cfg.jira.email} onChange={v => set('jira', 'email', v)} placeholder="you@yourorg.com" />
                    <Field label="API Token" value={cfg.jira.api_token} onChange={v => set('jira', 'api_token', v)} placeholder="Atlassian API token" />
                    <Field label="Project Key" value={cfg.jira.project_key} onChange={v => set('jira', 'project_key', v)} placeholder="PROJ" />
                  </>
                )}

                {activeSection === 'notion' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Notion</span>
                      <Toggle checked={cfg.notion.enabled} onChange={v => set('notion', 'enabled', v)} />
                    </div>
                    <Field label="Token" value={cfg.notion.token} onChange={v => set('notion', 'token', v)} placeholder="secret_..." />
                    <Field label="Database ID" value={cfg.notion.database_id} onChange={v => set('notion', 'database_id', v)} placeholder="32-char database ID" />
                  </>
                )}

                {activeSection === 'ai' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">AI Assistant</span>
                      <Toggle checked={cfg.ai.enabled} onChange={v => set('ai', 'enabled', v)} />
                    </div>
                    <Field label="API Key" value={cfg.ai.api_key} onChange={v => set('ai', 'api_key', v)} placeholder="sk-ant-..." />
                    <Field label="Model" value={cfg.ai.model} onChange={v => set('ai', 'model', v)} placeholder="claude-sonnet-4-6" />
                    <p className="text-xs text-gray-500">Powers morning summary, daily review, and time blocking.</p>
                  </>
                )}

                {activeSection === 'server' && (
                  <>
                    <span className="text-sm font-medium text-gray-300">Server</span>
                    <Field label="Port" value={cfg.server.port} type="number" onChange={v => set('server', 'port', parseInt(v, 10) || 8080)} />
                    <Field label="Host" value={cfg.server.host} onChange={v => set('server', 'host', v)} />
                    <Field label="Poll (sec)" value={cfg.server.poll_interval_seconds} type="number" onChange={v => set('server', 'poll_interval_seconds', parseInt(v, 10) || 30)} />
                  </>
                )}

                {activeSection === 'pomodoro' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Pomodoro Timer</span>
                      <Toggle checked={cfg.pomodoro.enabled} onChange={v => set('pomodoro', 'enabled', v)} />
                    </div>
                    <Field label="Work (min)" value={cfg.pomodoro.work_minutes} type="number" onChange={v => set('pomodoro', 'work_minutes', parseInt(v, 10) || 25)} />
                    <Field label="Break (min)" value={cfg.pomodoro.break_minutes} type="number" onChange={v => set('pomodoro', 'break_minutes', parseInt(v, 10) || 5)} />
                  </>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-3 px-5 py-4 border-t border-gray-800">
          {saveMsg ? (
            <p className="text-xs text-gray-400 flex-1">{saveMsg}</p>
          ) : (
            <p className="text-xs text-gray-600 flex-1">Changes are saved to ~/.daily-briefing/config.json</p>
          )}
          <button
            onClick={handleSave}
            disabled={saving || !cfg}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded disabled:opacity-50 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}
