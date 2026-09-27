import { NextResponse } from "next/server";
import dbConnect from "@/lib/dbConnect";
import Video from "@/models/Video";
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
      return NextResponse.json({ videos: [], query: rawQuery, total: 0 });
    }

    const searchQuery = rawQuery.trim();
    const cacheKey = `search_fuzzy_v3:${encodeURIComponent(searchQuery)}:${sortOption}`;

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

    const candidateMap = new Map();
    let usedAtlasSearch = false;

    // 1. Try MongoDB Atlas Search with fuzzy matching
    try {
      const atlasPipeline = [
        {
          $search: {
            index: "default", // Assuming the default Atlas search index name
            text: {
              query: searchQuery,
              path: ["title", "description", "category", "tags"],
              fuzzy: {
                maxEdits: 2,
                prefixLength: 1,
              },
            },
          },
        },
        {
          $match: {
            visibility: "public",
            streamtapeStatus: { $ne: "dead" },
          },
        },
        { $limit: 30 },
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
          $addFields: {
            searchScore: { $meta: "searchScore" },
            likesCount: { $size: { $ifNull: ["$likes", []] } },
            uploader: {
              _id: "$uploaderDetails._id",
              username: "$uploaderDetails.username",
              avatar: "$uploaderDetails.avatar",
            },
          },
        },
        {
          $project: {
            uploaderDetails: 0,
          },
        },
      ];

      const atlasResults = await Video.aggregate(atlasPipeline);

      if (atlasResults && atlasResults.length > 0) {
        usedAtlasSearch = true;
        atlasResults.forEach((v) => {
          candidateMap.set(v._id.toString(), {
            ...v,
            matchType: "atlas_fuzzy",
            // Amplify Atlas score to match our old scale somewhat
            searchScore: (v.searchScore || 1) * 20, 
          });
        });
      }
    } catch (atlasErr) {
      // Graceful fallback if Atlas Search is not available or index 'default' does not exist
      console.warn("Atlas Search failed or index missing, falling back to regex:", atlasErr.message);
    }

    // 2. Fallback to Regex Search if Atlas Search wasn't used or yielded no results
    if (!usedAtlasSearch || candidateMap.size === 0) {
      // Build flexible multi-field fuzzy regex conditions
      const orConditions = [];
      const words = searchQuery.split(/\s+/).filter(Boolean);
      const flexiblePattern = words.join(".*");

      // Phrase / flexible pattern match
      orConditions.push(
        { title: { $regex: flexiblePattern, $options: "i" } },
        { description: { $regex: flexiblePattern, $options: "i" } },
        { category: { $regex: searchQuery, $options: "i" } },
        { tags: { $in: [new RegExp(searchQuery, "i")] } }
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

      try {
        const keywordResults = await Video.find({
          visibility: "public",
          streamtapeStatus: { $ne: "dead" },
          $or: orConditions,
        })
          .populate("uploader", "username avatar")
          .limit(30)
          .lean();

        keywordResults.forEach((v) => {
          const idStr = v._id.toString();

          // Calculate text match score boost
          let textBoost = 40;
          const titleLower = (v.title || "").toLowerCase();
          const qLower = searchQuery.toLowerCase();

          if (titleLower === qLower) {
            textBoost += 60; // Exact match
          } else if (titleLower.includes(qLower)) {
            textBoost += 30; // Direct substring
          }

          candidateMap.set(idStr, {
            ...v,
            likesCount: v.likes?.length || 0,
            searchScore: textBoost,
            matchType: "keyword",
          });
        });
      } catch (keywordErr) {
        console.error("Keyword search error:", keywordErr);
      }
    }

    // Sort results
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
      query: searchQuery,
      total: results.length,
    };

    // Cache results for 5 minutes
    try {
      await redis.set(cacheKey, JSON.stringify(responsePayload), { ex: 300, tags: ['search'] });
    } catch (e) {
      // Non-blocking
    }

    return NextResponse.json(responsePayload);
  } catch (error) {
    console.error("Error searching videos:", error);
    return NextResponse.json({ message: "Server error during search." }, { status: 500 });
  }
}