import { useState, useEffect, useRef, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { apiRooms } from '../api';

function RoomCard({ room }) {
  const [playing, setPlaying] = useState(false);
  const pcRef = useRef(null);
  const videoRef = useRef(null);
  const timeoutRef = useRef(null);

  const closePreview = useCallback(() => {
    if (pcRef.current) {
      try { pcRef.current.close(); } catch {}
      pcRef.current = null;
    }
    setPlaying(false);
  }, []);

  const openPreview = useCallback(async () => {
    if (pcRef.current || !room.active) return;

    const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
    pcRef.current = pc;

    pc.ontrack = (e) => {
      if (videoRef.current) {
        videoRef.current.srcObject = e.streams[0];
        setPlaying(true);
      }
    };

    pc.oniceconnectionstatechange = () => {
      if (['failed', 'closed'].includes(pc.iceConnectionState)) closePreview();
    };

    try {
      pc.addTransceiver('video', { direction: 'recvonly' });
      pc.addTransceiver('audio', { direction: 'recvonly' });
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      const res = await fetch(`/whep/${room.id}`, {
        method: 'POST',
        body: pc.localDescription.sdp,
        headers: { 'Content-Type': 'application/sdp' },
      });
      if (res.ok) {
        const answer = await res.text();
        if (pc.signalingState === 'closed') return;
        await pc.setRemoteDescription({ sdp: answer, type: 'answer' });
      } else {
        closePreview();
      }
    } catch {
      closePreview();
    }
  }, [room, closePreview]);

  const handleEnter = () => {
    clearTimeout(timeoutRef.current);
    if (room.active) openPreview();
  };

  const handleLeave = () => {
    timeoutRef.current = setTimeout(closePreview, 300);
  };

  useEffect(() => () => { clearTimeout(timeoutRef.current); closePreview(); }, [closePreview]);

  return (
    <Link
      to={`/live/${room.id}`}
      className={`room-card ${room.active ? 'live-card' : ''}`}
      onMouseEnter={handleEnter}
      onMouseLeave={handleLeave}
    >
      <div className="card-preview">
        <video ref={videoRef} muted autoPlay playsInline webkit-playsinline="true" />
        <div className={`card-bg ${playing ? 'hidden' : ''}`}>
          <div className="card-icon">&#127909;</div>
          {room.active && <div className="card-hover-hint">悬停预览</div>}
        </div>
        <div className={`card-spinner ${playing ? '' : 'loading'}`}>
          <div className="mini-spinner" />
        </div>
        <div className={`card-badge ${room.active ? 'on-air' : 'offline'}`}>
          <div className="badge-dot" />
          <span>{room.active ? '直播中' : '未开播'}</span>
        </div>
        {room.viewers > 0 && (
          <div className="card-viewers">
            <span>&#128101;</span>
            <span>{room.viewers}</span>
          </div>
        )}
      </div>
      <div className="card-body">
        <div className="card-name">{room.name}</div>
        <div className="card-meta">
          <span>ID: {room.id}</span>
          {room.active && <span className="card-live">● 直播中</span>}
        </div>
      </div>
    </Link>
  );
}

export default function Home() {
  const [rooms, setRooms] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchRooms = useCallback(async () => {
    try {
      const data = await apiRooms();
      setRooms(Array.isArray(data) ? data : []);
    } catch {
      setRooms([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRooms();
    const t = setInterval(fetchRooms, 10000);
    return () => clearInterval(t);
  }, [fetchRooms]);

  const liveRooms = rooms.filter(r => r.active);
  const offlineRooms = rooms.filter(r => !r.active);
  const sorted = [...liveRooms, ...offlineRooms];

  return (
    <div className="page">
      <div className="hero">
        <h1>Go<span className="accent">Live</span></h1>
        <p>稳定、低延迟的实时直播平台</p>
      </div>

      <div className="section">
        <div className="section-header">
          <div className="section-title">
            <div className="live-dot" />
            正在直播
            {liveRooms.length > 0 && <span className="live-count">({liveRooms.length})</span>}
          </div>
        </div>

        {loading && (
          <div className="skeleton-grid">
            {[1,2,3].map(i => (
              <div key={i} className="skeleton-card">
                <div className="skeleton-preview" />
                <div className="skeleton-body">
                  <div className="skeleton-line" />
                  <div className="skeleton-line short" />
                </div>
              </div>
            ))}
          </div>
        )}

        {!loading && sorted.length === 0 && (
          <div className="empty-state">
            <div className="empty-icon">&#127909;</div>
            <div className="empty-title">暂无开播房间</div>
            <div className="empty-sub">
              注册账号即可开通专属直播间<br />
              <Link to="/register" className="accent-link">立即注册</Link>
            </div>
          </div>
        )}

        {!loading && sorted.length > 0 && (
          <div className="room-grid">
            {sorted.map(room => <RoomCard key={room.id} room={room} />)}
          </div>
        )}
      </div>
    </div>
  );
}
