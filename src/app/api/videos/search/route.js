import { NextResponse } from "next/server";
import dbConnect from "@/lib/dbConnect";
import Video from "@/models/Video";
import { generateEmbedding, correctSearchQuery } from "@/lib/gemini";
import redis from "@/lib/redis";
import mongoose from "mongoose";

export const dynamic = "force-dynamic";

export async function GET(request) {
  await dbConnect();
  try {
    const { searchParams } = request.nextUrl;
    const rawQuery = searchParams.get("q");
    const sortOption = searchParams.get("sort") || "relevance";

    if (!rawQuery || !rawQuery.trim()) {
      return NextResponse.json({ videos: [], didYouMean: null });
    }

    const searchQuery = rawQuery.trim();
    const cacheKey = `search_ai_v1:${encodeURIComponent(searchQuery)}:${sortOption}`;

    // Check Redis cache
    try {
      const cached = await redis.get(cacheKey);
      if (cached) {
        const parsed = typeof cached === "string" ? JSON.parse(cached) : cached;
        return NextResponse.json(parsed);
      }
    } catch (e) {
      // Non-blocking cache error
    }

    // Run AI Query Spell-Check & Query Embedding in parallel
    const [didYouMean, queryVector] = await Promise.all([
      correctSearchQuery(searchQuery).catch(() => null),
      generateEmbedding(searchQuery).catch(() => null),
    ]);

    // Build words and search patterns
    const terms = [searchQuery];
    if (didYouMean && didYouMean.toLowerCase() !== searchQuery.toLowerCase()) {
      terms.push(didYouMean);
    }

    // Build flexible multi-field fuzzy regex conditions
    const orConditions = [];

    terms.forEach((term) => {
      const words = term.split(/\s+/).filter(Boolean);
      const flexiblePattern = words.join(".*");

      // Phrase / flexible pattern match
      orConditions.push(
        { title: { $regex: flexiblePattern, $options: "i" } },
        { description: { $regex: flexiblePattern, $options: "i" } },
        { category: { $regex: term, $options: "i" } },
        { tags: { $in: [new RegExp(term, "i")] } }
      );

      // Individual word token matches
      words.forEach((w) => {
        if (w.length >= 3) {
          orConditions.push(
            { title: { $regex: w, $options: "i" } },
            { tags: { $in: [new RegExp(w, "i")] } }
          );
        }
      });
    });

    const candidateMap = new Map();

    // 1. Run AI Vector Search if query embedding exists
    if (queryVector && Array.isArray(queryVector) && queryVector.length > 0) {
      try {
        const vectorResults = await Video.aggregate([
          {
            $vectorSearch: {
              index: "vector_index",
              path: "embedding",
              queryVector,
              numCandidates: 100,
              limit: 25,
              filter: { visibility: { $eq: "public" } },
            },
          },
          {
            $addFields: {
              vectorScore: { $meta: "vectorSearchScore" },
            },
          },
          {
            $lookup: {
              from: "users",
              localField: "uploader",
              foreignField: "_id",
              as: "uploaderDetails",
            },
          },
          {
            $unwind: {
              path: "$uploaderDetails",
              preserveNullAndEmptyArrays: true,
            },
          },
          {
            $project: {
              title: 1,
              description: 1,
              fileId: 1,
              thumbnailUrl: 1,
              views: 1,
              likesCount: { $size: { $ifNull: ["$likes", []] } },
              createdAt: 1,
              isShort: 1,
              vectorScore: 1,
              uploader: {
                _id: "$uploaderDetails._id",
                username: "$uploaderDetails.username",
                avatar: "$uploaderDetails.avatar",
              },
            },
          },
        ]);

        if (vectorResults && vectorResults.length > 0) {
          vectorResults.forEach((v) => {
            const vScore = (v.vectorScore || 0) * 100;
            candidateMap.set(v._id.toString(), {
              ...v,
              searchScore: vScore,
              matchType: "semantic",
            });
          });
        }
      } catch (vectorErr) {
        // Fall back gracefully if Atlas index is not ready
      }
    }

    // 2. Run Fuzzy Keyword Search
    try {
      const keywordResults = await Video.find({
        visibility: "public",
        $or: orConditions,
      })
        .populate("uploader", "username avatar")
        .limit(30)
        .lean();

      keywordResults.forEach((v) => {
        const idStr = v._id.toString();
        const existing = candidateMap.get(idStr);

        // Calculate text match score boost
        let textBoost = 40;
        const titleLower = (v.title || "").toLowerCase();
        const qLower = searchQuery.toLowerCase();

        if (titleLower === qLower) {
          textBoost += 60; // Exact match
        } else if (titleLower.includes(qLower)) {
          textBoost += 30; // Direct substring
        }

        if (existing) {
          // Both vector and keyword matched!
          existing.searchScore += textBoost;
          existing.matchType = "hybrid";
        } else {
          candidateMap.set(idStr, {
            ...v,
            likesCount: v.likes?.length || 0,
            searchScore: textBoost,
            matchType: "keyword",
          });
        }
      });
    } catch (keywordErr) {
      console.error("Keyword search error:", keywordErr);
    }

    // 3. Fallback: If still zero results and didYouMean exists, do a broadened text search
    if (candidateMap.size === 0 && didYouMean) {
      const fallbackResults = await Video.find({
        visibility: "public",
        $or: [
          { title: { $regex: didYouMean, $options: "i" } },
          { description: { $regex: didYouMean, $options: "i" } },
        ],
      })
        .populate("uploader", "username avatar")
        .limit(20)
        .lean();

      fallbackResults.forEach((v) => {
        candidateMap.set(v._id.toString(), {
          ...v,
          likesCount: v.likes?.length || 0,
          searchScore: 30,
          matchType: "spell_corrected",
        });
      });
    }

    // 4. Sort results
    let results = Array.from(candidateMap.values());

    if (sortOption === "date_desc") {
      results.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    } else if (sortOption === "views_desc") {
      results.sort((a, b) => (b.views || 0) - (a.views || 0));
    } else if (sortOption === "likes_desc") {
      results.sort((a, b) => (b.likesCount || 0) - (a.likesCount || 0));
    } else {
      // Relevance (Default): searchScore descending, then views
      results.sort((a, b) => {
        if (b.searchScore !== a.searchScore) {
          return b.searchScore - a.searchScore;
        }
        return (b.views || 0) - (a.views || 0);
      });
    }

    const responsePayload = {
      videos: results,
      didYouMean: didYouMean && didYouMean.toLowerCase() !== searchQuery.toLowerCase() ? didYouMean : null,
      query: searchQuery,
      total: results.length,
    };

    // Cache results for 5 minutes
    try {
      await redis.set(cacheKey, JSON.stringify(responsePayload), { ex: 300 });
    } catch (e) {
      // Non-blocking
    }

    return NextResponse.json(responsePayload);
  } catch (error) {
    console.error("Error searching videos:", error);
    return NextResponse.json({ message: "Server error during search." }, { status: 500 });
  }
}