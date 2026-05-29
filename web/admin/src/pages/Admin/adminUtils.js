export const statusText = (status) => {
    const map = {
        0: '待审核',
        1: '正常展示',
        2: '已拒绝',
        3: '已下架',
    };
    return map[Number(status)] || '未知';
};

export const roleText = (role) => {
    const map = {
        admin: '管理员',
        operator: '运营',
        user: '普通用户',
    };
    return map[role || 'user'] || '普通用户';
};

export const postTypeText = (type) => {
    const map = {
        note: '笔记',
        lost: '失物',
        question: '问答',
        guide: '攻略',
        club: '社团',
    };
    return map[type || 'note'] || '笔记';
};

export const mediaTypeText = (type) => {
    const map = {
        text: '文字',
        image: '图文',
    };
    return map[type || 'text'] || '文字';
};

export const toArrayFromLines = (value) => value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);

export const compactNumber = (value) => {
    const number = Number(value || 0);
    if (Number.isNaN(number)) return String(value || 0);
    if (number >= 10000) return `${(number / 10000).toFixed(1)}w`;
    if (number >= 1000) return `${(number / 1000).toFixed(1)}k`;
    return String(number);
};

export const percentText = (numerator, denominator) => {
    const total = Number(denominator || 0);
    if (total <= 0) return '0%';
    return `${Math.round((Number(numerator || 0) / total) * 100)}%`;
};

export const ratioText = (numerator, denominator) => {
    const total = Number(denominator || 0);
    if (total <= 0) return '0%';
    return `${Math.round((Number(numerator || 0) / total) * 100)}%`;
};

export const postCover = (post) => post?.cover_url || post?.images?.[0] || '';

export const excerpt = (value, max = 72) => {
    const text = String(value || '').replace(/\s+/g, ' ').trim();
    if (text.length <= max) return text;
    return `${text.slice(0, max)}...`;
};
