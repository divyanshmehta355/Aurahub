import { createEnv } from "@t3-oss/env-nextjs";
import { z } from "zod";

const rawEnv = createEnv({
  server: {
    NODE_ENV: z
      .enum(["development", "test", "production"])
      .default("development"),
    REDIS_URL: z.string().min(1, "REDIS_URL is required"),
    MONGO_URI: z.string().min(1, "MONGO_URI is required"),
    NEXTAUTH_SECRET: z.string().min(1, "NEXTAUTH_SECRET is required"),
    NEXTAUTH_URL: z.string().optional(),
    FREEIMAGE_API_KEY: z.string().optional(),
    GITHUB_CLIENT_ID: z.string().optional(),
    GITHUB_CLIENT_SECRET: z.string().optional(),
    GOOGLE_CLIENT_ID: z.string().optional(),
    GOOGLE_CLIENT_SECRET: z.string().optional(),
    JWT_SECRET: z.string().optional(),
    NOTIFICATION_SERVER_URL: z.string().optional(),
    UPLOAD_FOLDER_ID: z.string().optional(),
    GEMINI_API_KEY: z.string().optional(),
  },
  client: {
    NEXT_PUBLIC_NOTIFICATION_SERVER_URL: z.string().optional(),
  },
  experimental__runtimeEnv: {
    NEXT_PUBLIC_NOTIFICATION_SERVER_URL:
      process.env.NEXT_PUBLIC_NOTIFICATION_SERVER_URL,
  },
  emptyStringAsUndefined: true,
  skipValidation: !!process.env.SKIP_ENV_VALIDATION,
});

/**
 * Deeply frozen and tamper-proof proxy for environment variables.
 * - Prevents runtime mutation, deletion, or re-assignment.
 * - Automatically blocks client-side code from accessing server-side secrets.
 */
export const env = new Proxy(rawEnv, {
  get(target, prop, receiver) {
    return Reflect.get(target, prop, receiver);
  },
  set(target, prop) {
    throw new TypeError(
      `[Security Error] Cannot modify environment variable "${String(prop)}". Environment variables are frozen and immutable.`
    );
  },
  deleteProperty(target, prop) {
    throw new TypeError(
      `[Security Error] Cannot delete environment variable "${String(prop)}". Environment variables are frozen and immutable.`
    );
  },
  defineProperty(target, prop) {
    throw new TypeError(
      `[Security Error] Cannot define property on environment variable "${String(prop)}". Environment variables are frozen and immutable.`
    );
  },
});
