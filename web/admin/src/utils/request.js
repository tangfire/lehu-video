// utils/request.js
import axios from 'axios';
import { clearUserData } from '../api/user';

// 创建axios实例
const request = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:18080/v1',
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json'
    }
});

// 请求跟踪
const pendingRequests = new Map();
const requestTimers = new Map();

const isFormData = (value) => typeof FormData !== 'undefined' && value instanceof FormData;

const createRequestId = () => {
    const random = Math.random().toString(16).slice(2, 10);
    return `web-${Date.now()}-${random}`;
};

const withRequestIdMessage = (message, requestId) => {
    if (!requestId || !window.location.pathname.startsWith('/admin')) {
        return message;
    }
    return `${message}\n请求编号：${requestId}`;
};

// 生成请求唯一标识
const generateRequestKey = (config) => {
    const url = config.url || '';
    const method = config.method || 'get';
    const methodLower = method.toLowerCase();

    if (methodLower === 'get') {
        return `${method}_${url}_${Date.now()}_${Math.random()}`;
    }

    if (isFormData(config.data)) {
        return `${method}_${url}_${Date.now()}_${Math.random()}`;
    }

    // 写操作：使用原有防重逻辑（相同 key 的请求会被取消）
    const dataStr = config.data ? JSON.stringify(config.data) : '';
    const paramsStr = config.params ? JSON.stringify(config.params) : '';
    return `${method}_${url}_${dataStr}_${paramsStr}`;
};

const cleanupRequest = (config) => {
    const requestKey = config?.metadata?.requestKey;
    if (!requestKey) return;

    pendingRequests.delete(requestKey);
    const timer = requestTimers.get(requestKey);
    if (timer) {
        clearTimeout(timer);
        requestTimers.delete(requestKey);
    }
};

// 请求拦截器
request.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        const requestId = config.headers['X-Request-ID'] || config.headers['x-request-id'] || createRequestId();
        config.headers['X-Request-ID'] = requestId;

        if (isFormData(config.data)) {
            delete config.headers['Content-Type'];
            delete config.headers['content-type'];
        }

        // 处理请求数据，确保ID为字符串
        if (config.data && !isFormData(config.data)) {
            config.data = processRequestData(config.data);
        }

        if (config.params) {
            config.params = processRequestData(config.params);
        }

        const requestKey = generateRequestKey(config);

        if (pendingRequests.has(requestKey)) {
            return Promise.reject(new Error('重复请求已取消'));
        }

        config.metadata = {
            ...(config.metadata || {}),
            requestKey,
            requestId
        };
        pendingRequests.set(requestKey, true);

        // 30秒后自动清理
        const timer = setTimeout(() => {
            pendingRequests.delete(requestKey);
            requestTimers.delete(requestKey);
        }, 30000);

        requestTimers.set(requestKey, timer);

        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// 响应拦截器
request.interceptors.response.use(
    (response) => {
        const { data: responseData } = response;

        // 清理请求标记
        cleanupRequest(response.config);

        // 处理响应数据，确保ID为字符串
        const processedData = processResponseData(responseData);

        if (processedData && processedData.code === 0) {
            return processedData.data;
        } else if (processedData && processedData.code !== undefined) {
            const requestId = processedData.request_id || response.headers?.['x-request-id'] || response.config?.metadata?.requestId || '';
            const error = {
                code: processedData.code || -1,
                message: withRequestIdMessage(processedData.message || '请求失败', requestId),
                data: processedData.data,
                timestamp: processedData.timestamp,
                request_id: requestId,
                requestId
            };
            console.error('[request failed]', error);
            return Promise.reject(error);
        } else {
            return processedData;
        }
    },
    (error) => {
        cleanupRequest(error.config);

        if (error.response?.status === 401) {
            clearUserData();
            setTimeout(() => {
                window.location.href = '/admin/login';
            }, 100);
        }

        const message = error.response?.data?.message || error.message || '请求失败';
        const requestId = error.response?.data?.request_id || error.response?.headers?.['x-request-id'] || error.config?.metadata?.requestId || '';
        const rejectError = {
            code: error.response?.status || 500,
            message: withRequestIdMessage(message, requestId),
            data: error.response?.data,
            request_id: requestId,
            requestId
        };
        console.error('[request failed]', rejectError);

        return Promise.reject(rejectError);
    }
);

// 处理请求数据，确保ID为字符串
function processRequestData(data) {
    if (!data || typeof data !== 'object') return data;
    if (isFormData(data)) return data;

    const process = (obj) => {
        if (Array.isArray(obj)) {
            return obj.map(item => process(item));
        }

        if (obj && typeof obj === 'object') {
            const result = { ...obj };
            Object.keys(result).forEach(key => {
                const value = result[key];
                if (key.includes('id') || key.includes('Id') || key.includes('ID')) {
                    if (typeof value === 'number') {
                        result[key] = String(value);
                    }
                }
                if (value && typeof value === 'object') {
                    result[key] = process(value);
                }
            });
            return result;
        }

        return obj;
    };

    return process(data);
}

// 处理响应数据，确保ID为字符串
function processResponseData(data) {
    if (!data || typeof data !== 'object') return data;

    const process = (obj) => {
        if (Array.isArray(obj)) {
            return obj.map(item => process(item));
        }

        if (obj && typeof obj === 'object') {
            const result = { ...obj };
            Object.keys(result).forEach(key => {
                const value = result[key];
                if (key.includes('id') || key.includes('Id') || key.includes('ID')) {
                    if (typeof value === 'number' || (typeof value === 'string' && /^\d+$/.test(value))) {
                        result[key] = String(value);
                    }
                }
                if (value && typeof value === 'object') {
                    result[key] = process(value);
                }
            });
            return result;
        }

        return obj;
    };

    return process(data);
}

// 添加取消请求的方法
export const cancelRequest = (config) => {
    const requestKey = generateRequestKey(config);
    if (pendingRequests.has(requestKey)) {
        pendingRequests.delete(requestKey);
        const timer = requestTimers.get(requestKey);
        if (timer) {
            clearTimeout(timer);
            requestTimers.delete(requestKey);
        }
    }
};

// 清除所有待处理请求
export const clearAllPendingRequests = () => {
    pendingRequests.clear();
    requestTimers.forEach(timer => clearTimeout(timer));
    requestTimers.clear();
};

export default request;
