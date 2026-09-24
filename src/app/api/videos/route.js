import { NextResponse } from 'next/server';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import { buildVideoAggregation } from '@/lib/videoUtils';
import redis from '@/lib/redis';

export async function GET(request) {
    try {
        const { searchParams } = request.nextUrl;
        const sortOption = searchParams.get('sort') || 'trending';
        const page = parseInt(searchParams.get('page') || '1');
        const limit = parseInt(searchParams.get('limit') || '12');
        const category = searchParams.get('category');
        const type = searchParams.get('type') || 'standard';

        const cacheKey = `videos_v2:${category || 'all'}:${sortOption}:p${page}:t${type}`;

        const cachedData = await redis.get(cacheKey);

        if (cachedData) {
            const parsed = typeof cachedData === 'string' ? JSON.parse(cachedData) : cachedData;
            return NextResponse.json(parsed);
        }
        
        await dbConnect();
        
        const skip = (page - 1) * limit;
        const sortCriteria = {
            'trending': { trendingScore: -1 },
            'date_desc': { createdAt: -1 },
            'views_desc': { views: -1 },
            'likes_desc': { likesCount: -1 },
            'comments_desc': { commentCount: -1 }
        }[sortOption] || { trendingScore: -1 };

        const filter = { visibility: 'public' };
        if (category && category !== "All") {
            filter.category = category;
        }
        
        if (type === 'short') {
            filter.isShort = true;
        } else {
            filter.isShort = { $ne: true };
        }

        const aggregation = buildVideoAggregation(filter, sortCriteria, { skip, limit });

        const [totalVideos, videos] = await Promise.all([
            Video.countDocuments(filter),
            Video.aggregate(aggregation)
        ]);

        const responseData = {
            videos,
            currentPage: page,
            totalPages: Math.ceil(totalVideos / limit),
        };

        const cacheExpiry = type === 'short' ? 30 : 180;
        await redis.set(cacheKey, JSON.stringify(responseData), { ex: cacheExpiry });

        return NextResponse.json(responseData);

    } catch (error) {
        console.error('Error fetching videos:', error);
        return NextResponse.json({ message: 'Failed to fetch videos' }, { status: 500 });
    }
}