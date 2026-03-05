'use client';

import { useState, useEffect } from 'react';
import type { ConfigResponse } from '@/lib/types';
import { fetchConfig, saveConfig } from '@/lib/api';

interface Props {
  open: boolean;
  onClose: () => void;
}

type Section = 'weather' | 'news' | 'calendar' | 'slack' | 'email' | 'github' | 'jira' | 'notion' | 'ai' | 'server' | 'pomodoro';

const SECTIONS: { key: Section; label: string; icon: string }[] = [
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
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState('');
  const [activeSection, setActiveSection] = useState<Section>('weather');

  useEffect(() => {
    if (open && !cfg) {
      setLoading(true);
      fetchConfig()
        .then(setCfg)
        .catch(() => setSaveMsg('Failed to load config'))
        .finally(() => setLoading(false));
    }
  }, [open, cfg]);

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
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Calendar</span>
                      <Toggle checked={cfg.google.calendar_enabled} onChange={v => set('google', 'calendar_enabled', v)} />
                    </div>
                    <p className="text-xs text-gray-500">Paste your private iCal URL from Google Calendar, iCloud, or Outlook.</p>
                    <Field label="iCal URL" value={cfg.google.ical_url} onChange={v => set('google', 'ical_url', v)} placeholder="https://calendar.google.com/calendar/ical/..." />
                  </>
                )}

                {activeSection === 'slack' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Slack</span>
                      <Toggle checked={cfg.slack.enabled} onChange={v => set('slack', 'enabled', v)} />
                    </div>
                    <Field label="Bot Token" value={cfg.slack.bot_token} onChange={v => set('slack', 'bot_token', v)} placeholder="xoxb-..." />
                    <Field label="App Token" value={cfg.slack.app_token} onChange={v => set('slack', 'app_token', v)} placeholder="xapp-..." />
                  </>
                )}

                {activeSection === 'email' && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-300">Email (IMAP)</span>
                      <Toggle checked={cfg.email.enabled} onChange={v => set('email', 'enabled', v)} />
                    </div>
                    <Field label="IMAP Server" value={cfg.email.imap_server} onChange={v => set('email', 'imap_server', v)} placeholder="imap.gmail.com" />
                    <Field label="Port" value={cfg.email.imap_port} type="number" onChange={v => set('email', 'imap_port', parseInt(v, 10) || 993)} />
                    <Field label="Username" value={cfg.email.username} onChange={v => set('email', 'username', v)} placeholder="you@example.com" />
                    <Field label="Password" value={cfg.email.password} onChange={v => set('email', 'password', v)} placeholder="App password" />
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
