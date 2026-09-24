import { NextResponse } from "next/server";
import dbConnect from "@/lib/dbConnect";
import Video from "@/models/Video";
import User from "@/models/User";
import { correctSearchQuery } from "@/lib/gemini";

export const dynamic = "force-dynamic";

export async function GET(request) {
  try {
    const { searchParams } = new URL(request.url);
    const query = searchParams.get("q");

    if (!query || query.trim().length < 2) {
      return NextResponse.json({ videos: [], users: [] });
    }

    await dbConnect();

    const cleanQuery = query.trim();
    const words = cleanQuery.split(/\s+/).filter(Boolean);
    const flexiblePattern = words.join(".*");
    const regex = new RegExp(flexiblePattern, "i");

    // Fetch up to 4 matching public videos (matching title or tags)
    const videoPromise = Video.find({
      visibility: "public",
      $or: [
        { title: { $regex: regex } },
        { tags: { $in: [new RegExp(cleanQuery, "i")] } },
      ],
    })
      .select("_id title thumbnailUrl")
      .limit(4)
      .lean();

    // Fetch up to 2 matching users
    const userPromise = User.find({
      $or: [
        { username: { $regex: regex } },
        { name: { $regex: regex } },
      ],
    })
      .select("username name image")
      .limit(2)
      .lean();

    let [videos, users] = await Promise.all([videoPromise, userPromise]);

    // If zero videos found and query is long enough, try AI typo correction
    if (videos.length === 0 && cleanQuery.length >= 4) {
      try {
        const didYouMean = await correctSearchQuery(cleanQuery);
        if (didYouMean && didYouMean.toLowerCase() !== cleanQuery.toLowerCase()) {
          const correctedWords = didYouMean.split(/\s+/).filter(Boolean);
          const correctedPattern = correctedWords.join(".*");
          const correctedRegex = new RegExp(correctedPattern, "i");

          const fallbackVideos = await Video.find({
            visibility: "public",
            $or: [
              { title: { $regex: correctedRegex } },
              { tags: { $in: [new RegExp(didYouMean, "i")] } },
            ],
          })
            .select("_id title thumbnailUrl")
            .limit(4)
            .lean();

          if (fallbackVideos.length > 0) {
            videos = fallbackVideos;
          }
        }
      } catch (aiErr) {
        // Non-blocking fallback
      }
    }

    return NextResponse.json({ videos, users });
  } catch (error) {
    console.error("Autocomplete Error:", error);
    return NextResponse.json(
      { message: "Internal Server Error" },
      { status: 500 }
    );
  }
}
