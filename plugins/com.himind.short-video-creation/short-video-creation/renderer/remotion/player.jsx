import React, {useMemo} from 'react';
import {createRoot} from 'react-dom/client';
import {Player} from '@remotion/player';
import {LeaderboardTech, MapStory} from './index.jsx';

const clamp = (value, min, max) => Math.max(min, Math.min(max, Number(value) || 0));

function resolveSpec() {
  if (typeof window !== 'undefined' && window.__PREVIEW_SPEC__) {
    return window.__PREVIEW_SPEC__;
  }
  try {
    const raw = new URLSearchParams(window.location.search).get('p');
    return raw ? JSON.parse(decodeURIComponent(raw)) : null;
  } catch (err) {
    return null;
  }
}

function PlayerApp() {
  const spec = useMemo(resolveSpec, []);
  const canvas = (spec && spec.canvas) || {};
  const fps = clamp(canvas.fps || 30, 1, 60);
  const duration = Math.max(1, Math.round(clamp((spec && spec.duration_seconds) || 12, 1, 300) * fps));
  const component = (spec && spec.composition === 'GDPMapStory') || (spec && spec.composition === 'LeaderboardMapTrend')
    ? MapStory
    : LeaderboardTech;
  return (
    <div style={{width: '100vw', height: '100vh', background: '#0a0f1f'}}>
      <Player
        component={component}
        inputProps={spec || {}}
        durationInFrames={duration}
        compositionWidth={clamp(canvas.width || 1080, 240, 4096)}
        compositionHeight={clamp(canvas.height || 1920, 240, 4096)}
        fps={fps}
        style={{width: '100%', height: '100%'}}
        controls
        loop
        autoPlay
      />
    </div>
  );
}

createRoot(document.getElementById('root')).render(<PlayerApp />);
