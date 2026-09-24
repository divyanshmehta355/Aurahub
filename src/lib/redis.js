import { Redis } from '@upstash/redis';

let rawRedis = null;

try {
  if (process.env.UPSTASH_REDIS_REST_URL && process.env.UPSTASH_REDIS_REST_TOKEN) {
    rawRedis = Redis.fromEnv();
  }
} catch {
  rawRedis = null;
}

// In-Memory L1 Cache Layer to eliminate external network latency
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
    // Evict oldest item
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

    // 2. Fallback to Upstash Redis (L2)
    if (!rawRedis) return null;
    try {
      const val = await rawRedis.get(key);
      if (val !== null && val !== undefined) {
        setInMemory(key, val, 60);
      }
      return val;
    } catch {
      return null;
    }
  },

  set: async (key, value, options) => {
    // 1. Write to L1 in-memory cache immediately
    const ttl = options && typeof options.ex === 'number' ? options.ex : 60;
    setInMemory(key, value, ttl);

    // 2. Persist to Upstash Redis (L2)
    if (!rawRedis) return null;
    try {
      return await rawRedis.set(key, value, options);
    } catch {
      return null;
    }
  },

  del: async (...args) => {
    for (const k of args) {
      deleteFromMemory(k);
    }
    if (!rawRedis) return null;
    try {
      return await rawRedis.del(...args);
    } catch {
      return null;
    }
  },
};

export default safeRedis;