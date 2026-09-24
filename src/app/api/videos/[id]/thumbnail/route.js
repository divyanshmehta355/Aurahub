import { NextResponse } from 'next/server';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import redis from '@/lib/redis';
import axios from 'axios';
import mongoose from 'mongoose';
import { getFallbackThumbnailUrl } from '@/lib/thumbnailSvg';

const AURA_API_BASE_URL = "https://aurahub-api.ashwathama249.workers.dev";
const OLD_STATIC_FALLBACK = 'https://iili.io/Ku93A2n.png';

export async function GET(request, { params }) {
    try {
        const { id } = await params;
        if (!mongoose.Types.ObjectId.isValid(id)) {
            return NextResponse.json({ message: "Invalid video ID format." }, { status: 400 });
        }

        const cacheKey = `thumbnail:${id}`;
        
        const cachedUrl = await redis.get(cacheKey);
        if (cachedUrl && cachedUrl !== OLD_STATIC_FALLBACK) {
            return NextResponse.json({ thumbnailUrl: cachedUrl });
        }
        
        await dbConnect();
        const video = await Video.findById(id).select('fileId thumbnailUrl title category');
        if (!video) {
            return NextResponse.json({ message: "Video not found" }, { status: 404 });
        }
        
        let finalUrl = getFallbackThumbnailUrl(video._id.toString(), video.title, video.category);

        if (video.thumbnailUrl) {
            finalUrl = video.thumbnailUrl;
        } else if (video.fileId) {
            try {
                const response = await axios.get(`${AURA_API_BASE_URL}/fs/files/thumbnail/${video.fileId}`);
                if (response.data?.thumbnail_url) {
                    finalUrl = response.data.thumbnail_url;
                }
            } catch (apiError) {
                console.error(`AuraHub thumbnail fetch failed for ${video.fileId}, using dynamic fallback.`);
            }
        }
        
        await redis.set(cacheKey, finalUrl, { ex: 86400 });

        return NextResponse.json({ thumbnailUrl: finalUrl });

    } catch (error) {
        console.error("Error fetching thumbnail URL:", error);
        return NextResponse.json({ message: "Server error" }, { status: 500 });
    }
}