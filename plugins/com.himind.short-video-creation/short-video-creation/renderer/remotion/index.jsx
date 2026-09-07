import React from 'react';
import {AbsoluteFill, Composition, interpolate, registerRoot, spring, useCurrentFrame, useVideoConfig} from 'remotion';

const clamp = (value, min, max) => Math.max(min, Math.min(max, Number(value) || 0));

const LeaderboardTech = (props) => {
  const frame = useCurrentFrame();
  const {fps, durationInFrames} = useVideoConfig();
  const canvas = props.canvas || {};
  const width = Number(canvas.width) || 2160;
  const height = Number(canvas.height) || 3840;
  const scale = width / 2160;
  const px = (value) => Math.max(1, Math.round(value * scale));
  const data = props.data || {};
  const items = (props.items || data.items || []).slice(0, 10).map((item, index) => ({
    ...item,
    name: item.name || item.label || `项目 ${index + 1}`,
    value: Number(item.value) || 0,
    subValue: item.sub_value || item.subValue || item.change || '',
    icon: item.icon || '',
  }));
  const palette = props.style_id === 'minimal-white'
    ? {bg: '#f7f9fc', bg2: '#e9eef7', panel: 'rgba(16,35,61,.06)', accent: '#1769aa', accent2: '#35a7d8', text: '#10233d', muted: '#60748e', line: 'rgba(16,35,61,.12)', footer: 'rgba(16,35,61,.06)'}
    : {bg: '#080c1a', bg2: '#131b33', panel: 'rgba(255,255,255,.06)', accent: '#ff6b6b', accent2: '#4fc3f7', text: '#ffffff', muted: 'rgba(255,255,255,.52)', line: 'rgba(255,255,255,.06)', footer: 'rgba(255,255,255,.04)'};
  // Keep the same motion language at every duration. Fixed frame offsets made
  // short previews end before the ranking rows ever appeared.
  const totalFrames = Math.max(1, durationInFrames || 360);
  // Match the reference composition: the first row starts at frame 60 and
  // subsequent rows are staggered by 12 frames. Scale the tokens by the
  // requested duration so short previews still reveal the whole list.
  const rowStart = Math.max(8, Math.min(60, Math.round(totalFrames * (60 / 360))));
  const rowStagger = Math.max(3, Math.min(12, Math.round(totalFrames * (12 / 360))));
  const footerStart = Math.round(totalFrames * (200 / 360));
  const intro = spring({frame: frame - Math.round(totalFrames * (5 / 360)), fps, config: {damping: 14, stiffness: 100, mass: .6}});
  const subtitleOpacity = interpolate(frame, [Math.round(totalFrames * (20 / 360)), Math.round(totalFrames * (40 / 360))], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
  const lineProgress = interpolate(frame, [Math.round(totalFrames * (10 / 360)), Math.round(totalFrames * (45 / 360))], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
  const maxValue = Math.max(1, ...items.map(item => item.value));
  // Keep the source template's four-stop background so the center remains
  // restrained and the content, rather than the decoration, carries focus.
  const background = props.style_id === 'minimal-white'
    ? `linear-gradient(180deg, ${palette.bg} 0%, ${palette.bg2} 56%, ${palette.bg} 100%)`
    : 'linear-gradient(180deg, #080c1a 0%, #0f1629 30%, #131b33 60%, #0a0f1f 100%)';
  const title = props.title || '数据排行榜';
  const subtitle = props.subtitle || data.subtitle || '';
  const unit = props.unit || data.unit || '';
  const period = props.period || data.period || '';
  const source = props.source || data.source || '数据来源：HiMind 短视频创作引擎';
  const titleLines = String(title).split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
  const highlight = props.highlight || data.highlight || '';
  // Recipe safe-area values are already expressed in the requested canvas
  // coordinates. Only defaults use the 2160px reference scale.
  const safeTop = Number(props.safe_area && props.safe_area.top) || px(170);
  const safeLeft = Number(props.safe_area && props.safe_area.left) || px(150);
  const safeRight = Number(props.safe_area && props.safe_area.right) || px(200);
  const safeBottom = Number(props.safe_area && props.safe_area.bottom) || px(450);
  const rowHeight = px(228);
  const barColors = [
    ['#ff6b6b', '#ff8e53'], ['#ffca28', '#ffe082'], ['#4fc3f7', '#81d4fa'],
    ['#66bb6a', '#a5d6a7'], ['#ab47bc', '#ce93d8'], ['#ff7043', '#ffab91'],
    ['#26c6da', '#80deea'], ['#5c6bc0', '#9fa8da'], ['#ec407a', '#f48fb1'],
    ['#78909c', '#b0bec5'],
  ];
  const badgeEnd = props.style_id === 'minimal-white' ? palette.accent2 : '#ff8e53';
  return <AbsoluteFill style={{background, color: palette.text, fontFamily: 'Microsoft YaHei, PingFang SC, sans-serif', overflow: 'hidden'}}>
    <div style={{position: 'absolute', inset: 0, backgroundImage: `radial-gradient(circle at 1px 1px, rgba(255,255,255,${interpolate(frame % 120, [0, 60, 120], [.02, .05, .02])}) 1px, transparent 0)`, backgroundSize: `${px(48)}px ${px(48)}px`}} />
    <div style={{position: 'absolute', top: -px(360), left: '50%', transform: 'translateX(-50%)', width: px(1350), height: px(850), background: `radial-gradient(ellipse, ${palette.accent}14, transparent 68%)`, filter: `blur(${px(42)}px)`}} />
    <div style={{position: 'absolute', top: safeTop, left: safeLeft - px(10), width: px(4), height: height - safeTop - safeBottom, background: `linear-gradient(180deg, transparent 10%, ${palette.accent}4d 30%, ${palette.accent2}4d 70%, transparent 90%)`}} />
    <div style={{position: 'absolute', top: safeTop + px(30), left: 0, right: 0, textAlign: 'center'}}>
      {period ? <div style={{opacity: subtitleOpacity, marginBottom: px(24)}}><span style={{display: 'inline-block', padding: `${px(14)}px ${px(44)}px`, borderRadius: px(28), background: `linear-gradient(135deg, ${palette.accent}, ${badgeEnd})`, color: '#fff', fontSize: px(44), fontWeight: 700, letterSpacing: px(4)}}>{period}</span></div> : null}
      <div style={{fontSize: px(120), lineHeight: 1.2, fontWeight: 800, margin: 0, opacity: intro, transform: `translateY(${(1 - intro) * px(40)}px)`, letterSpacing: px(2)}}>{(titleLines.length ? titleLines : ['数据排行榜']).map((line, index, lines) => <div key={`${line}-${index}`} style={index === lines.length - 1 ? {background: `linear-gradient(90deg, ${palette.accent}, #ffca28, ${palette.accent2})`, WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent'} : {color: palette.text}}>{line}</div>)}</div>
      <div style={{height: px(4), width: `${lineProgress * px(600)}px`, maxWidth: '100%', margin: `${px(32)}px auto ${px(16)}px`, background: `linear-gradient(90deg, ${palette.accent}, #ffca28, ${palette.accent2}, transparent)`, transformOrigin: 'left', borderRadius: px(2)}} />
      <div style={{fontSize: px(44), color: palette.muted, opacity: subtitleOpacity, lineHeight: 1.35}}>{subtitle}</div>
    </div>
    <div style={{position: 'absolute', top: safeTop + px(500), left: safeLeft, right: safeRight}}>
      {items.map((item, index) => {
        const reveal = spring({frame: frame - rowStart - index * rowStagger, fps, config: {damping: 18, stiffness: 80, mass: .7}});
        const barReveal = spring({frame: frame - rowStart - 8 - index * rowStagger, fps, config: {damping: 25, stiffness: 60, mass: 1}});
        const valueReveal = interpolate(Math.max(0, frame - rowStart - index * rowStagger - 5), [0, Math.max(12, Math.round(totalFrames * (35 / 360)))], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
        const displayValue = Math.round(item.value * valueReveal);
        const medal = index < 3;
        const colors = barColors[index] || [palette.accent2, palette.accent2];
        const medalColors = index === 0 ? ['#ffd700', '#ffa000'] : index === 1 ? ['#c0c0c0', '#9e9e9e'] : ['#cd7f32', '#a0522d'];
        return <div key={index} style={{height: rowHeight, display: 'flex', alignItems: 'center', gap: px(36), opacity: reveal, transform: `translateX(${(1 - reveal) * px(120)}px)`, borderBottom: `${px(1)}px solid ${props.style_id === 'minimal-white' ? palette.line : 'rgba(255,255,255,.04)'}`}}>
          <div style={{width: px(92), height: px(92), borderRadius: medal ? '50%' : px(18), background: medal ? `linear-gradient(135deg, ${medalColors[0]}, ${medalColors[1]})` : palette.panel, border: `${px(3)}px solid ${medal ? medalColors[0] : palette.line}`, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: medal ? `0 0 ${px(26)}px ${medalColors[0]}66, inset 0 ${px(2)}px ${px(4)}px rgba(255,255,255,.3)` : 'none', flexShrink: 0}}><span style={{fontSize: px(45), fontWeight: 800, fontFamily: 'Helvetica Neue, Arial, sans-serif', color: medal ? '#fff' : palette.muted, textShadow: medal ? '0 1px 3px rgba(0,0,0,.3)' : 'none'}}>{index + 1}</span></div>
          <div style={{flex: 1, minWidth: 0}}>
            <div style={{display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: px(18), marginBottom: px(14)}}>
              <div style={{display: 'flex', alignItems: 'center', gap: px(14), minWidth: 0}}><span style={{fontSize: px(48), width: px(54)}}>{item.icon}</span><span style={{fontSize: px(52), fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis'}}>{item.name}</span></div>
              <div style={{display: 'flex', alignItems: 'baseline', gap: px(8), flexShrink: 0}}><span style={{fontSize: px(58), fontWeight: 700, fontFamily: 'Helvetica Neue, Arial, sans-serif', fontVariantNumeric: 'tabular-nums'}}>{displayValue.toLocaleString()}</span>{unit ? <span style={{fontSize: px(34), color: palette.muted}}>{unit}</span> : null}</div>
            </div>
            <div style={{height: px(32), background: palette.panel, borderRadius: px(18), overflow: 'hidden'}}><div style={{height: '100%', width: `${(item.value / maxValue) * 100 * barReveal}%`, background: `linear-gradient(90deg, ${colors[0]}, ${colors[1]})`, borderRadius: px(18), boxShadow: `0 0 ${px(20)}px ${colors[0]}44`, position: 'relative'}}><div style={{position: 'absolute', top: px(2), left: px(8), right: px(8), height: '40%', background: 'linear-gradient(180deg, rgba(255,255,255,.25), transparent)', borderRadius: px(16)}} /></div></div>
            {item.subValue ? <div style={{display: 'flex', alignItems: 'center', gap: px(6), marginTop: px(8), fontSize: px(34), color: '#4caf50', opacity: interpolate(frame - rowStart - 20 - index * rowStagger, [0, Math.max(8, Math.round(totalFrames * .04))], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})}}><span style={{fontWeight: 600}}>▲</span><span>同比 {item.subValue}</span></div> : null}
          </div>
        </div>;
      })}
    </div>
    <div style={{position: 'absolute', left: 0, right: 0, bottom: safeBottom - px(200), padding: `${px(40)}px ${safeRight}px ${px(40)}px ${safeLeft}px`, background: props.style_id === 'minimal-white' ? 'linear-gradient(transparent, rgba(16,35,61,.06))' : 'linear-gradient(transparent, rgba(0,0,0,.3))', opacity: interpolate(frame, [footerStart, footerStart + Math.max(8, Math.round(totalFrames * .06))], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})}}>
      <div style={{padding: `${px(32)}px ${px(40)}px`, marginBottom: px(32), background: palette.footer, border: `${px(1)}px solid ${props.style_id === 'minimal-white' ? palette.line : 'rgba(255,255,255,.06)'}`, borderRadius: px(20), display: 'flex', justifyContent: 'space-between', gap: px(30)}}>
      {[["TOP10 合计", props.total || data.total || '—', palette.accent], ['平均增长', props.average || data.average || '—', '#4caf50'], [highlight ? '最高增速' : '数据周期', highlight || period || '—', '#ffca28']].map((stat, index) => <div key={index} style={{flex: 1, textAlign: 'center'}}><div style={{fontSize: px(32), color: palette.muted, marginBottom: px(8)}}>{stat[0]}</div><div style={{fontSize: px(58), fontWeight: 700, color: stat[2], fontFamily: 'Helvetica Neue, Arial, sans-serif'}}>{stat[1]}</div></div>)}
      </div>
      <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}><span style={{color: props.style_id === 'minimal-white' ? palette.muted : 'rgba(255,255,255,.2)', fontSize: px(32)}}>{source}</span><span style={{color: props.style_id === 'minimal-white' ? palette.muted : 'rgba(255,255,255,.15)', fontSize: px(28), fontFamily: 'Helvetica Neue, Arial, sans-serif'}}>Made with Remotion · Video Flow</span></div>
    </div>
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
