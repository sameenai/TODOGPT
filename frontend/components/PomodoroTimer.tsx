'use client';

import { useState, useEffect } from 'react';

type Phase = 'work' | 'break';

const WORK_SECS = 25 * 60;
const BREAK_SECS = 5 * 60;

export function PomodoroTimer() {
  const [phase, setPhase] = useState<Phase>('work');
  const [timeLeft, setTimeLeft] = useState(WORK_SECS);
  const [running, setRunning] = useState(false);

  useEffect(() => {
    if (!running) return;
    const id = setInterval(() => {
      setTimeLeft(t => {
        if (t > 1) return t - 1;
        // Phase complete
        const next: Phase = phase === 'work' ? 'break' : 'work';
        if (typeof Notification !== 'undefined' && Notification.permission === 'granted') {
          new Notification(next === 'work' ? 'Back to work!' : 'Break time!', {
            body: next === 'work' ? 'Start your focus session.' : 'Take a 5-minute break.',
          });
        }
        setPhase(next);
        return next === 'work' ? WORK_SECS : BREAK_SECS;
      });
    }, 1000);
    return () => clearInterval(id);
  }, [running, phase]);

  function toggle() {
    if (!running && typeof Notification !== 'undefined' && Notification.permission === 'default') {
      Notification.requestPermission();
    }
    setRunning(r => !r);
  }

  function reset() {
    setRunning(false);
    setTimeLeft(phase === 'work' ? WORK_SECS : BREAK_SECS);
  }

  function skip() {
    setRunning(false);
    const next: Phase = phase === 'work' ? 'break' : 'work';
    setPhase(next);
    setTimeLeft(next === 'work' ? WORK_SECS : BREAK_SECS);
  }

  const mins = String(Math.floor(timeLeft / 60)).padStart(2, '0');
  const secs = String(timeLeft % 60).padStart(2, '0');
  const pct = Math.round((1 - timeLeft / (phase === 'work' ? WORK_SECS : BREAK_SECS)) * 100);

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-4">
      <div className="panel-header -mx-4 -mt-4 mb-4 px-4">
        <h3 className="section-title">Focus Timer</h3>
        <span className={`text-xs px-2 py-0.5 rounded-full font-semibold ${
          phase === 'work'
            ? 'bg-rose-950/60 text-rose-300 border border-rose-800/50'
            : 'bg-sky-950/60 text-sky-300 border border-sky-800/50'
        }`}>
          {phase === 'work' ? 'Work' : 'Break'}
        </span>
      </div>

      <div className="text-5xl font-mono font-bold text-center text-white mb-2 tabular-nums tracking-tight">
        {mins}:{secs}
      </div>
      <div className="w-full bg-gray-800 rounded-full h-0.5 mb-4 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-1000 ${
            phase === 'work' ? 'bg-rose-500' : 'bg-sky-500'
          }`}
          style={{ width: `${pct}%` }}
        />
      </div>

      <div className="flex gap-2">
        <button
          onClick={toggle}
          className={`flex-1 py-2 rounded-lg text-sm font-semibold transition-colors ${
            running
              ? 'bg-amber-600/20 hover:bg-amber-600/30 text-amber-300 border border-amber-700/50'
              : 'bg-rose-600 hover:bg-rose-700 text-white'
          }`}
        >
          {running ? 'Pause' : 'Start'}
        </button>
        <button
          onClick={reset}
          className="px-3 py-2 rounded-lg text-sm bg-gray-800 hover:bg-gray-700 text-gray-400 hover:text-gray-200 transition-colors"
        >
          Reset
        </button>
        <button
          onClick={skip}
          className="px-3 py-2 rounded-lg text-sm bg-gray-800 hover:bg-gray-700 text-gray-400 hover:text-gray-200 transition-colors"
        >
          Skip
        </button>
      </div>
    </div>
  );
}
