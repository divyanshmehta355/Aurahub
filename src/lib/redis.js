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
        setInMemory(key, val, 60);
      }
      return val;
    } catch {
      return null;
    }
  },

  set: async (key, value, options) => {
    // 1. Write to L1 in-memory cache immediately
    const ttl = options && (typeof options.ex === 'number' ? options.ex : typeof options.EX === 'number' ? options.EX : 60);
    setInMemory(key, value, ttl);

    // 2. Persist to Redis (L2)
    try {
      const c = await getConnectedClient();
      if (!c) return null;

      const payload = typeof value === 'string' || Buffer.isBuffer(value)
        ? value
        : JSON.stringify(value);

      if (options) {
        return await c.set(key, payload, options);
      }
      return await c.set(key, payload);
    } catch {
      return null;
    }
  },

  del: async (...args) => {
    const keys = args.flat();
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

  getClient: getConnectedClient,
};

export { createClient };
export default safeRedis;