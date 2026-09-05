import React from 'react';
import {AbsoluteFill, Composition, interpolate, registerRoot, spring, useCurrentFrame, useVideoConfig} from 'remotion';

const clamp = (value, min, max) => Math.max(min, Math.min(max, Number(value) || 0));

const LeaderboardTech = (props) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const data = props.data || {};
  const items = (props.items || data.items || []).slice(0, 8);
  const palette = props.style_id === 'minimal-white'
    ? {bg: '#f7f9fc', panel: '#e7edf5', accent: '#1769aa', text: '#10233d', muted: '#60748e'}
    : {bg: '#09111f', panel: '#14243a', accent: '#4fc3f7', text: '#eef5ff', muted: '#9eb2cc'};
  const intro = spring({frame, fps, config: {damping: 18, stiffness: 90}});
  return <AbsoluteFill style={{backgroundColor: palette.bg, color: palette.text, fontFamily: 'Microsoft YaHei, Noto Sans CJK SC, sans-serif', padding: '100px 72px 86px', boxSizing: 'border-box'}}>
    <div style={{position: 'absolute', inset: 34, border: '3px solid ' + palette.accent, opacity: .45}} />
    <div style={{fontSize: 25, color: palette.accent, fontWeight: 700, opacity: intro, transform: 'translateY(' + (1 - intro) * 28 + 'px)'}}>DATA RANKING · HIMIND</div>
    <div style={{fontSize: 72, lineHeight: 1.12, fontWeight: 800, marginTop: 34, opacity: intro, transform: 'translateY(' + (1 - intro) * 42 + 'px)'}}>{props.title || '数据排行榜'}</div>
    <div style={{fontSize: 31, color: palette.muted, marginTop: 24, opacity: intro}}>{props.subtitle || ''}</div>
    <div style={{display: 'flex', flexDirection: 'column', gap: 34, marginTop: 116}}>
      {items.map((item, index) => {
        const reveal = spring({frame: frame - 22 - index * 10, fps, config: {damping: 17, stiffness: 100}});
        const value = clamp(item.value, 0, 100);
        const width = interpolate(reveal, [0, 1], [0, value], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
        return <div key={index} style={{opacity: reveal, transform: 'translateX(' + (1 - reveal) * 55 + 'px)'}}>
          <div style={{display: 'grid', gridTemplateColumns: '58px 1fr 86px', alignItems: 'end', gap: 18, fontSize: 29}}>
            <strong style={{color: palette.accent, fontSize: 34}}>{String(index + 1).padStart(2, '0')}</strong>
            <span style={{fontWeight: 700}}>{item.name || '未命名'}</span>
            <strong style={{textAlign: 'right', color: palette.accent}}>{value}</strong>
          </div>
          <div style={{height: 19, background: palette.panel, marginTop: 14, overflow: 'hidden'}}><div style={{height: '100%', width: width + '%', background: palette.accent, boxShadow: '0 0 25px ' + palette.accent}} /></div>
        </div>;
      })}
    </div>
    <div style={{marginTop: 'auto', display: 'flex', justifyContent: 'space-between', color: palette.muted, fontSize: 21}}><span>Template · LeaderboardTech</span><span>Powered by Remotion</span></div>
  </AbsoluteFill>;
};

const MapStory = (props) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const data = props.data || {};
  const items = (props.items || data.items || data.legend || []).slice(0, 6);
  const mapData = data.map_data || {};
  const intro = spring({frame, fps, config: {damping: 19, stiffness: 85}});
  const accent = '#4fc3f7';
  return <AbsoluteFill style={{backgroundColor: '#09111f', color: '#eef5ff', fontFamily: 'Microsoft YaHei, Noto Sans CJK SC, sans-serif', padding: '92px 68px 82px', boxSizing: 'border-box'}}>
    <div style={{position: 'absolute', inset: 34, border: '3px solid ' + accent, opacity: .38}} />
    <div style={{color: accent, fontSize: 24, fontWeight: 700, opacity: intro}}>REGIONAL DATA · HIMIND</div>
    <div style={{fontSize: 66, lineHeight: 1.14, fontWeight: 800, marginTop: 28, opacity: intro, transform: 'translateY(' + (1 - intro) * 34 + 'px)'}}>{props.title || '区域数据故事'}</div>
    <div style={{color: '#9eb2cc', fontSize: 29, marginTop: 18}}>{props.subtitle || mapData.region || ''}</div>
    <div style={{position: 'relative', height: 620, marginTop: 84, border: '1px solid #29435f', backgroundColor: '#0d1b2d', overflow: 'hidden'}}>
      {[0,1,2,3,4].map((line) => <div key={'h'+line} style={{position:'absolute',left:0,right:0,top:(line+1)*100,borderTop:'1px solid #17324d'}} />)}
      {[0,1,2,3].map((line) => <div key={'v'+line} style={{position:'absolute',top:0,bottom:0,left:(line+1)*110,borderLeft:'1px solid #17324d'}} />)}
      {(mapData.points || items).slice(0, 8).map((point, index) => {
        const reveal = spring({frame: frame - 16 - index * 7, fps, config: {damping: 16, stiffness: 100}});
        const x = 70 + ((index * 137) % 390);
        const y = 85 + ((index * 91) % 430);
        return <div key={index} style={{position:'absolute',left:x,top:y,opacity:reveal,transform:'scale(' + reveal + ')'}}>
          <div style={{width:18,height:18,backgroundColor:accent,boxShadow:'0 0 24px '+accent}} />
          <div style={{marginTop:8,fontSize:20,whiteSpace:'nowrap'}}>{point.name || point.label || ('区域 '+(index+1))}</div>
        </div>;
      })}
      <div style={{position:'absolute',left:42,bottom:34,color:'#718aa6',fontSize:18}}>地图、Three.js 与图表作为 Remotion 模板内部能力组合</div>
    </div>
    <div style={{display:'grid',gridTemplateColumns:'repeat(3,1fr)',gap:18,marginTop:36}}>
      {items.slice(0,3).map((item,index) => <div key={index} style={{border:'1px solid #29435f',padding:'22px',backgroundColor:'#102039'}}>
        <div style={{fontSize:20,color:'#9eb2cc'}}>{item.label || item.name || ('指标 '+(index+1))}</div><div style={{fontSize:42,fontWeight:800,color:accent,marginTop:10}}>{item.value ?? '--'}</div>
      </div>)}
    </div>
    <div style={{marginTop:'auto',display:'flex',justifyContent:'space-between',color:'#718aa6',fontSize:20}}><span>Template · Remotion Map Story</span><span>Powered by Remotion</span></div>
  </AbsoluteFill>;
};

const metadata = ({props}) => {
  const canvas = props.canvas || {};
  const fps = clamp(canvas.fps || 30, 1, 60);
  return {width: clamp(canvas.width || 1080, 240, 4096), height: clamp(canvas.height || 1920, 240, 4096), fps, durationInFrames: Math.max(1, Math.round(clamp(props.duration_seconds || 12, 1, 300) * fps))};
};

const Root = () => <>
  <Composition id="LeaderboardTech" component={LeaderboardTech} width={1080} height={1920} fps={30} durationInFrames={360} defaultProps={{}} calculateMetadata={metadata} />
  <Composition id="LeaderboardMapTrend" component={MapStory} width={1080} height={1920} fps={30} durationInFrames={450} defaultProps={{}} calculateMetadata={metadata} />
  <Composition id="GDPMapStory" component={MapStory} width={1080} height={1920} fps={30} durationInFrames={540} defaultProps={{}} calculateMetadata={metadata} />
</>;

registerRoot(Root);
