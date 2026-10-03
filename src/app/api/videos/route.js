import { NextResponse } from 'next/server';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import { buildVideoAggregation } from '@/lib/videoUtils';
import redis from '@/lib/redis';
import mongoose from 'mongoose';

const RANDOM_FEED_TTL_SECONDS = 300;

function shuffle(items) {
    for (let index = items.length - 1; index > 0; index--) {
        const randomIndex = Math.floor(Math.random() * (index + 1));
        [items[index], items[randomIndex]] = [items[randomIndex], items[index]];
    }
    return items;
}

export async function GET(request) {
    try {
        const { searchParams } = request.nextUrl;
        let sortOption = searchParams.get('sort') || 'random';
        const page = parseInt(searchParams.get('page') || '1');
        const limit = parseInt(searchParams.get('limit') || '8');
        const category = searchParams.get('category');
        const type = searchParams.get('type') || 'all';

        const filter = { visibility: 'public', streamtapeStatus: { $ne: 'dead' } };
        if (category && category !== "All") {
            filter.category = category;
        }

        if (type === 'short') {
            filter.isShort = true;
        } else if (type === 'standard') {
            filter.isShort = { $ne: true };
        }

        let randomOrder = null;
        if (sortOption === 'random') {
            const orderKey = `videos_random_order_v1:${category || 'all'}:${type}`;
            const cachedOrder = await redis.get(orderKey);
            if (cachedOrder) {
                randomOrder = typeof cachedOrder === 'string' ? JSON.parse(cachedOrder) : cachedOrder;
            }

            if (!randomOrder || !Array.isArray(randomOrder.ids)) {
                await dbConnect();
                const ids = (await Video.distinct('_id', filter)).map((id) => id.toString());
                randomOrder = {
                    seed: `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`,
                    ids: shuffle(ids),
                };
                const savedOrder = await redis.set(orderKey, JSON.stringify(randomOrder), {
                    ex: RANDOM_FEED_TTL_SECONDS,
                    tags: ['videos'],
                });
                if (!savedOrder) {
                    randomOrder = null;
                    sortOption = 'newest';
                }
            }
        }

        const sortCacheKey = randomOrder ? `random-${randomOrder.seed}` : sortOption;
        const cacheKey = `videos_v5:${category || 'all'}:${sortCacheKey}:p${page}:l${limit}:t${type}`;
        const cachedData = await redis.get(cacheKey);
        if (cachedData) {
            const parsed = typeof cachedData === 'string' ? JSON.parse(cachedData) : cachedData;
            return NextResponse.json(parsed, {
                headers: { 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=15' },
            });
        }

        await dbConnect();

        const skip = (page - 1) * limit;
        const sortCriteria = {
            'trending': { trendingScore: -1, createdAt: -1, _id: -1 },
            'newest': { createdAt: -1, _id: -1 },
            'views': { views: -1, createdAt: -1, _id: -1 },
            'likes': { likesCount: -1, createdAt: -1, _id: -1 },
            'comments': { commentCount: -1, createdAt: -1, _id: -1 }
        }[sortOption] || { trendingScore: -1, createdAt: -1, _id: -1 };

        let videos;
        let totalVideos;
        if (randomOrder) {
            const pageIds = randomOrder.ids.slice(skip, skip + limit);
            const objectIds = pageIds.map((id) => new mongoose.Types.ObjectId(id));
            const orderedFilter = { ...filter, _id: { $in: objectIds } };
            const pageVideos = pageIds.length > 0
                ? await Video.aggregate(buildVideoAggregation(orderedFilter, { createdAt: -1 }))
                : [];
            const videosById = new Map(pageVideos.map((video) => [video._id.toString(), video]));
            videos = pageIds.map((id) => videosById.get(id)).filter(Boolean);
            totalVideos = randomOrder.ids.length;
        } else {
            const aggregation = buildVideoAggregation(filter, sortCriteria, { skip, limit });
            [totalVideos, videos] = await Promise.all([
                Video.countDocuments(filter),
                Video.aggregate(aggregation),
            ]);
        }

        const responseData = {
            videos,
            currentPage: page,
            totalPages: Math.ceil(totalVideos / limit),
            totalVideos,
        };

        const cacheExpiry = 30;
        await redis.set(cacheKey, JSON.stringify(responseData), { EX: cacheExpiry, tags: ['videos'] });

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