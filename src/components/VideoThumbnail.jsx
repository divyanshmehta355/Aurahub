"use client";

import React, { useState, useEffect } from 'react';
import API from '@/lib/api';
import Image from 'next/image';
import { getFallbackThumbnailUrl } from '@/lib/thumbnailSvg';

const VideoThumbnail = ({ videoId, altText, title, category }) => {
    const videoTitle = title || altText;
    const fallbackUrl = getFallbackThumbnailUrl(videoId, videoTitle, category);
    const [imageUrl, setImageUrl] = useState(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        let isMounted = true;
        const fetchThumbnail = async () => {
            if (!videoId) {
                if (isMounted) {
                    setImageUrl(fallbackUrl);
                    setIsLoading(false);
                }
                return;
            }
            try {
                const response = await API.get(`/videos/${videoId}/thumbnail`);
                if (isMounted) {
                    setImageUrl(response.data.thumbnailUrl);
                }

            } catch (error) {
                console.error(`Could not fetch thumbnail for ${videoId}, using dynamic fallback.`);
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
    }, [videoId, fallbackUrl]);

    if (isLoading) {
        return <div className="w-full h-full bg-gray-300 dark:bg-slate-800 animate-pulse rounded-lg"></div>;
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