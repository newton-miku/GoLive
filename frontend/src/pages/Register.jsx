import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { apiRegister } from '../api';
import { useAuth } from '../context/AuthContext';

export default function Register() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [password2, setPassword2] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { checkAuth } = useAuth();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    if (password !== password2) {
      setError('两次密码不一致');
      return;
    }
    setLoading(true);
    try {
      const data = await apiRegister(username, password);
      // 注册成功后直接用响应中的用户信息更新状态，避免额外的 apiMe() 调用
      checkAuth(); // 触发 AuthContext 重新检查
      navigate('/');
    } catch (err) {
      setError(err.message || '注册失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-logo">
          <div className="logo-dot" />
          GoLive
        </div>
        <h1 className="auth-title">注册账号</h1>
        <p className="auth-sub">注册后自动获得专属直播间</p>
        {error && <div className="auth-error">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>用户名</label>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value.trim())}
              placeholder="3-20位字母、数字或下划线"
              required
              minLength={3}
              maxLength={20}
              pattern="[a-zA-Z0-9_]+"
              autoFocus
            />
          </div>
          <div className="form-group">
            <label>密码</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="至少 6 个字符"
              required
              minLength={6}
              maxLength={32}
            />
          </div>
          <div className="form-group">
            <label>确认密码</label>
            <input
              type="password"
              value={password2}
              onChange={e => setPassword2(e.target.value)}
              placeholder="再次输入密码"
              required
              minLength={6}
              maxLength={32}
            />
          </div>
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? '注册中...' : '注册'}
          </button>
        </form>
        <Link to="/login" className="auth-link">已有账号？去登录</Link>
      </div>
    </div>
  );
}
