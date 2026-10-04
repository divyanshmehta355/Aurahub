import { NextResponse } from 'next/server';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import User from '@/models/User';
import Comment from '@/models/Comment';
import UserActivity from '@/models/UserActivity';
import mongoose from 'mongoose';
import redis from '@/lib/redis';
import { getServerSession } from 'next-auth/next';
import { authOptions } from '@/app/api/auth/[...nextauth]/route';

const ALLOWED_PERIODS = new Set([7, 30, 90]);

export async function GET(request) {
    try {
        const session = await getServerSession(authOptions);
        if (!session || !session.user) {
            return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
        }

        const requestedDays = Number(request.nextUrl.searchParams.get('days')) || 30;
        const days = ALLOWED_PERIODS.has(requestedDays) ? requestedDays : 30;
        const cacheKey = `creator_analytics:${session.user.id}:${days}`;
        const cachedAnalytics = await redis.get(cacheKey);
        if (cachedAnalytics) {
            const analytics = typeof cachedAnalytics === 'string'
                ? JSON.parse(cachedAnalytics)
                : cachedAnalytics;
            return NextResponse.json(analytics);
        }

        await dbConnect();

        const uploaderId = new mongoose.Types.ObjectId(session.user.id);
        const periodStart = new Date();
        periodStart.setUTCHours(0, 0, 0, 0);
        periodStart.setUTCDate(periodStart.getUTCDate() - (days - 1));

        const [user, videoIds, [videoRollups]] = await Promise.all([
            User.findById(uploaderId).select('subscribersCount').lean(),
            Video.distinct('_id', { uploader: uploaderId }),
            Video.aggregate([
                { $match: { uploader: uploaderId } },
                {
                    $project: {
                        title: 1,
                        category: 1,
                        views: { $ifNull: ['$views', 0] },
                        likesCount: { $size: { $ifNull: ['$likes', []] } },
                    },
                },
                {
                    $facet: {
                        totals: [
                            {
                                $group: {
                                    _id: null,
                                    videos: { $sum: 1 },
                                    views: { $sum: '$views' },
                                    likes: { $sum: '$likesCount' },
                                },
                            },
                        ],
                        topVideos: [
                            { $sort: { views: -1, _id: 1 } },
                            { $limit: 10 },
                            {
                                $lookup: {
                                    from: 'comments',
                                    localField: '_id',
                                    foreignField: 'video',
                                    pipeline: [{ $count: 'count' }],
                                    as: 'commentStats',
                                },
                            },
                            {
                                $project: {
                                    _id: 1,
                                    title: 1,
                                    category: 1,
                                    views: 1,
                                    likesCount: 1,
                                    commentCount: {
                                        $ifNull: [{ $arrayElemAt: ['$commentStats.count', 0] }, 0],
                                    },
                                },
                            },
                        ],
                        categories: [
                            {
                                $group: {
                                    _id: { $ifNull: ['$category', 'Other'] },
                                    videos: { $sum: 1 },
                                    views: { $sum: '$views' },
                                },
                            },
                            { $sort: { views: -1 } },
                            { $limit: 6 },
                            { $project: { _id: 0, category: '$_id', videos: 1, views: 1 } },
                        ],
                    },
                },
            ]),
        ]);

        const activities = videoIds.length > 0
            ? await UserActivity.aggregate([
                {
                    $match: {
                        videoId: { $in: videoIds },
                        createdAt: { $gte: periodStart },
                    },
                },
                {
                    $group: {
                        _id: {
                            date: { $dateToString: { format: '%Y-%m-%d', date: '$createdAt', timezone: 'UTC' } },
                            type: '$interactionType',
                        },
                        count: { $sum: 1 },
                    },
                },
            ])
            : [];

        const activityByDate = new Map();
        const periodActivity = { viewInteractions: 0, likeInteractions: 0 };
        for (const activity of activities) {
            const dayData = activityByDate.get(activity._id.date) || {
                viewInteractions: 0,
                likeInteractions: 0,
            };
            if (activity._id.type === 'view') {
                dayData.viewInteractions = activity.count;
                periodActivity.viewInteractions += activity.count;
            }
            if (activity._id.type === 'like') {
                dayData.likeInteractions = activity.count;
                periodActivity.likeInteractions += activity.count;
            }
            activityByDate.set(activity._id.date, dayData);
        }

        const timeSeries = Array.from({ length: days }, (_, index) => {
            const date = new Date(periodStart);
            date.setUTCDate(date.getUTCDate() + index);
            const dateKey = date.toISOString().slice(0, 10);
            return {
                date: date.toLocaleDateString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    timeZone: 'UTC',
                }),
                ...activityByDate.get(dateKey),
            };
        }).map((day) => ({
            ...day,
            viewInteractions: day.viewInteractions || 0,
            likeInteractions: day.likeInteractions || 0,
        }));

        const totals = videoRollups?.totals?.[0] || { videos: 0, views: 0, likes: 0 };
        const comments = videoIds.length > 0
            ? await Comment.countDocuments({ video: { $in: videoIds } })
            : 0;

        const analytics = {
            lifetimeStats: {
                views: totals.views,
                likes: totals.likes,
                comments,
                videos: totals.videos,
                subscribers: user?.subscribersCount || 0,
            },
            period: days,
            periodActivity,
            timeSeries,
            topVideos: videoRollups?.topVideos || [],
            categories: videoRollups?.categories || [],
        };

        await redis.set(cacheKey, JSON.stringify(analytics), { ex: 60 });
        return NextResponse.json(analytics);

    } catch (error) {
        console.error('Error fetching creator analytics:', error);
        return NextResponse.json({ message: 'Server error' }, { status: 500 });
    }
}
