import { NextResponse } from "next/server";
import dbConnect from "@/lib/dbConnect";
import Video from "@/models/Video";
import { generateVideoEmbedding } from "@/lib/gemini";
import { getServerSession } from "next-auth/next";
import { authOptions } from "@/app/api/auth/[...nextauth]/route";

export const dynamic = "force-dynamic";

export async function POST(request) {
  await dbConnect();
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ message: "Unauthorized" }, { status: 401 });
    }

    if (!process.env.GEMINI_API_KEY) {
      return NextResponse.json(
        { message: "GEMINI_API_KEY is not configured in .env.local" },
        { status: 400 }
      );
    }

    // Find videos lacking embeddings
    const videos = await Video.find({
      $or: [
        { embedding: { $exists: false } },
        { embedding: null },
        { embedding: { $size: 0 } },
      ],
    })
      .select("+embedding")
      .limit(20);

    let updatedCount = 0;
    const errors = [];

    for (const video of videos) {
      try {
        const embedding = await generateVideoEmbedding(video);
        if (embedding && embedding.length > 0) {
          video.embedding = embedding;
          await video.save();
          updatedCount++;
        }
      } catch (err) {
        errors.push({ id: video._id, error: err.message });
      }
    }

    const remaining = await Video.countDocuments({
      $or: [
        { embedding: { $exists: false } },
        { embedding: null },
        { embedding: { $size: 0 } },
      ],
    });

    return NextResponse.json({
      success: true,
      processed: updatedCount,
      remaining,
      errors: errors.length > 0 ? errors : undefined,
    });
  } catch (error) {
    console.error("Backfill error:", error);
    return NextResponse.json(
      { message: "Error backfilling embeddings" },
      { status: 500 }
    );
  }
}
