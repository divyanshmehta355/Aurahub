import { NextResponse } from 'next/server';
import axios from 'axios';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import Comment from '@/models/Comment';
import { getServerSession } from "next-auth/next";
import { authOptions } from "@/app/api/auth/[...nextauth]/route";
import mongoose from 'mongoose';
import redis from '@/lib/redis';

const AURA_API_BASE_URL = "https://aurahub-api-hono.ashwathama249.workers.dev";

// Bulk Update Visibility
export async function PUT(request) {
    await dbConnect();
    try {
        const session = await getServerSession(authOptions);
        if (!session || !session.user) {
            return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
        }

        const body = await request.json();
        const { videoIds, visibility } = body;

        if (!Array.isArray(videoIds) || videoIds.length === 0) {
            return NextResponse.json({ message: 'No video IDs provided' }, { status: 400 });
        }

        if (!['public', 'unlisted', 'private'].includes(visibility)) {
            return NextResponse.json({ message: 'Invalid visibility state' }, { status: 400 });
        }

        // Validate ObjectIds
        const validIds = videoIds.filter(id => mongoose.Types.ObjectId.isValid(id));
        if (validIds.length === 0) {
            return NextResponse.json({ message: 'No valid video IDs provided' }, { status: 400 });
        }

        // Ensure user only updates their own videos
        const result = await Video.updateMany(
            { _id: { $in: validIds }, uploader: session.user.id },
            { $set: { visibility } }
        );

        await redis.invalidateVideoCaches(validIds);
        if (session.user.name || session.user.username) {
            await redis.invalidateProfile(session.user.name || session.user.username);
        }

        return NextResponse.json({ 
            message: 'Bulk visibility update successful', 
            modifiedCount: result.modifiedCount 
        });

    } catch (error) {
        console.error('Error in bulk update API:', error);
        return NextResponse.json({ message: 'Internal Server Error' }, { status: 500 });
    }
}

// Bulk Delete
export async function DELETE(request) {
    await dbConnect();
    try {
        const session = await getServerSession(authOptions);
        if (!session || !session.user) {
            return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
        }

        const body = await request.json();
        const { videoIds } = body;

        if (!Array.isArray(videoIds) || videoIds.length === 0) {
            return NextResponse.json({ message: 'No video IDs provided' }, { status: 400 });
        }

        // Validate ObjectIds
        const validIds = videoIds.filter(id => mongoose.Types.ObjectId.isValid(id));
        if (validIds.length === 0) {
            return NextResponse.json({ message: 'No valid video IDs provided' }, { status: 400 });
        }

        const videos = await Video.find(
            { _id: { $in: validIds }, uploader: session.user.id },
            { _id: 1, fileId: 1 }
        ).lean();

        const deletionResults = await Promise.all(videos.map(async (video) => {
            try {
                const response = await axios.delete(
                    `${AURA_API_BASE_URL}/fs/files/delete/${video.fileId}`,
                    { headers: { accept: '*/*' } }
                );
                if (response.data?.success !== true) {
                    throw new Error('AuraHub API did not confirm the file deletion');
                }
                return { id: video._id.toString(), success: true };
            } catch (error) {
                console.error(`Failed to delete file ${video.fileId} from Streamtape:`, error.message);
                return { id: video._id.toString(), success: false };
            }
        }));

        const deletedIds = deletionResults
            .filter((result) => result.success)
            .map((result) => result.id);
        let deletedCount = 0;

        if (deletedIds.length > 0) {
            const result = await Video.deleteMany(
                { _id: { $in: deletedIds }, uploader: session.user.id }
            );
            deletedCount = result.deletedCount;
            await Comment.deleteMany({ video: { $in: deletedIds } });
            await redis.invalidateVideoCaches(deletedIds);
        }

        const deletedIdSet = new Set(deletedIds);
        const failedIds = videoIds
            .filter((id) => typeof id === 'string')
            .filter((id) => !deletedIdSet.has(id));

        if (deletedCount > 0 && (session.user.name || session.user.username)) {
            await redis.invalidateProfile(session.user.name || session.user.username);
        }

        return NextResponse.json({ 
            message: failedIds.length > 0 ? 'Some videos could not be deleted' : 'Bulk deletion successful',
            deletedCount,
            deletedIds,
            failedIds,
        });

    } catch (error) {
        console.error('Error in bulk delete API:', error);
        return NextResponse.json({ message: 'Internal Server Error' }, { status: 500 });
    }
}
