import { NextResponse } from "next/server";
import redis from "@/lib/redis";

export const dynamic = "force-dynamic";

/**
 * Resolves a Streamtape fileId into a direct video stream URL
 * and redirects the browser's video player to it.
 */
export async function GET(request, { params }) {
  try {
    const { id } = await params;
    if (!id || typeof id !== "string") {
      return new NextResponse("Invalid file ID", { status: 400 });
    }

    const cleanId = id.trim();
    const cacheKey = `stream_url:${cleanId}`;

    // Check Redis cache first
    try {
      const cachedUrl = await redis.get(cacheKey);
      if (cachedUrl && typeof cachedUrl === "string") {
        return NextResponse.redirect(cachedUrl, 302);
      }
    } catch (e) {
      // Non-blocking cache error
    }

    // Fetch Streamtape embed page to extract direct stream link
    const embedUrl = `https://streamtape.com/e/${cleanId}`;
    const response = await fetch(embedUrl, {
      headers: {
        "User-Agent":
          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      },
    });

    if (!response.ok) {
      return new NextResponse("Stream source unavailable", { status: 502 });
    }

    const html = await response.text();

    // Match the obfuscated robotlink in Streamtape embed
    const robotMatch = html.match(
      /document\.getElementById\(['"]robotlink['"]\)\.innerHTML\s*=\s*(.+?);/
    );

    if (!robotMatch || !robotMatch[1]) {
      return new NextResponse("Failed to resolve stream link", { status: 502 });
    }

    // Safely parse the concatenation expression
    let rawExpr = robotMatch[1].trim();
    let streamUrl = "";

    try {
      // Evaluate the pure string expression (e.g. '//stre' + ('xcdamtape...').substring(2).substring(1))
      const evaluated = Function(`"use strict"; return (${rawExpr});`)();
      streamUrl = evaluated.startsWith("//") ? `https:${evaluated}` : evaluated;
    } catch (evalErr) {
      console.error("Error evaluating stream URL expression:", evalErr);
      return new NextResponse("Error parsing stream URL", { status: 500 });
    }

    if (!streamUrl || !streamUrl.startsWith("http")) {
      return new NextResponse("Invalid stream URL resolved", { status: 500 });
    }

    // Cache the resolved URL for 3 hours (Streamtape tokens expire after 4 hours)
    try {
      await redis.set(cacheKey, streamUrl, { ex: 10800 });
    } catch (cacheErr) {
      // Non-blocking
    }

    // Redirect client video element to direct mp4 stream
    return NextResponse.redirect(streamUrl, 302);
  } catch (error) {
    console.error("Stream route error:", error);
    return new NextResponse("Internal Server Error", { status: 500 });
  }
}
