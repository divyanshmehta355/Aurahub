import { Redis } from '@upstash/redis';

let rawRedis = null;

try {
    if (process.env.UPSTASH_REDIS_REST_URL && process.env.UPSTASH_REDIS_REST_TOKEN) {
        rawRedis = Redis.fromEnv();
    }
} catch {
    rawRedis = null;
}

const safeRedis = {
    get: async (key) => {
        if (!rawRedis) return null;
        try {
            return await rawRedis.get(key);
        } catch {
            return null;
        }
    },
    set: async (...args) => {
        if (!rawRedis) return null;
        try {
            return await rawRedis.set(...args);
        } catch {
            return null;
        }
    },
    del: async (...args) => {
        if (!rawRedis) return null;
        try {
            return await rawRedis.del(...args);
        } catch {
            return null;
        }
    }
};

export default safeRedis;