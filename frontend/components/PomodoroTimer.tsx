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
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Focus Timer</h3>
        <span className={`text-xs px-2 py-0.5 rounded font-medium ${
          phase === 'work' ? 'bg-red-900 text-red-300' : 'bg-blue-900 text-blue-300'
        }`}>
          {phase === 'work' ? 'Work' : 'Break'}
        </span>
      </div>

      {/* Progress ring via border trick */}
      <div className="text-5xl font-mono font-bold text-center text-white mb-1">
        {mins}:{secs}
      </div>
      <div className="w-full bg-gray-800 rounded-full h-1 mb-4">
        <div
          className={`h-1 rounded-full transition-all ${phase === 'work' ? 'bg-red-500' : 'bg-blue-500'}`}
          style={{ width: `${pct}%` }}
        />
      </div>

      <div className="flex gap-2">
        <button
          onClick={toggle}
          className={`flex-1 py-2 rounded text-sm font-medium transition-colors ${
            running
              ? 'bg-yellow-600 hover:bg-yellow-700 text-white'
              : 'bg-red-600 hover:bg-red-700 text-white'
          }`}
        >
          {running ? 'Pause' : 'Start'}
        </button>
        <button onClick={reset} className="px-3 py-2 rounded text-sm bg-gray-700 hover:bg-gray-600 text-gray-300 transition-colors">
          Reset
        </button>
        <button onClick={skip} className="px-3 py-2 rounded text-sm bg-gray-700 hover:bg-gray-600 text-gray-300 transition-colors">
          Skip
        </button>
      </div>
    </div>
  );
}
