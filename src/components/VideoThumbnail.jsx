"use client";

import React, { useState, useEffect } from 'react';
import API from '@/lib/api';
import Image from 'next/image';
import { getFallbackThumbnailUrl } from '@/lib/thumbnailSvg';

const VideoThumbnail = ({ videoId, altText, title, category, thumbnailUrl }) => {
    const videoTitle = title || altText;
    const fallbackUrl = getFallbackThumbnailUrl(videoId, videoTitle, category);

    // If thumbnail URL is already provided by parent, render immediately with 0 delay/waterfalls!
    const [imageUrl, setImageUrl] = useState(thumbnailUrl || fallbackUrl);
    const [isLoading, setIsLoading] = useState(!thumbnailUrl && !videoId);

    useEffect(() => {
        // If thumbnailUrl was passed from parent, use it directly without making an HTTP request
        if (thumbnailUrl) {
            setImageUrl(thumbnailUrl);
            setIsLoading(false);
            return;
        }

        // If no videoId, use fallback immediately
        if (!videoId) {
            setImageUrl(fallbackUrl);
            setIsLoading(false);
            return;
        }

        let isMounted = true;
        const fetchThumbnail = async () => {
            try {
                const response = await API.get(`/videos/${videoId}/thumbnail`);
                if (isMounted && response.data?.thumbnailUrl) {
                    setImageUrl(response.data.thumbnailUrl);
                }
            } catch (error) {
                if (isMounted) {
                    setImageUrl(fallbackUrl);
                }
            } finally {
                if (isMounted) {
                    setIsLoading(false);
                }
            }
        };

        fetchThumbnail();

        return () => {
            isMounted = false;
        };
    }, [videoId, thumbnailUrl, fallbackUrl]);

    if (isLoading) {
        return <div className="w-full h-full bg-gray-200 dark:bg-slate-800 animate-pulse rounded-lg" />;
    }

    return (
        <Image
            src={imageUrl || fallbackUrl}
            alt={altText || videoTitle || 'Video Thumbnail'}
            width={500}
            height={300}
            unoptimized
            onError={() => setImageUrl(fallbackUrl)}
            className="w-full h-full object-cover rounded-lg"
        />
    );
};

export default VideoThumbnail;