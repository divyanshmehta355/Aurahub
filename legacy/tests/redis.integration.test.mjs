import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import { Worker } from "node:worker_threads";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import test from "node:test";

const workerSource = `
  const { parentPort, workerData } = require("node:worker_threads");

  (async () => {
    const { default: redis } = await import(workerData.redisModuleUrl);
    await redis.getClient();
    parentPort.postMessage({ type: "ready" });

    parentPort.on("message", async ({ id, action, key, tag }) => {
      try {
        let result;
        if (action === "warm") {
          await redis.set(key, "cached-value", { ex: 60, tags: [tag] });
          result = await redis.get(key);
        } else if (action === "warmDirect") {
          await redis.set(key, "cached-value", { ex: 60 });
          result = await redis.get(key);
        } else if (action === "invalidate") {
          await redis.invalidateTags(tag);
          result = true;
        } else if (action === "deleteKey") {
          result = await redis.del(key);
        } else if (action === "get") {
          result = await redis.get(key);
        }
        parentPort.postMessage({ id, result });
      } catch (error) {
        parentPort.postMessage({ id, error: error.message });
      }
    });
  })().catch((error) => {
    parentPort.postMessage({ type: "init-error", error: error.message });
  });
`;

function startWorker(redisModuleUrl) {
  const worker = new Worker(workerSource, {
    eval: true,
    workerData: { redisModuleUrl },
  });
  let nextId = 0;
  const pending = new Map();
  let resolveReady;
  let rejectReady;
  const ready = new Promise((resolve, reject) => {
    resolveReady = resolve;
    rejectReady = reject;
  });

  worker.on("message", (message) => {
    if (message.type === "ready") {
      resolveReady();
      return;
    }
    if (message.type === "init-error") {
      rejectReady(new Error(message.error));
      return;
    }
    const request = pending.get(message.id);
    if (!request) return;
    pending.delete(message.id);
    if (message.error) request.reject(new Error(message.error));
    else request.resolve(message.result);
  });
  worker.on("error", rejectReady);

  return {
    worker,
    ready,
    request(action, data = {}) {
      const id = ++nextId;
      return new Promise((resolve, reject) => {
        pending.set(id, { resolve, reject });
        worker.postMessage({ id, action, ...data });
      });
    },
  };
}

const redisModuleUrl = pathToFileURL(resolve("src/lib/redis.js")).href;

test(
  "tag invalidation clears L1 caches in other processes",
  { skip: !process.env.REDIS_URL },
  async () => {
    const suffix = randomUUID();
    const key = `integration:redis:${suffix}`;
    const directKey = `${key}:direct`;
    const tag = `integration:${suffix}`;
    const cacheWorker = startWorker(redisModuleUrl);
    const invalidatorWorker = startWorker(redisModuleUrl);

    try {
      await Promise.all([cacheWorker.ready, invalidatorWorker.ready]);
      assert.equal(await cacheWorker.request("warm", { key, tag }), "cached-value");
      assert.equal(await invalidatorWorker.request("invalidate", { tag }), true);

      const deadline = Date.now() + 3000;
      let cachedValue;
      do {
        cachedValue = await cacheWorker.request("get", { key });
        if (cachedValue === null) break;
        await new Promise((resolve) => setTimeout(resolve, 25));
      } while (Date.now() < deadline);

      assert.equal(cachedValue, null, "peer process should observe tag invalidation");

      assert.equal(await cacheWorker.request("warmDirect", { key: directKey }), "cached-value");
      assert.equal(await invalidatorWorker.request("deleteKey", { key: directKey }), 1);

      const directDeadline = Date.now() + 3000;
      let directCachedValue;
      do {
        directCachedValue = await cacheWorker.request("get", { key: directKey });
        if (directCachedValue === null) break;
        await new Promise((resolve) => setTimeout(resolve, 25));
      } while (Date.now() < directDeadline);

      assert.equal(directCachedValue, null, "peer process should observe direct-key deletion");
    } finally {
      await Promise.all([
        cacheWorker.worker.terminate(),
        invalidatorWorker.worker.terminate(),
      ]);
    }
  }
);
