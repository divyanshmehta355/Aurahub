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
    const page = parseInt(searchParams.get("page") || "1");
    const limit = parseInt(searchParams.get("limit") || "10");
    const skip = (page - 1) * limit;

    const cacheKey = `feed:p${page}:l${limit}`;

    const cachedFeed = await redis.get(cacheKey);
    if (cachedFeed) {
      const data = typeof cachedFeed === "string" ? JSON.parse(cachedFeed) : cachedFeed;
      return NextResponse.json(data);
    }

    const baseMatch = { visibility: "public" };

    const pipeline = [
      { $match: baseMatch },
      { $sort: { createdAt: -1, views: -1 } },
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
              },
            },
          ],
        },
      },
    ];

    const results = await Video.aggregate(pipeline);
    const totalVideos = results[0]?.metadata[0]?.total || 0;
    const videos = results[0]?.data || [];

    const responseData = {
      videos,
      currentPage: page,
      totalPages: Math.ceil(totalVideos / limit),
      isAiVectorSearch: false,
    };

    await redis.set(cacheKey, JSON.stringify(responseData), { ex: 300, tags: ['feed'] });
    return NextResponse.json(responseData);
  } catch (error) {
    console.error("Error fetching feed:", error);
    return NextResponse.json({ message: "Server error" }, { status: 500 });
  }
}
