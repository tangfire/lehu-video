import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { userApi, identifyInputType, saveUserData } from '../../api/user';
import { campusAdminApi } from '../../api/admin';
import './Admin.css';

const AdminLogin = () => {
    const [account, setAccount] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();
    const location = useLocation();
    const from = location.state?.from?.pathname || '/admin';

    const submit = async (event) => {
        event.preventDefault();
        const accountType = identifyInputType(account.trim());
        if (!accountType) {
            setError('请输入正确的手机号或邮箱');
            return;
        }
        if (!password.trim()) {
            setError('请输入密码');
            return;
        }
        setLoading(true);
        setError('');
        try {
            const payload = { password };
            if (accountType === 'mobile') {
                payload.mobile = account.trim();
            } else {
                payload.email = account.trim();
            }
            const data = await userApi.login(payload);
            if (!data.token || !data.user) {
                throw new Error('登录响应不完整');
            }
            saveUserData(data.token, data.user);
            await campusAdminApi.summary();
            navigate(from, { replace: true });
        } catch (err) {
            setError(err.message || '登录失败或没有后台权限');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="admin-login-page">
            <div className="admin-login-panel">
                <h1>深汕校园e站后台</h1>
                <p>运营内容、举报审核和数据总览统一在这里处理。</p>
                {error && <div className="admin-error">{error}</div>}
                <form className="admin-form" onSubmit={submit}>
                    <div className="admin-field">
                        <label>手机号/邮箱</label>
                        <input className="admin-input" value={account} onChange={(e) => setAccount(e.target.value)} disabled={loading} />
                    </div>
                    <div className="admin-field">
                        <label>密码</label>
                        <input className="admin-input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={loading} />
                    </div>
                    <button className="admin-button primary" disabled={loading}>
                        {loading ? '登录中...' : '登录后台'}
                    </button>
                </form>
            </div>
        </div>
    );
};

export default AdminLogin;
