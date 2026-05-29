
import request from '../utils/request';

export const AUTH_CHANGED_EVENT = 'lehu:auth-changed';

const emitAuthChanged = () => {
    window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
};

// 用户API服务
export const userApi = {
    // 获取验证码
    getVerificationCode: (data) => {
        return request.post('/user/code', data);
    },

    // 注册
    register: (data) => {
        return request.post('/user/register', data);
    },

    // 登录
    login: (data) => {
        return request.post('/user/login', data);
    },

    // 获取用户信息（用于查看其他用户信息）
    getUserInfo: (userId) => {
        return request.get(`/user/info/${userId}`);
    },

    // 批量获取用户信息
    batchGetUserInfo: (data) => {
        return request.post('/users/batch', data);
    },

    // 更新用户信息
    updateUserInfo: (data) => {
        return request.put('/user/info', data);
    },

    // 搜索用户
    searchUsers: (data) => {
        return request.post('/users/search', data);
    },

    // 绑定凭证（手机/邮箱）
    bindUserVoucher: (data) => {
        return request.post('/user/voucher', data);
    },

    // 解绑凭证（手机/邮箱）
    unbindUserVoucher: (data) => {
        return request.delete('/user/voucher', { data });
    }
};

// 工具函数：判断是手机号还是邮箱
export const identifyInputType = (input) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    const phoneRegex = /^1[3-9]\d{9}$/; // 简单的中国手机号验证

    if (emailRegex.test(input)) {
        return 'email';
    } else if (phoneRegex.test(input)) {
        return 'mobile';
    }
    return null;
};

// VoucherType枚举
export const VoucherType = {
    PHONE: 0,
    EMAIL: 1
};

// 获取voucher类型文本
export const getVoucherTypeText = (type) => {
    return type === VoucherType.PHONE ? '手机号' : '邮箱';
};

// 存储用户信息时也修复 mobile 字段
export const saveUserData = (token, userInfo) => {
    localStorage.setItem('token', token);

    // 确保userInfo是对象且包含id，将mobile转为字符串
    const userData = {
        id: userInfo.id || userInfo.user_id || "0",
        name: userInfo.name || '',
        nickname: userInfo.nickname || '',
        avatar: userInfo.avatar || '',
        background_image: userInfo.background_image || '',
        signature: userInfo.signature || '',
        mobile: String(userInfo.mobile || ''),
        email: userInfo.email || '',
        gender: userInfo.gender || 0,
        follow_count: userInfo.follow_count || 0,
        follower_count: userInfo.follower_count || 0,
        total_favorited: userInfo.total_favorited || 0,
        work_count: userInfo.work_count || 0,
        favorite_count: userInfo.favorite_count || 0,
        created_at: userInfo.created_at || new Date().toISOString(),
        online_status: userInfo.online_status || 0,
        last_online_time: userInfo.last_online_time || ''
    };

    localStorage.setItem('userInfo', JSON.stringify(userData));
    emitAuthChanged();
    return userData;
};

/**
 * 用户相关工具函数
 */
export const getCurrentUser = () => {
    try {
        const userInfo = localStorage.getItem('userInfo');
        return userInfo ? JSON.parse(userInfo) : null;
    } catch {
        return null;
    }
};

export const searchUsers =  (keyword, pageStats = { page: 1, page_size: 20 }) => {
    return request.post('/users/search', {
        keyword,
        page_stats: pageStats
    })
};

export const getUserDisplayName = (user) => {
    if (!user) return '用户';
    return user.nickname || user.name || user.username || '用户';
};

export const isLoggedIn = () => {
    const token = localStorage.getItem('token');
    const user = getCurrentUser();
    return !!(token && user);
};

export const clearUserData = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('userInfo');
    localStorage.removeItem('refresh_token');
    emitAuthChanged();
};

// 更新本地存储的用户信息
export const updateLocalUserInfo = (newUserInfo) => {
    const currentUser = getCurrentUser();
    if (currentUser) {
        const updatedUser = {
            ...currentUser,
            ...newUserInfo,
            // 确保id不变
            id: currentUser.id
        };
        localStorage.setItem('userInfo', JSON.stringify(updatedUser));
        emitAuthChanged();
        return updatedUser;
    }
    return null;
};

// 获取用户头像
export const getUserAvatar = (user) => {
    if (!user) return '/default-avatar.png';
    return user.avatar || '/default-avatar.png';
};

// 获取用户背景图
export const getUserBackground = (user) => {
    if (!user) return '/default-background.jpg';
    return user.background_image || '/default-background.jpg';
};

// 格式化用户统计数据
export const formatUserStats = (user) => {
    return {
        following: user.follow_count || 0,
        followers: user.follower_count || 0,
        likes: user.total_favorited || 0,
        works: user.work_count || 0,
        favorites: user.favorite_count || 0
    };
};

// 检查用户是否在线
export const isUserOnline = (user) => {
    if (!user || !user.online_status) return false;
    return user.online_status === 1; // 1表示在线
};

// 获取用户在线状态文本
export const getOnlineStatusText = (user) => {
    if (!user || !user.online_status) return '离线';

    switch (user.online_status) {
        case 1: return '在线';
        case 2: return '忙碌';
        case 3: return '离开';
        default: return '离线';
    }
};

// 获取用户在线状态颜色
export const getOnlineStatusColor = (user) => {
    if (!user || !user.online_status) return '#9E9E9E';

    switch (user.online_status) {
        case 1: return '#4CAF50'; // 绿色
        case 2: return '#F44336'; // 红色
        case 3: return '#FF9800'; // 橙色
        default: return '#9E9E9E'; // 灰色
    }
};
