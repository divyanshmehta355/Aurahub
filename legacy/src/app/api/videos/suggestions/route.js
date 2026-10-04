import { NextResponse } from "next/server";
import dbConnect from "@/lib/dbConnect";
import Video from "@/models/Video";
import mongoose from "mongoose";
import redis from "@/lib/redis";

export const dynamic = "force-dynamic";

export async function GET(request) {
  try {
    const { searchParams } = request.nextUrl;
    const page = parseInt(searchParams.get("page") || "1");
    const limit = parseInt(searchParams.get("limit") || "10");
    const currentVideoId = searchParams.get("exclude");

    const cacheKey = `suggestions:p${page}:exclude:${currentVideoId || "none"}`;

    const cachedSuggestions = await redis.get(cacheKey);
    if (cachedSuggestions) {
      const data =
        typeof cachedSuggestions === "string"
          ? JSON.parse(cachedSuggestions)
          : cachedSuggestions;
      return NextResponse.json(data);
    }

    await dbConnect();

    let currentVideo = null;
    if (currentVideoId && mongoose.Types.ObjectId.isValid(currentVideoId)) {
      currentVideo = await Video.findById(currentVideoId).select(
        "+embedding tags category"
      );
    }

    const skip = (page - 1) * limit;

    // Try AI Vector Search if the current video has an embedding
    if (
      currentVideo &&
      Array.isArray(currentVideo.embedding) &&
      currentVideo.embedding.length > 0
    ) {
      try {
        const filter = {
          visibility: { $eq: "public" },
          _id: { $ne: currentVideo._id },
        };

        const vectorPipeline = [
          {
            $vectorSearch: {
              index: "vector_index",
              path: "embedding",
              queryVector: currentVideo.embedding,
              numCandidates: 100,
              limit: limit * 2,
              filter,
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
            $addFields: {
              uploader: {
                _id: "$uploaderDetails._id",
                username: "$uploaderDetails.username",
                avatar: "$uploaderDetails.avatar",
              },
              score: {
                $add: [
                  { $multiply: ["$vectorScore", 100] },
                  // Boost if same category
                  {
                    $cond: [
                      { $eq: ["$category", currentVideo.category] },
                      15,
                      0,
                    ],
                  },
                ],
              },
            },
          },
          { $project: { uploaderDetails: 0, embedding: 0 } },
          { $sort: { score: -1, views: -1, createdAt: -1 } },
          { $skip: skip },
          { $limit: limit },
        ];

        const vectorResults = await Video.aggregate(vectorPipeline);
        if (vectorResults && vectorResults.length > 0) {
          const totalVideos = await Video.countDocuments({
            visibility: "public",
            _id: { $ne: currentVideo._id },
          });

          const responseData = {
            videos: vectorResults,
            currentPage: page,
            totalPages: Math.ceil(totalVideos / limit),
            isAiVectorSearch: true,
          };

          await redis.set(cacheKey, JSON.stringify(responseData), { ex: 300, tags: ['suggestions'] });
          return NextResponse.json(responseData);
        }
      } catch (vectorSearchError) {
        // Graceful fallback to standard category/tags pipeline
      }
    }

    // Standard / Fallback Pipeline
    const pipeline = [];
    const baseMatch = { visibility: "public" };
    if (currentVideoId && mongoose.Types.ObjectId.isValid(currentVideoId)) {
      baseMatch._id = { $ne: new mongoose.Types.ObjectId(currentVideoId) };
    }
    pipeline.push({ $match: baseMatch });

    // Scoring logic (Context-aware based on current video + views)
    pipeline.push({
      $addFields: {
        score: {
          $add: [
            currentVideo && currentVideo.category
              ? {
                  $cond: [
                    { $eq: ["$category", currentVideo.category] },
                    20,
                    0,
                  ],
                }
              : 0,

            currentVideo && currentVideo.tags
              ? {
                  $multiply: [
                    {
                      $size: {
                        $setIntersection: [
                          { $ifNull: ["$tags", []] },
                          currentVideo.tags,
                        ],
                      },
                    },
                    15,
                  ],
                }
              : 0,
          ],
        },
      },
    });

    pipeline.push({
      $sort: { score: -1, views: -1, createdAt: -1 },
    });

    const facetPipeline = [
      ...pipeline,
      {
        $facet: {
          metadata: [{ $count: "total" }],
          data: [
            { $skip: skip },
            { $limit: limit },
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
                embedding: 0,
              },
            },
          ],
        },
      },
    ];

    const results = await Video.aggregate(facetPipeline);
    const totalVideos = results[0].metadata[0]?.total || 0;
    const videos = results[0].data;

    const responseData = {
      videos,
      currentPage: page,
      totalPages: Math.ceil(totalVideos / limit),
      isAiVectorSearch: false,
    };

    await redis.set(cacheKey, JSON.stringify(responseData), { ex: 300, tags: ['suggestions'] });
    return NextResponse.json(responseData);
  } catch (error) {
    console.error("Error fetching suggestions:", error);
    return NextResponse.json({ message: "Server error" }, { status: 500 });
  }
}