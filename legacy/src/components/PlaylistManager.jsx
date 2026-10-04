"use client";

import React, { useState, useEffect } from 'react';
import API from '@/lib/api';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { toast } from 'react-toastify';
import Link from 'next/link';
import VideoThumbnail from './VideoThumbnail';
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

const SortableVideoItem = ({ video, onRemove }) => {
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
    } = useSortable({ id: video._id });

    const style = {
        transform: CSS.Transform.toString(transform),
        transition,
    };

    return (
        <li
            ref={setNodeRef}
            style={style}
            className="flex items-center space-x-3 p-3 bg-card rounded-xl border border-border hover:shadow-sm transition-shadow group"
        >
            <span {...attributes} {...listeners} className="text-muted-foreground hover:text-foreground cursor-grab touch-none p-1 transition-colors">☰</span>
            <div className="w-24 h-14 object-cover rounded flex-shrink-0">
                <VideoThumbnail
                    videoId={video._id}
                    thumbnailUrl={video.thumbnailUrl}
                    title={video.title}
                    category={video.category}
                    altText={video.title}
                />
            </div>
            <div className="flex-grow w-0">
                <Link href={`/video/${video._id}`} className="font-semibold text-foreground hover:text-primary truncate text-sm transition-colors block">
                    {video.title}
                </Link>
                <p className="text-xs text-muted-foreground truncate mt-0.5">{video.uploader.username}</p>
            </div>
            <Button 
                variant="destructive" 
                size="sm"
                onClick={onRemove} 
                className="opacity-0 group-hover:opacity-100 transition-opacity"
            >
                Remove
            </Button>
        </li>
    );
};

const PlaylistManager = ({ playlistId }) => {
    const [playlist, setPlaylist] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchPlaylistDetails = async () => {
            if (!playlistId) return;
            try {
                setLoading(true);
                const response = await API.get(`/playlist/${playlistId}`);
                setPlaylist(response.data);
            } catch (error) {
                console.error("Failed to fetch playlist details", error);
                toast.error("Could not load playlist details.");
            } finally {
                setLoading(false);
            }
        };
        fetchPlaylistDetails();
    }, [playlistId]);

    const sensors = useSensors(
        useSensor(PointerSensor),
        useSensor(KeyboardSensor, {
          coordinateGetter: sortableKeyboardCoordinates,
        })
    );

    const handleDragEnd = (event) => {
        const { active, over } = event;

        if (active && over && active.id !== over.id) {
            const oldIndex = playlist.videos.findIndex(v => v._id === active.id);
            const newIndex = playlist.videos.findIndex(v => v._id === over.id);
            const newVideoOrder = arrayMove(playlist.videos, oldIndex, newIndex);
            
            // Optimistically update the UI
            setPlaylist(prev => ({...prev, videos: newVideoOrder}));
            
            // Send the new order of video IDs to the backend
            const newVideoOrderIds = newVideoOrder.map(video => video._id);
            API.put('/playlists/my-playlists', { playlistId, newVideoOrder: newVideoOrderIds })
                .then(() => toast.success("Playlist order saved!"))
                .catch(() => toast.error("Failed to save new order."));
        }
    };

    const handleRemoveVideo = async (videoId) => {
        try {
            await API.put('/playlists', { playlistId, videoId }); // The PUT route toggles the video
            // Optimistically update the UI to remove the video
            setPlaylist(prev => ({...prev, videos: prev.videos.filter(v => v._id !== videoId) }));
            toast.success("Video removed from playlist.");
        } catch (error) { 
            toast.error("Failed to remove video."); 
        }
    };

    if (loading) {
        return (
            <Card className="p-6 sm:p-8 rounded-2xl shadow-sm space-y-4">
                <Skeleton className="h-8 w-1/2" />
                <Skeleton className="h-4 w-1/4 mb-6" />
                <div className="space-y-3">
                    <Skeleton className="h-20 w-full rounded-xl" />
                    <Skeleton className="h-20 w-full rounded-xl" />
                    <Skeleton className="h-20 w-full rounded-xl" />
                </div>
            </Card>
        );
    }
    
    if (!playlist) return <div className="p-6 text-muted-foreground text-center font-medium">Playlist not found.</div>;

    return (
        <Card className="p-6 sm:p-8 rounded-2xl shadow-sm transition-colors duration-300">
            <h2 className="text-2xl font-bold text-foreground">{playlist.title}</h2>
            <p className="text-sm text-muted-foreground mt-1 mb-6 font-medium">{playlist.videos.length} videos</p>

            <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleDragEnd}
            >
                <SortableContext
                    items={playlist.videos.map(v => v._id)}
                    strategy={verticalListSortingStrategy}
                >
                    <ul className="mt-4 space-y-2">
                        {playlist.videos.map((video) => (
                            <SortableVideoItem 
                                key={video._id} 
                                video={video} 
                                onRemove={() => handleRemoveVideo(video._id)} 
                            />
                        ))}
                    </ul>
                </SortableContext>
            </DndContext>
            {playlist.videos.length === 0 && <p className="mt-8 text-sm text-center text-muted-foreground bg-muted p-6 rounded-xl border border-dashed border-border">This playlist is empty. Add videos by clicking the "Watch Later" button on a video card or using the Save modal.</p>}
        </Card>
    );
};

export default PlaylistManager;