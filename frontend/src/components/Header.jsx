import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { apiLogout } from '../api';
import { useNavigate } from 'react-router-dom';

export default function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await apiLogout();
    logout();
    navigate('/');
  };

  return (
    <header className="header">
      <Link to="/" className="header-logo">
        <div className="logo-dot" />
        GoLive
      </Link>
      <div className="header-actions">
        {user ? (
          <>
            <span className="header-user">{user.username}</span>
            <Link to={`/live/${user.roomId}`} className="header-btn header-btn-accent">
              我的直播间
            </Link>
            <button className="header-btn" onClick={handleLogout}>登出</button>
          </>
        ) : (
          <>
            <Link to="/register" className="header-btn header-btn-accent">开始直播</Link>
            <Link to="/login" className="header-btn">登录</Link>
          </>
        )}
      </div>
    </header>
  );
}
