import { createClient } from 'redis';
import { env } from '../env.mjs';

const REDIS_URL = env.REDIS_URL;

const REDIS_CONFIG = {
  url: REDIS_URL,
  pingInterval: 30000,
  socket: {
    reconnectStrategy: (retries) => {
      if (retries > 10) {
        return new Error('Redis maximum reconnect attempts reached');
      }
      return Math.min(retries * 100, 3000);
    },
  },
};

// Reuse client across module re-evaluations in Next.js development
let client = globalThis.__redisClient || null;
let connectingPromise = null;

export function getClient() {
  if (!client) {
    client = createClient(REDIS_CONFIG);
    client.on('error', (err) => {
      if (err?.message?.includes('Socket closed unexpectedly')) return;
      console.log('Redis Client Error', err);
    });
    globalThis.__redisClient = client;
  }
  return client;
}

export async function getConnectedClient() {
  const c = getClient();
  if (c.isOpen) {
    return c;
  }

  if (!connectingPromise) {
    connectingPromise = c.connect().catch((err) => {
      console.log('Redis Client Error', err);
      connectingPromise = null;
      throw err;
    }).finally(() => {
      connectingPromise = null;
    });
  }

  await connectingPromise;
  return c;
}

// In-Memory L1 Cache Layer to eliminate external network latency for ultra-fast response
const memoryCache = globalThis.__aurahub_memoryCache || (globalThis.__aurahub_memoryCache = new Map());
const MAX_MEMORY_ITEMS = 1000;

function getFromMemory(key) {
  const item = memoryCache.get(key);
  if (!item) return null;
  if (Date.now() > item.expiresAt) {
    memoryCache.delete(key);
    return null;
  }
  return item.value;
}

function setInMemory(key, value, ttlSeconds = 60) {
  if (memoryCache.size >= MAX_MEMORY_ITEMS) {
    const oldestKey = memoryCache.keys().next().value;
    if (oldestKey) memoryCache.delete(oldestKey);
  }
  memoryCache.set(key, {
    value,
    expiresAt: Date.now() + Math.min(ttlSeconds, 300) * 1000,
  });
}

function deleteFromMemory(key) {
  memoryCache.delete(key);
}

const safeRedis = {
  get: async (key) => {
    // 1. Fast path: check L1 in-memory cache
    const memVal = getFromMemory(key);
    if (memVal !== null && memVal !== undefined) {
      return memVal;
    }

    // 2. Fetch from Redis (L2)
    try {
      const c = await getConnectedClient();
      if (!c) return null;
      const val = await c.get(key);
      if (val !== null && val !== undefined) {
        let remTtl = 30;
        try {
          const ttl = await c.ttl(key);
          if (ttl > 0) {
            remTtl = ttl;
          } else if (ttl === -2) {
            return null;
          }
        } catch {}
        setInMemory(key, val, Math.min(remTtl, 60));
      }
      return val;
    } catch {
      return null;
    }
  },

  set: async (key, value, options) => {
    let ttlSeconds = 60;
    let redisOptions = undefined;

    if (typeof options === 'number') {
      ttlSeconds = options;
      redisOptions = { EX: options };
    } else if (options && typeof options === 'object') {
      const ex = typeof options.EX === 'number'
        ? options.EX
        : (typeof options.ex === 'number' ? options.ex : null);
      const px = typeof options.PX === 'number'
        ? options.PX
        : (typeof options.px === 'number' ? options.px : null);

      redisOptions = { ...options };

      if (ex !== null) {
        ttlSeconds = ex;
        redisOptions.EX = ex;
        delete redisOptions.ex;
      }
      if (px !== null) {
        ttlSeconds = Math.max(1, Math.ceil(px / 1000));
        redisOptions.PX = px;
        delete redisOptions.px;
      }

      // Default to 60s if neither EX nor PX was specified (prevents unexpired ghost keys)
      if (ex === null && px === null && !redisOptions.KEEPTTL) {
        redisOptions.EX = 60;
        ttlSeconds = 60;
      }
    } else {
      redisOptions = { EX: 60 };
      ttlSeconds = 60;
    }

    // 1. Write to L1 in-memory cache with exact TTL
    setInMemory(key, value, ttlSeconds);

    // 2. Persist to Redis (L2) with normalized uppercase EX/PX option
    try {
      const c = await getConnectedClient();
      if (!c) return null;

      const payload = typeof value === 'string' || Buffer.isBuffer(value)
        ? value
        : JSON.stringify(value);

      return await c.set(key, payload, redisOptions);
    } catch (err) {
      console.error('Redis set error:', err);
      return null;
    }
  },

  del: async (...args) => {
    const keys = args.flat().filter(Boolean);
    for (const k of keys) {
      deleteFromMemory(k);
    }
    try {
      const c = await getConnectedClient();
      if (!c || keys.length === 0) return null;
      return await c.del(keys);
    } catch {
      return null;
    }
  },

  delPattern: async (pattern) => {
    // 1. Remove matching keys from L1 memoryCache
    const regex = new RegExp('^' + pattern.replace(/([.+?^=!:${}()|\[\]\/\\])/g, '\\$1').replace(/\*/g, '.*') + '$');
    for (const k of memoryCache.keys()) {
      if (regex.test(k)) {
        memoryCache.delete(k);
      }
    }

    // 2. Remove matching keys from Redis L2
    try {
      const c = await getConnectedClient();
      if (!c) return null;

      const keysToDelete = [];
      for await (const key of c.scanIterator({ MATCH: pattern, COUNT: 100 })) {
        keysToDelete.push(key);
      }

      if (keysToDelete.length > 0) {
        for (let i = 0; i < keysToDelete.length; i += 200) {
          await c.del(keysToDelete.slice(i, i + 200));
        }
        return keysToDelete.length;
      }
      return 0;
    } catch (err) {
      console.error('Error in redis.delPattern:', err);
      return null;
    }
  },

  invalidateVideoCaches: async (videoIds) => {
    const ids = Array.isArray(videoIds) ? videoIds : (videoIds ? [videoIds] : []);

    // Invalidate all video feed listings across categories, sort orders, and pages
    await safeRedis.delPattern('videos_*');
    await safeRedis.delPattern('suggestions:*');
    await safeRedis.delPattern('recommendations:*');
    await safeRedis.delPattern('search:*');

    // Invalidate individual video caches
    for (const id of ids) {
      const idStr = id?.toString?.() || String(id);
      if (idStr) {
        await safeRedis.del(`video:${idStr}`);
        await safeRedis.del(`stream:${idStr}`);
        await safeRedis.delPattern(`*${idStr}*`);
      }
    }
  },

  getClient: getConnectedClient,
};

export { createClient };
export default safeRedis;