import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import Artplayer from 'artplayer';
import { apiMe, apiResetKey } from '../api';

const HOST = `${window.location.protocol}//${window.location.host}`;

export default function LiveRoom() {
  const { roomId } = useParams();
  const artRef = useRef(null);
  const artInstanceRef = useRef(null);
  const pcRef = useRef(null);
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('loading');
  const [isOwner, setIsOwner] = useState(false);
  const [ownerKey, setOwnerKey] = useState('');
  const [keyRevealed, setKeyRevealed] = useState(false);
  const [viewerCount, setViewerCount] = useState(0);
  const [uptime, setUptime] = useState('');
  const [duration, setDuration] = useState(0);
  const [stats, setStats] = useState({ fps: 0, bitrate: 0, rtt: 0, jitter: 0, res: '' });
  const statsTimerRef = useRef(null);
  const durationTimerRef = useRef(null);
  const isManualDisconnect = useRef(false);
  const checkTimerRef = useRef(null);
  const [toast, setToast] = useState('');

  useEffect(() => {
    apiMe().then(data => {
      if (data.loggedIn && data.user.roomId === roomId) {
        setIsOwner(true);
        setOwnerKey(data.user.streamKey || '');
      }
    });
  }, [roomId]);

  const showToast = (msg) => {
    setToast(msg);
    setTimeout(() => setToast(''), 2500);
  };

  const toggleKey = () => setKeyRevealed(v => !v);

  const handleResetKey = async () => {
    if (!confirm('确定要重置推流密钥吗？重置后旧密钥立即失效。')) return;
    try {
      const data = await apiResetKey();
      setOwnerKey(data.user?.streamKey || '');
      setKeyRevealed(false);
      showToast('密钥已重置');
    } catch (err) {
      showToast(err.message || '重置失败');
    }
  };

  const copyToClipboard = (textToCopy) => {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(textToCopy).then(() => showToast('已复制')).catch(() => {});
    } else {
      const textArea = document.createElement('textarea');
      textArea.value = textToCopy;
      textArea.style.position = 'absolute';
      textArea.style.opacity = '0';
      textArea.style.left = '-999999px';
      textArea.style.top = '-999999px';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      document.execCommand('copy') ? showToast('已复制') : showToast('复制失败');
      textArea.remove();
    }
  };

  // 初始化 ArtPlayer
  const initArt = useCallback((el) => {
    if (!el || artInstanceRef.current) return;

    const art = new Artplayer({
      container: el,
      url: '',
      autoplay: false,
      muted: false,
      pip: true,
      autoSize: true,
      autoMini: true,
      screenshot: true,
      setting: true,
      loop: false,
      flip: true,
      playbackRate: true,
      aspectRatio: true,
      fullscreen: true,
      fullscreenWeb: true,
      miniProgressBar: true,
      playsInline: true,
      mutex: true,
      backdrop: true,
      whitelist: ['*'],
      moreVideoAttr: {
        crossOrigin: 'anonymous',
      },
    });

    // ArtPlayer v5: video 元素在 play() 后才创建
    // 等待 video 准备好后，如果有待播放的流则立即播放
    art.on('ready', () => {
      const art2 = artInstanceRef.current;
      if (!art2) return;
      if (art2._pendingStream) {
        const stream = art2._pendingStream;
        delete art2._pendingStream;
        if (art2.video) {
          art2.video.srcObject = stream;
          art2.video.play().catch(e => console.warn('[ArtPlayer] play error:', e));
        }
      }
    });

    art.on('error', (e) => console.warn('[ArtPlayer] error:', e));
    artInstanceRef.current = art;
  }, []);

  // 检查房间状态
  useEffect(() => {
    const check = async () => {
      if (isManualDisconnect.current) return;
      try {
        const res = await fetch(`/status?room=${roomId}`);
        if (res.status === 503) {
          if (connected || connecting) {
            closePC();
            setStatus('offline');
            setConnected(false);
            setConnecting(false);
          }
        } else if (res.ok) {
          const data = await res.json();
          setViewerCount(data.viewerCount ?? 0);
          if (data.uptime) setUptime(data.uptime);
          if (!connected && !connecting) {
            setStatus('loading');
          }
        }
      } catch {}
    };
    check();
    checkTimerRef.current = setInterval(check, 3000);
    return () => clearInterval(checkTimerRef.current);
  }, [roomId, connected, connecting]);

  const closePC = () => {
    if (pcRef.current) {
      try { pcRef.current.close(); } catch {}
      pcRef.current = null;
    }
    const art = artInstanceRef.current;
    if (art) {
      try { art.destroy(false); } catch {}
      artInstanceRef.current = null;
    }
    if (statsTimerRef.current) {
      clearInterval(statsTimerRef.current);
      statsTimerRef.current = null;
    }
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current);
      durationTimerRef.current = null;
    }
  };

  const startStats = useCallback(() => {
    if (statsTimerRef.current) clearInterval(statsTimerRef.current);
    let lastBytes = 0, lastTs = 0;
    statsTimerRef.current = setInterval(async () => {
      const pc = pcRef.current;
      if (!pc) return;
      try {
        const reports = await pc.getStats();
        let fps = 0, width = 0, height = 0, rtt = 0, jitter = 0;
        reports.forEach(r => {
          if (r.type === 'inbound-rtp' && r.kind === 'video') {
            if (r.framesPerSecond !== undefined) fps = r.framesPerSecond;
            if (r.frameWidth !== undefined) width = r.frameWidth;
            if (r.frameHeight !== undefined) height = r.frameHeight;
            if (r.timestamp && r.bytesReceived && lastTs) {
              const elapsed = (r.timestamp - lastTs) / 1000;
              if (elapsed > 0) {
                const br = Math.round(((r.bytesReceived - lastBytes) * 8) / elapsed / 1000);
                setStats(s => ({ ...s, bitrate: br > 0 ? br : s.bitrate }));
              }
            }
            if (r.bytesReceived) lastBytes = r.bytesReceived;
            if (r.timestamp) lastTs = r.timestamp;
          }
          if (r.type === 'candidate-pair' && r.state === 'succeeded') {
            if (r.currentRoundTripTime !== undefined) rtt = Math.round(r.currentRoundTripTime * 1000);
            if (r.jitter !== undefined) jitter = Math.round(r.jitter * 1000);
          }
        });
        setStats(s => ({ ...s, fps, rtt, jitter, res: width && height ? `${width}x${height}` : s.res }));
      } catch {}
    }, 1000);
  }, []);

  const connect = useCallback(async () => {
    if (connecting || connected) return;
    isManualDisconnect.current = false;
    setConnecting(true);
    setStatus('connecting');
    setError('');

    closePC();

    const pc = new RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.miwifi.com:3478' },
        { urls: 'stun:stun.chat.bilibili.com:3478' },
        { urls: 'stun:stun.douyucdn.cn:18000' },
        { urls: 'stun:stun.hitv.com:3478' },
        { urls: 'stun:stun.cloudflare.com:3478' },
        { urls: 'stun:stun.l.google.com:19302' },
      ]
    });
    pcRef.current = pc;

    pc.oniceconnectionstatechange = () => {
      console.log('[LiveRoom] ICE:', pc.iceConnectionState);
      if (pc.iceConnectionState === 'connected' || pc.iceConnectionState === 'completed') {
        setConnected(true);
        setConnecting(false);
        setStatus('connected');
        durationTimerRef.current = setInterval(() => setDuration(d => d + 1), 1000);
        startStats();
      }
      if (['failed', 'closed'].includes(pc.iceConnectionState)) {
        if (!isManualDisconnect.current) {
          closePC();
          setConnected(false);
          setConnecting(false);
          setStatus('offline');
        }
      }
    };

    // pion-webrtc v4: e.streams[0] 是 MediaStream
    pc.ontrack = (e) => {
      console.log('[LiveRoom] ontrack:', e.streams);
      const stream = e.streams && e.streams[0];
      if (!stream) return;

      const art = artInstanceRef.current;
      if (!art) return;

      if (art.video) {
        // video 已就绪，直接赋值
        art.video.srcObject = stream;
        art.video.play().catch(e => console.warn('[ArtPlayer] play error:', e));
      } else {
        // video 尚未创建，缓存流，等 'ready' 事件触发
        art._pendingStream = stream;
        art.play().catch(e => console.warn('[ArtPlayer] play error:', e));
      }
    };

    try {
      pc.addTransceiver('video', { direction: 'recvonly' });
      pc.addTransceiver('audio', { direction: 'recvonly' });
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      const res = await fetch(`/whep/${roomId}`, {
        method: 'POST',
        body: pc.localDescription.sdp,
        headers: { 'Content-Type': 'application/sdp' },
      });
      if (!res.ok) throw new Error(`服务器 ${res.status}`);
      const answer = await res.text();
      if (pc.signalingState === 'closed') return;
      await pc.setRemoteDescription({ sdp: answer, type: 'answer' });
    } catch (err) {
      console.error('[LiveRoom] connect error:', err);
      closePC();
      setConnecting(false);
      setStatus('error');
      setError(err.message);
    }
  }, [roomId, connecting, connected, startStats]);

  useEffect(() => {
    return () => closePC();
  }, []);

  useEffect(() => {
    if (status === 'loading' || status === 'connecting') {
      connect();
    }
  }, [status, connect]);

  const formatDuration = (s) => {
    const h = String(Math.floor(s / 3600)).padStart(2, '0');
    const m = String(Math.floor((s % 3600) / 60)).padStart(2, '0');
    const sec = String(s % 60).padStart(2, '0');
    return `${h}:${m}:${sec}`;
  };

  const pushUrl = `${HOST}/whip/${roomId}`;

  return (
    <div className="live-page">
      <div className="live-status-bar">
        <div className="live-status-left">
          <div className={`live-badge ${connected ? 'on-air' : 'offline'}`}>
            <div className="live-dot" />
            <span>{connected ? '直播中' : '未开播'}</span>
          </div>
          <span className="status-text">
            {status === 'loading' && '等待推流...'}
            {status === 'connecting' && '正在连接...'}
            {status === 'error' && '连接失败'}
            {status === 'offline' && !connected && '暂未开播'}
            {connected && '直播中'}
          </span>
        </div>
        <div className="live-status-right">
          {uptime && <span className="stat-item">&#9201; {uptime}</span>}
          <span className="stat-item">&#128101; {viewerCount}</span>
        </div>
      </div>

      <div className="live-content">
        <div className="video-area">
          <div className="video-wrapper">
            {/* ArtPlayer 容器 — 仅在连接/已连接时渲染 */}
            {(connected || connecting) && (
              <div
                ref={initArt}
                className="artplayer-app"
                style={{ width: '100%', height: '100%', background: '#000' }}
              />
            )}

            {status === 'loading' && (
              <div className="overlay loading-overlay">
                <div className="spinner" />
                <span>正在连接...</span>
              </div>
            )}

            {status === 'offline' && !connected && (
              <div className="overlay placeholder-overlay">
                <div className="placeholder-icon">&#127909;</div>
                <div className="placeholder-title">暂未开播</div>
                <div className="placeholder-sub">
                  主播尚未开始直播<br />
                  {isOwner && <code>{pushUrl}</code>}
                </div>
              </div>
            )}

            {status === 'error' && (
              <div className="overlay error-overlay">
                <div className="error-title">{error || '连接失败'}</div>
                <button className="btn-primary" onClick={connect}>重试</button>
              </div>
            )}
          </div>

          <div className="control-bar">
            <span className="stream-duration">{formatDuration(duration)}</span>
            <div className="control-right">
              {connected && (
                <span className="stat-pill">
                  {stats.res} {stats.fps > 0 ? `${stats.fps}fps` : ''} {stats.bitrate}kbps
                </span>
              )}
              <button className="ctrl-btn" onClick={connect} title="重连">&#8635;</button>
            </div>
          </div>
        </div>

        <div className="sidebar">
          <div className="sidebar-section">
            <div className="sidebar-title">直播信息</div>
            <div className="info-grid">
              <div className="info-row">
                <span className="info-label">房间ID</span>
                <span className="info-value mono">{roomId}</span>
              </div>

              {isOwner && (
                <>
                  <div className="info-row">
                    <span className="info-label">推流地址</span>
                    <div className="url-row">
                      <span className="info-value copyable mono" onClick={() => copyToClipboard(pushUrl)} title="点击复制">
                        {pushUrl}
                      </span>
                      <button className="key-copy-btn" onClick={() => copyToClipboard(pushUrl)} title="复制推流地址">复制</button>
                    </div>
                  </div>
                  <div className="info-row">
                    <span className="info-label">Bearer 令牌</span>
                    <div className="key-row">
                      <span
                        className={`mono key-val ${keyRevealed ? '' : 'masked'}`}
                        onClick={() => keyRevealed && copyToClipboard(ownerKey)}
                        title={keyRevealed ? '点击复制' : '先点击"显示"再复制'}
                        style={{ cursor: keyRevealed ? 'pointer' : 'default' }}
                      >
                        {ownerKey ? (keyRevealed ? ownerKey : ownerKey.replace(/./g, '*')) : ''}
                      </span>
                      <button className="key-btn" onClick={toggleKey}>
                        {keyRevealed ? '隐藏' : '显示'}
                      </button>
                      {keyRevealed && (
                        <button className="key-btn" onClick={() => copyToClipboard(ownerKey)} title="复制密钥">
                          复制
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="info-row">
                    <button className="reset-btn" onClick={handleResetKey}>重置密钥</button>
                  </div>
                </>
              )}

              {!isOwner && (
                <div className="info-row">
                  <span className="info-label">播放地址</span>
                  <span className="info-value copyable mono" onClick={() => copyToClipboard(`${HOST}/live/${roomId}`)}>
                    {`${HOST}/live/${roomId}`}
                  </span>
                </div>
              )}

              {connected && stats.fps > 0 && (
                <div className="info-row stats-row">
                  <div className="stats-group-title">播放统计</div>
                  <div className="stats-item">分辨率: {stats.res}</div>
                  <div className="stats-item">帧率: {stats.fps} fps</div>
                  <div className="stats-item">码率: {stats.bitrate} kbps</div>
                  <div className="stats-item">RTT: {stats.rtt} ms</div>
                  <div className="stats-item">抖动: {stats.jitter} ms</div>
                </div>
              )}
            </div>
          </div>

          <div className="sidebar-section" style={{ flex: 1 }}>
            <div className="sidebar-title">直播互动</div>
            <div className="chat-placeholder">
              <div className="chat-icon">&#128172;</div>
              <div className="chat-text">互动功能即将上线</div>
            </div>
          </div>
        </div>
      </div>

      {toast && <div className="toast">{toast}</div>}
    </div>
  );
}
