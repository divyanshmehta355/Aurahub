import axios from 'axios';
import mongoose from 'mongoose';
import { NextResponse } from 'next/server';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import Comment from '@/models/Comment';
import CATEGORIES from '@/constants/categories';
import { getServerSession } from "next-auth/next";
import { authOptions } from "@/app/api/auth/[...nextauth]/route";
import redis from '@/lib/redis';

const AURA_API_BASE_URL = "https://aurahub-api-hono.ashwathama249.workers.dev";

export async function GET(request, { params }) {
  try {
    const { id } = await params;
    if (!mongoose.Types.ObjectId.isValid(id)) {
      return NextResponse.json({ message: "Invalid video ID format." }, { status: 400 });
    }

    const cacheKey = `video:${id}`;

    const cachedVideo = await redis.get(cacheKey);
    if (cachedVideo) {
      const parsedVideo = typeof cachedVideo === 'string' ? JSON.parse(cachedVideo) : cachedVideo;
      if (parsedVideo && typeof parsedVideo === 'object') {
        const session = await getServerSession(authOptions);
        const user = session?.user;
        const isLiked = Boolean(
          user &&
          Array.isArray(parsedVideo.likes) &&
          parsedVideo.likes.some((likeId) => likeId.toString() === user.id.toString())
        );
        return NextResponse.json({ ...parsedVideo, isLiked });
      }
    }

    await dbConnect();
    const video = await Video.findById(id)
      .populate('uploader', 'username avatar')
      .lean();
    if (!video) {
      return NextResponse.json({ message: 'Video not found' }, { status: 404 });
    }

    const session = await getServerSession(authOptions);
    const user = session?.user;

    const uploaderId = video.uploader?._id?.toString() || video.uploader?.toString();
    if (video.visibility === 'private' && uploaderId !== user?.id) {
      return NextResponse.json({ message: 'This video is private' }, { status: 403 });
    }

    const commentCount = await Comment.countDocuments({ video: id });
    const videoObject = {
      ...video,
      likesCount: video.likes?.length || 0,
      commentCount,
    };

    await redis.set(cacheKey, JSON.stringify(videoObject), { ex: 3600 });

    const isLiked = Boolean(
      user &&
      Array.isArray(videoObject.likes) &&
      videoObject.likes.some((likeId) => likeId.toString() === user.id.toString())
    );

    return NextResponse.json({ ...videoObject, isLiked });
  } catch (error) {
    console.error("Error fetching video by ID:", error);
    return NextResponse.json({ message: "Server error" }, { status: 500 });
  }
}

export async function PUT(request, { params }) {
  await dbConnect();
  try {
    const session = await getServerSession(authOptions);
    const user = session?.user;
    if (!user) {
      return NextResponse.json({ message: "Unauthorized" }, { status: 401 });
    }

    const { id } = await params;
    const video = await Video.findById(id);
    if (!video) {
      return NextResponse.json({ message: "Video not found" }, { status: 404 });
    }

    if (video.uploader.toString() !== user.id) {
      return NextResponse.json({ message: "User not authorized to edit this video" }, { status: 403 });
    }

    const body = await request.json();
    const { title, description, visibility, tags, category } = body;
    if (title) video.title = title;
    if (description !== undefined) video.description = description;
    if (visibility) video.visibility = visibility;
    if (category !== undefined) {
      if (!CATEGORIES.includes(category)) {
        return NextResponse.json({ message: 'Invalid video category' }, { status: 400 });
      }
      video.category = category;
    }
    if (tags !== undefined) {
      if (!Array.isArray(tags) || tags.some((tag) => typeof tag !== 'string')) {
        return NextResponse.json({ message: 'Tags must be an array of strings' }, { status: 400 });
      }
      video.tags = [...new Set(tags.map((tag) => tag.trim()).filter(Boolean))];
    }

    const updatedVideo = await video.save();
    await redis.invalidateVideo({
      id,
      uploaderUsername: user.name || user.username,
      fileId: video.fileId,
    });
    return NextResponse.json(updatedVideo);
  } catch (error) {
    console.error("Error updating video:", error);
    return NextResponse.json({ message: "Server error while updating video" }, { status: 500 });
  }
}

export async function DELETE(request, { params }) {
  await dbConnect();
  try {
    const session = await getServerSession(authOptions);
    const user = session?.user;
    if (!user) {
      return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
    }

    const { id } = await params;

    const video = await Video.findById(id);
    if (!video) {
      return NextResponse.json({ message: 'Video not found' }, { status: 404 });
    }

    if (video.uploader.toString() !== user.id) {
      return NextResponse.json({ message: 'User not authorized to delete this video' }, { status: 403 });
    }

    try {
      const response = await axios.delete(`${AURA_API_BASE_URL}/fs/files/delete/${video.fileId}`, {
        headers: { accept: '*/*' },
      });
      if (response.data?.success !== true) {
        throw new Error('AuraHub API did not confirm the file deletion');
      }
      console.log(`Successfully deleted file ${video.fileId} from Streamtape.`);
    } catch (auraError) {
      console.error(`Failed to delete file ${video.fileId} from Streamtape:`, auraError.message);
      return NextResponse.json(
        { message: 'Failed to delete video from Streamtape. The video was not removed.' },
        { status: 502 }
      );
    }

    await Video.deleteOne({ _id: id });
    await Comment.deleteMany({ video: id });
    await redis.invalidateVideoCaches(id);

    return NextResponse.json({ message: 'Video deleted successfully' });
  } catch (error) {
    console.error("Error deleting video:", error);
    return NextResponse.json({ message: 'Server error while deleting video' }, { status: 500 });
  }
}