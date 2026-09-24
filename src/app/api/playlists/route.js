import { NextResponse } from 'next/server';
import dbConnect from '@/lib/dbConnect';
import Playlist from '@/models/Playlist';
import { getServerSession } from "next-auth/next";
import { authOptions } from "@/app/api/auth/[...nextauth]/route";

export async function GET(request) {
    await dbConnect();
    try {
        const session = await getServerSession(authOptions);
        if (!session?.user) {
            return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
        }

        const playlists = await Playlist.find({ owner: session.user.id }).select('title videos');
        return NextResponse.json(playlists);
    } catch (error) {
        console.error("Error fetching playlists:", error);
        return NextResponse.json({ message: 'Server error' }, { status: 500 });
    }
}

export async function POST(request) {
    await dbConnect();
    try {
        const session = await getServerSession(authOptions);
        if (!session?.user) {
            return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
        }

        const { title, description, isPublic } = await request.json();
        if (!title) {
            return NextResponse.json({ message: 'Title is required' }, { status: 400 });
        }

        const newPlaylist = new Playlist({
            title,
            description,
            owner: session.user.id,
            isPublic,
        });
        await newPlaylist.save();
        return NextResponse.json(newPlaylist, { status: 201 });
    } catch (error) {
        console.error("Error creating playlist:", error);
        return NextResponse.json({ message: 'Server error' }, { status: 500 });
    }
}

export async function PUT(request) {
    await dbConnect();
    try {
        const session = await getServerSession(authOptions);
        if (!session?.user) {
            return NextResponse.json({ message: 'Unauthorized' }, { status: 401 });
        }

        const { playlistId, videoId, title, description, newVideoOrder } = await request.json();
        const playlist = await Playlist.findById(playlistId);

        if (!playlist) {
            return NextResponse.json({ message: 'Playlist not found' }, { status: 404 });
        }
        if (playlist.owner.toString() !== session.user.id) {
            return NextResponse.json({ message: 'User not authorized to edit this playlist' }, { status: 403 });
        }

        const updateDoc = {};
        if (title) updateDoc.title = title;
        if (description !== undefined) updateDoc.description = description;

        if (newVideoOrder) {
            updateDoc.videos = newVideoOrder;
            const updated = await Playlist.findOneAndUpdate(
                { _id: playlistId, owner: session.user.id },
                { $set: updateDoc },
                { new: true }
            );
            return NextResponse.json(updated);
        }

        if (videoId) {
            const hasVideo = playlist.videos.some(v => v.toString() === videoId.toString());
            const updateOps = hasVideo ? { $pull: { videos: videoId } } : { $addToSet: { videos: videoId } };
            if (Object.keys(updateDoc).length > 0) {
                updateOps.$set = updateDoc;
            }
            const updated = await Playlist.findOneAndUpdate(
                { _id: playlistId, owner: session.user.id },
                updateOps,
                { new: true }
            );
            return NextResponse.json(updated);
        }

        if (Object.keys(updateDoc).length > 0) {
            const updated = await Playlist.findOneAndUpdate(
                { _id: playlistId, owner: session.user.id },
                { $set: updateDoc },
                { new: true }
            );
            return NextResponse.json(updated);
        }

        return NextResponse.json(playlist);
    } catch (error) {
        console.error("Error updating playlist:", error);
        return NextResponse.json({ message: 'Server error' }, { status: 500 });
    }
}