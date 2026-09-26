import { NextResponse } from "next/server";
import dbConnect from "@/lib/dbConnect";
import UserActivity from "@/models/UserActivity";
import Video from "@/models/Video";
import User from "@/models/User";
import { getServerSession } from "next-auth/next";
import { authOptions } from "@/app/api/auth/[...nextauth]/route";
import redis from "@/lib/redis";
import mongoose from "mongoose";
import { computeUserTasteVector } from "@/lib/gemini";

export const dynamic = "force-dynamic";

export async function GET(request) {
  await dbConnect();
  try {
    const session = await getServerSession(authOptions);
    const { searchParams } = request.nextUrl;
    const currentVideoId = searchParams.get("exclude");
    const page = parseInt(searchParams.get("page") || "1");
    const limit = parseInt(searchParams.get("limit") || "10");
    const skip = (page - 1) * limit;

    const userId = session?.user?.id || "anonymous";
    const cacheKey = `recommendations:${userId}:p${page}:exclude:${
      currentVideoId || "none"
    }`;

    const cachedRecs = await redis.get(cacheKey);
    if (cachedRecs) {
      const data = typeof cachedRecs === "string" ? JSON.parse(cachedRecs) : cachedRecs;
      return NextResponse.json(data);
    }

    let user = null;
    let interactedVideoIds = [];
    let favoriteTags = [];
    let favoriteCategories = [];
    let userTasteVector = null;

    if (session?.user?.id) {
      // Fetch recent interactions (views & likes) directly using index
      const recentActivities = await UserActivity.find({ userId: session.user.id })
        .sort({ createdAt: -1 })
        .limit(20)
        .lean();

        interactedVideoIds = recentActivities.map((a) => a.videoId);

        // Fetch videos with embeddings
        const interactedVideos = await Video.find({
          _id: { $in: interactedVideoIds },
        })
          .select("+embedding tags category")
          .lean();

        const videoMap = new Map();
        interactedVideos.forEach((v) => videoMap.set(v._id.toString(), v));

        const tagScores = {};
        const categoryScores = {};
        const vectorInteractions = [];

        recentActivities.forEach((activity) => {
          const video = videoMap.get(activity.videoId.toString());
          if (!video) return;

          const weight = activity.interactionType === "like" ? 3.0 : 1.0;

          if (video.category) {
            categoryScores[video.category] =
              (categoryScores[video.category] || 0) + weight;
          }
          if (video.tags) {
            video.tags.forEach((tag) => {
              tagScores[tag] = (tagScores[tag] || 0) + weight;
            });
          }
          if (Array.isArray(video.embedding) && video.embedding.length > 0) {
            vectorInteractions.push({ embedding: video.embedding, weight });
          }
        });

        favoriteTags = Object.keys(tagScores)
          .sort((a, b) => tagScores[b] - tagScores[a])
          .slice(0, 5);

        favoriteCategories = Object.keys(categoryScores)
          .sort((a, b) => categoryScores[b] - categoryScores[a])
          .slice(0, 3);

        // Compute AI taste vector
        userTasteVector = computeUserTasteVector(vectorInteractions);
    }

    let currentVideo = null;
    if (currentVideoId && mongoose.Types.ObjectId.isValid(currentVideoId)) {
      currentVideo = await Video.findById(currentVideoId).select(
        "+embedding tags category"
      );
    }

    // Try MongoDB Atlas $vectorSearch if userTasteVector exists
    if (userTasteVector && Array.isArray(userTasteVector) && userTasteVector.length > 0) {
      try {
        const filter = { visibility: { $eq: "public" } };
        if (currentVideoId && mongoose.Types.ObjectId.isValid(currentVideoId)) {
          filter._id = { $ne: new mongoose.Types.ObjectId(currentVideoId) };
        }

        const vectorPipeline = [
          {
            $vectorSearch: {
              index: "vector_index",
              path: "embedding",
              queryVector: userTasteVector,
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
                  user?.subscriptions?.length
                    ? {
                        $cond: [
                          {
                            $in: [
                              "$uploader._id",
                              user.subscriptions.map(
                                (id) => new mongoose.Types.ObjectId(id)
                              ),
                            ],
                          },
                          30,
                          0,
                        ],
                      }
                    : 0,
                  interactedVideoIds.length
                    ? {
                        $cond: [
                          {
                            $in: [
                              "$_id",
                              interactedVideoIds.map(
                                (id) => new mongoose.Types.ObjectId(id)
                              ),
                            ],
                          },
                          -30,
                          0,
                        ],
                      }
                    : 0,
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
          const totalVideos = await Video.countDocuments({ visibility: "public" });
          const responseData = {
            videos: vectorResults,
            currentPage: page,
            totalPages: Math.ceil(totalVideos / limit),
            isAiVectorSearch: true,
          };
          await redis.set(cacheKey, JSON.stringify(responseData), { ex: 300, tags: ['recommendations'] });
          return NextResponse.json(responseData);
        }
      } catch (vectorSearchError) {
        // Graceful fallback to standard pipeline if Atlas index is not ready
      }
    }

    // Standard / Fallback Pipeline
    const baseMatch = { visibility: "public" };
    if (currentVideoId && mongoose.Types.ObjectId.isValid(currentVideoId)) {
      baseMatch._id = { $ne: new mongoose.Types.ObjectId(currentVideoId) };
    }

    const pipeline = [{ $match: baseMatch }];

    // Hybrid scoring logic
    pipeline.push({
      $addFields: {
        score: {
          $add: [
            // Subscriptions boost
            user?.subscriptions?.length
              ? {
                  $cond: [
                    {
                      $in: [
                        "$uploader",
                        user.subscriptions.map(
                          (id) => new mongoose.Types.ObjectId(id)
                        ),
                      ],
                    },
                    50,
                    0,
                  ],
                }
              : 0,

            // Favorite Categories boost
            favoriteCategories.length
              ? {
                  $cond: [{ $in: ["$category", favoriteCategories] }, 30, 0],
                }
              : 0,

            // Favorite Tags boost
            favoriteTags.length
              ? {
                  $multiply: [
                    {
                      $size: {
                        $setIntersection: [
                          { $ifNull: ["$tags", []] },
                          favoriteTags,
                        ],
                      },
                    },
                    10,
                  ],
                }
              : 0,

            // Current Video Context boost
            currentVideo?.category
              ? {
                  $cond: [
                    { $eq: ["$category", currentVideo.category] },
                    20,
                    0,
                  ],
                }
              : 0,

            currentVideo?.tags?.length
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

            // Watched penalty
            interactedVideoIds.length
              ? {
                  $cond: [
                    {
                      $in: [
                        "$_id",
                        interactedVideoIds.map(
                          (id) => new mongoose.Types.ObjectId(id)
                        ),
                      ],
                    },
                    -40,
                    0,
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

    await redis.set(cacheKey, JSON.stringify(responseData), { ex: 300, tags: ['recommendations'] });
    return NextResponse.json(responseData);
  } catch (error) {
    console.error("Error fetching recommendations:", error);
    return NextResponse.json({ message: "Server error" }, { status: 500 });
  }
}
