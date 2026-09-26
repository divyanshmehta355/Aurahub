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
        const limit = parseInt(searchParams.get('limit') || '8');
        const category = searchParams.get('category');
        const type = searchParams.get('type') || 'all';

        const cacheKey = `videos_v5:${category || 'all'}:${sortOption}:p${page}:l${limit}:t${type}`;

        const cachedData = await redis.get(cacheKey);

        if (cachedData) {
            const parsed = typeof cachedData === 'string' ? JSON.parse(cachedData) : cachedData;
            return NextResponse.json(parsed, {
                headers: {
                    'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=15',
                },
            });
        }
        
        await dbConnect();
        
        const skip = (page - 1) * limit;
        const sortCriteria = {
            'trending': { trendingScore: -1, createdAt: -1, _id: -1 },
            'date_desc': { createdAt: -1, _id: -1 },
            'views_desc': { views: -1, createdAt: -1, _id: -1 },
            'likes_desc': { likesCount: -1, createdAt: -1, _id: -1 },
            'comments_desc': { commentCount: -1, createdAt: -1, _id: -1 }
        }[sortOption] || { trendingScore: -1, createdAt: -1, _id: -1 };

        const filter = { visibility: 'public', streamtapeStatus: { $ne: 'dead' } };
        if (category && category !== "All") {
            filter.category = category;
        }
        
        if (type === 'short') {
            filter.isShort = true;
        } else if (type === 'standard') {
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
            totalVideos,
        };

        const cacheExpiry = 30;
        await redis.set(cacheKey, JSON.stringify(responseData), { EX: cacheExpiry });

        return NextResponse.json(responseData, {
            headers: {
                'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=15',
            },
        });

    } catch (error) {
        console.error('Error fetching videos:', error);
        return NextResponse.json({ message: 'Failed to fetch videos' }, { status: 500 });
    }
}